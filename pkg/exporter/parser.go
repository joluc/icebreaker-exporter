package exporter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/joluc/icebreaker-exporter/pkg/config"
	"github.com/joluc/icebreaker-exporter/pkg/models"
)

func fetchPositions(ctx context.Context, client *http.Client, cfg config.Config) ([]models.IcebreakerPosition, error) {
	reqCtx, cancel := context.WithTimeout(ctx, cfg.RequestTimeout)
	defer cancel()

	vesselsPayload, err := fetchJSON(reqCtx, client, cfg.VesselsURL, cfg.DigitrafficUser)
	if err != nil {
		return nil, fmt.Errorf("fetch vessels: %w", err)
	}
	locationsPayload, err := fetchJSON(reqCtx, client, cfg.LocationsURL, cfg.DigitrafficUser)
	if err != nil {
		return nil, fmt.Errorf("fetch locations: %w", err)
	}

	vessels := ExtractVesselMetadata(vesselsPayload)
	locations := ExtractLocations(locationsPayload)
	positions := SelectIcebreakerPositions(vessels, locations, cfg.TargetNames)

	if len(positions) == 0 {
		return nil, errors.New("no positions found for configured icebreakers")
	}

	return positions, nil
}

func fetchJSON(ctx context.Context, client *http.Client, endpoint, userAgent string) (any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if userAgent != "" {
		req.Header.Set("Digitraffic-User", userAgent)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("status %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	dec := json.NewDecoder(resp.Body)
	dec.UseNumber()
	var payload any
	if err := dec.Decode(&payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func ExtractVesselMetadata(payload any) []models.VesselMetadata {
	byMMSI := map[string]models.VesselMetadata{}
	walkJSON(payload, func(item map[string]any) {
		mmsi := getMMSI(item, "mmsi")
		if mmsi == "" {
			return
		}

		name := getString(item, "name", "vesselName")
		if name == "" {
			return
		}

		// Infer country based on MID (first 3 digits of MMSI)
		country := "Unknown"
		if len(mmsi) >= 3 {
			mid := mmsi[:3]
			switch mid {
			case "230":
				country = "FI"
			case "265", "266":
				country = "SE"
			case "257", "258", "259":
				country = "NO"
			case "219", "220":
				country = "DK"
			}
		}

		byMMSI[mmsi] = models.VesselMetadata{
			Name:    strings.TrimSpace(name),
			MMSI:    mmsi,
			Country: country,
		}
	})

	out := make([]models.VesselMetadata, 0, len(byMMSI))
	for _, meta := range byMMSI {
		out = append(out, meta)
	}
	return out
}

func ExtractLocations(payload any) []models.LocationRecord {
	var out []models.LocationRecord
	walkJSON(payload, func(item map[string]any) {
		if loc, ok := parseGeoJSONLocation(item); ok {
			out = append(out, loc)
			return
		}
		if loc, ok := parseFlatLocation(item); ok {
			out = append(out, loc)
			return
		}
	})
	return out
}

func parseGeoJSONLocation(item map[string]any) (models.LocationRecord, bool) {
	geometry, ok := item["geometry"].(map[string]any)
	if !ok {
		return models.LocationRecord{}, false
	}
	coords, ok := geometry["coordinates"].([]any)
	if !ok || len(coords) < 2 {
		return models.LocationRecord{}, false
	}

	lon, okLon := toFloat64(coords[0])
	lat, okLat := toFloat64(coords[1])
	if !okLon || !okLat {
		return models.LocationRecord{}, false
	}

	properties, _ := item["properties"].(map[string]any)
	mmsi := getMMSI(properties, "mmsi")
	if mmsi == "" {
		mmsi = getMMSI(item, "mmsi")
	}
	if mmsi == "" {
		return models.LocationRecord{}, false
	}

	name := getString(properties, "name", "vesselName")
	if name == "" {
		name = getString(item, "name", "vesselName")
	}

	ts := getTimestamp(properties, "timestamp", "time", "locUpdateTimestamp")
	if ts == 0 {
		ts = getTimestamp(item, "timestamp", "time", "locUpdateTimestamp")
	}

	return models.LocationRecord{
		Name:      strings.TrimSpace(name),
		MMSI:      mmsi,
		Latitude:  lat,
		Longitude: lon,
		Timestamp: ts,
	}, true
}

func parseFlatLocation(item map[string]any) (models.LocationRecord, bool) {
	lat, okLat := getNumber(item, "lat", "latitude")
	lon, okLon := getNumber(item, "lon", "lng", "longitude", "long")
	if !okLat || !okLon {
		return models.LocationRecord{}, false
	}

	mmsi := getMMSI(item, "mmsi")
	if mmsi == "" {
		return models.LocationRecord{}, false
	}

	ts := getTimestamp(item, "timestamp", "time", "locUpdateTimestamp")

	return models.LocationRecord{
		Name:      strings.TrimSpace(getString(item, "name", "vesselName")),
		MMSI:      mmsi,
		Latitude:  lat,
		Longitude: lon,
		Timestamp: ts,
	}, true
}

func SelectIcebreakerPositions(vessels []models.VesselMetadata, locations []models.LocationRecord, targets map[string]struct{}) []models.IcebreakerPosition {
	selectedByMMSI := map[string]models.VesselMetadata{}
	for _, vessel := range vessels {
		if _, ok := targets[config.NormalizeName(vessel.Name)]; !ok {
			continue
		}
		selectedByMMSI[vessel.MMSI] = vessel
	}

	bestByMMSI := map[string]models.IcebreakerPosition{}
	for _, loc := range locations {
		if loc.MMSI == "" {
			continue
		}
		if math.IsNaN(loc.Latitude) || math.IsNaN(loc.Longitude) {
			continue
		}

		vessel, ok := selectedByMMSI[loc.MMSI]
		if !ok {
			locName := strings.TrimSpace(loc.Name)
			if _, nameMatch := targets[config.NormalizeName(locName)]; !nameMatch {
				continue
			}
			vessel = models.VesselMetadata{
				Name:    locName,
				MMSI:    loc.MMSI,
				Country: "Unknown",
			}
			selectedByMMSI[loc.MMSI] = vessel
		}

		candidate := models.IcebreakerPosition{
			Name:      vessel.Name,
			MMSI:      loc.MMSI,
			Country:   vessel.Country,
			Latitude:  loc.Latitude,
			Longitude: loc.Longitude,
			Timestamp: loc.Timestamp,
		}
		current, exists := bestByMMSI[loc.MMSI]
		if !exists || candidate.Timestamp > current.Timestamp {
			bestByMMSI[loc.MMSI] = candidate
		}
	}

	out := make([]models.IcebreakerPosition, 0, len(bestByMMSI))
	for _, pos := range bestByMMSI {
		out = append(out, pos)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Name == out[j].Name {
			return out[i].MMSI < out[j].MMSI
		}
		return out[i].Name < out[j].Name
	})

	return out
}

func walkJSON(node any, fn func(map[string]any)) {
	switch value := node.(type) {
	case map[string]any:
		fn(value)
		for _, child := range value {
			walkJSON(child, fn)
		}
	case []any:
		for _, child := range value {
			walkJSON(child, fn)
		}
	}
}

func getString(item map[string]any, keys ...string) string {
	for _, key := range keys {
		raw, ok := item[key]
		if !ok || raw == nil {
			continue
		}
		switch value := raw.(type) {
		case string:
			if strings.TrimSpace(value) != "" {
				return value
			}
		}
	}
	return ""
}

func getNumber(item map[string]any, keys ...string) (float64, bool) {
	for _, key := range keys {
		raw, ok := item[key]
		if !ok || raw == nil {
			continue
		}
		if value, ok := toFloat64(raw); ok {
			return value, true
		}
	}
	return 0, false
}

func getMMSI(item map[string]any, keys ...string) string {
	if item == nil {
		return ""
	}
	for _, key := range keys {
		raw, ok := item[key]
		if !ok || raw == nil {
			continue
		}
		if mmsi := toMMSI(raw); mmsi != "" {
			return mmsi
		}
	}
	return ""
}

func toMMSI(value any) string {
	switch v := value.(type) {
	case string:
		v = strings.TrimSpace(v)
		if v == "" {
			return ""
		}
		if strings.Contains(v, ".") {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				return strconv.FormatInt(int64(f), 10)
			}
		}
		return v
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return strconv.FormatInt(i, 10)
		}
		if f, err := v.Float64(); err == nil {
			return strconv.FormatInt(int64(f), 10)
		}
	case float64:
		return strconv.FormatInt(int64(v), 10)
	case float32:
		return strconv.FormatInt(int64(v), 10)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	}
	return ""
}

func getTimestamp(item map[string]any, keys ...string) int64 {
	if item == nil {
		return 0
	}
	for _, key := range keys {
		raw, ok := item[key]
		if !ok || raw == nil {
			continue
		}
		if ts := toTimestamp(raw); ts > 0 {
			return ts
		}
	}
	return 0
}

func toTimestamp(value any) int64 {
	switch v := value.(type) {
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return normalizeTimestamp(i)
		}
		if f, err := v.Float64(); err == nil {
			return normalizeTimestamp(int64(f))
		}
	case float64:
		return normalizeTimestamp(int64(v))
	case float32:
		return normalizeTimestamp(int64(v))
	case int:
		return normalizeTimestamp(int64(v))
	case int64:
		return normalizeTimestamp(v)
	case int32:
		return normalizeTimestamp(int64(v))
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return 0
		}
		if parsed, err := strconv.ParseInt(s, 10, 64); err == nil {
			return normalizeTimestamp(parsed)
		}
		if parsed, err := time.Parse(time.RFC3339, s); err == nil {
			return parsed.Unix()
		}
	}
	return 0
}

func normalizeTimestamp(ts int64) int64 {
	// Handle milliseconds and microseconds.
	switch {
	case ts > 1_000_000_000_000_000:
		return ts / 1_000_000
	case ts > 1_000_000_000_000:
		return ts / 1_000
	default:
		return ts
	}
}

func toFloat64(value any) (float64, bool) {
	switch v := value.(type) {
	case json.Number:
		f, err := v.Float64()
		return f, err == nil
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case int32:
		return float64(v), true
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		return f, err == nil
	default:
		return 0, false
	}
}
