package models

import "time"

type VesselMetadata struct {
	MMSI    string    // Maritime Mobile Service Identity
	Name    string    // Name of the vessel
	Country string    // Inferred country
	Updated time.Time // When the metadata was last updated
}

type LocationRecord struct {
	Name      string
	MMSI      string
	Latitude  float64
	Longitude float64
	Timestamp int64
	// AIS movement fields
	SpeedOverGround  float64 // knots
	CourseOverGround float64 // degrees 0-360
	Heading          float64 // degrees 0-360
	NavigationStatus int     // 0-15
	RateOfTurn       float64 // degrees per minute
}

type IcebreakerPosition struct {
	Name      string
	MMSI      string
	Country   string
	Latitude  float64
	Longitude float64
	Timestamp int64 // Unix timestamp of the location record
	// AIS movement fields
	SpeedOverGround  float64 // knots
	CourseOverGround float64 // degrees 0-360
	Heading          float64 // degrees 0-360
	NavigationStatus int     // 0-15
	RateOfTurn       float64 // degrees per minute
}

type Snapshot struct {
	Positions        []IcebreakerPosition
	LastRefresh      time.Time
	RefreshDuration  time.Duration
	LastRefreshError string
}
