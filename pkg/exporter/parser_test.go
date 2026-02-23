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
					"sog":       5.2,
					"cog":       45.0,
					"heading":   47,
					"navStat":   0,
					"rot":       2.5,
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
				"sog":       0.0,
				"cog":       0.0,
				"heading":   180,
				"navStat":   5,
				"rot":       0.0,
			},
		},
	}

	locations := ExtractLocations(payload)
	if len(locations) != 2 {
		t.Fatalf("expected 2 locations, got %d", len(locations))
	}

	// Verify first location has movement data
	if locations[0].SpeedOverGround != 5.2 {
		t.Errorf("expected SOG=5.2, got %.2f", locations[0].SpeedOverGround)
	}
	if locations[0].CourseOverGround != 45.0 {
		t.Errorf("expected COG=45.0, got %.1f", locations[0].CourseOverGround)
	}
	if locations[0].Heading != 47 {
		t.Errorf("expected Heading=47, got %.0f", locations[0].Heading)
	}
	if locations[0].NavigationStatus != 0 {
		t.Errorf("expected NavStat=0, got %d", locations[0].NavigationStatus)
	}
	if locations[0].RateOfTurn != 2.5 {
		t.Errorf("expected ROT=2.5, got %.1f", locations[0].RateOfTurn)
	}

	// Verify second location (flat format) has movement data
	if locations[1].NavigationStatus != 5 {
		t.Errorf("expected NavStat=5 for second location, got %d", locations[1].NavigationStatus)
	}
}

func TestSelectIcebreakerPositions(t *testing.T) {
	metas := []models.VesselMetadata{
		{Name: "Otso", MMSI: "230124000"},
		{Name: "Kontio", MMSI: "230123000"},
		{Name: "Random", MMSI: "111111111"},
	}
	locations := []models.LocationRecord{
		{Name: "Otso", MMSI: "230124000", Latitude: 60.1, Longitude: 24.9, Timestamp: 100, SpeedOverGround: 3.5, NavigationStatus: 0},
		{Name: "Otso", MMSI: "230124000", Latitude: 60.2, Longitude: 25.0, Timestamp: 200, SpeedOverGround: 5.2, NavigationStatus: 0},
		{Name: "Kontio", MMSI: "230123000", Latitude: 61.1, Longitude: 21.2, Timestamp: 150, SpeedOverGround: 0.0, NavigationStatus: 5},
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

	// Verify movement fields are propagated
	if positions[1].SpeedOverGround != 5.2 {
		t.Errorf("expected SOG=5.2 for latest Otso position, got %.2f", positions[1].SpeedOverGround)
	}
	if positions[0].NavigationStatus != 5 {
		t.Errorf("expected NavStat=5 for Kontio, got %d", positions[0].NavigationStatus)
	}
}
