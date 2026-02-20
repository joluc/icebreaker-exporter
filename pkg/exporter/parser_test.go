package exporter

import (
	"testing"

	"github.com/joluc/icebreaker-exporter/pkg/models"
)

func TestExtractVesselMetadata(t *testing.T) {
	payload := map[string]any{
		"vessels": []any{
			map[string]any{"name": "Otso", "mmsi": 230124000},
			map[string]any{"name": "Kontio", "mmsi": "230123000"},
		},
	}

	metas := ExtractVesselMetadata(payload)
	if len(metas) != 2 {
		t.Fatalf("expected 2 vessels, got %d", len(metas))
	}
}

func TestExtractLocations(t *testing.T) {
	payload := map[string]any{
		"features": []any{
			map[string]any{
				"properties": map[string]any{
					"mmsi":      230124000,
					"name":      "Otso",
					"timestamp": 1700000000,
				},
				"geometry": map[string]any{
					"coordinates": []any{24.91, 60.17},
				},
			},
			map[string]any{
				"mmsi":      "230123000",
				"name":      "Kontio",
				"lat":       61.3,
				"lon":       21.4,
				"timestamp": 1700000001,
			},
		},
	}

	locations := ExtractLocations(payload)
	if len(locations) != 2 {
		t.Fatalf("expected 2 locations, got %d", len(locations))
	}
}

func TestSelectIcebreakerPositions(t *testing.T) {
	metas := []models.VesselMetadata{
		{Name: "Otso", MMSI: "230124000"},
		{Name: "Kontio", MMSI: "230123000"},
		{Name: "Random", MMSI: "111111111"},
	}
	locations := []models.LocationRecord{
		{Name: "Otso", MMSI: "230124000", Latitude: 60.1, Longitude: 24.9, Timestamp: 100},
		{Name: "Otso", MMSI: "230124000", Latitude: 60.2, Longitude: 25.0, Timestamp: 200},
		{Name: "Kontio", MMSI: "230123000", Latitude: 61.1, Longitude: 21.2, Timestamp: 150},
		{Name: "Random", MMSI: "111111111", Latitude: 10, Longitude: 10, Timestamp: 100},
	}

	targets := map[string]struct{}{
		"OTSO":   {},
		"KONTIO": {},
	}

	positions := SelectIcebreakerPositions(metas, locations, targets)
	if len(positions) != 2 {
		t.Fatalf("expected 2 positions, got %d", len(positions))
	}

	if positions[1].MMSI != "230124000" || positions[1].Timestamp != 200 {
		t.Fatalf("expected latest Otso position to be selected, got %+v", positions[1])
	}
}
