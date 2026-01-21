package main

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	osm "github.com/codingsince1985/geo-golang/openstreetmap"
)

// Coordinates represents a geographic location.
type Coordinates struct {
	Latitude  float64
	Longitude float64
}

// cityCache holds hardcoded coordinates for major US cities.
// This provides fast lookups without external API calls.
var cityCache = map[string]Coordinates{
	"new york":     {40.7128, -74.0060},
	"los angeles":  {34.0522, -118.2437},
	"chicago":      {41.8781, -87.6298},
	"houston":      {29.7604, -95.3698},
	"phoenix":      {33.4484, -112.0740},
	"philadelphia": {39.9526, -75.1652},
	"san antonio":  {29.4241, -98.4936},
	"san diego":    {32.7157, -117.1611},
	"dallas":       {32.7767, -96.7970},
	"austin":       {30.2672, -97.7431},
	"jacksonville": {30.3322, -81.6557},
	"fort worth":   {32.7555, -97.3308},
	"san jose":     {37.3382, -121.8863},
	"columbus":     {39.9612, -82.9988},
	"charlotte":    {35.2271, -80.8431},
	"indianapolis": {39.7684, -86.1581},
	"seattle":      {47.6062, -122.3321},
	"denver":       {39.7392, -104.9903},
	"boston":       {42.3601, -71.0589},
	"portland":     {45.5152, -122.6784},
	"las vegas":    {36.1699, -115.1398},
	"miami":        {25.7617, -80.1918},
	"atlanta":      {33.7490, -84.3880},
	"minneapolis":  {44.9778, -93.2650},
	"detroit":      {42.3314, -83.0458},
}

// runtimeCache stores coordinates discovered via geocoding API during runtime.
// This prevents repeated API calls for the same cities within a session.
var (
	runtimeCache   = make(map[string]Coordinates)
	runtimeCacheMu sync.RWMutex
)

// GetCoordinates returns the latitude and longitude for a given city name.
// It first checks the hardcoded cache, then the runtime cache, and finally
// falls back to the OpenStreetMap geocoding API if needed.
//
// Returns:
//   - lat, lon: The coordinates
//   - source: "cache" if from hardcoded cache, "geocoded" if from API
//   - error: Any error encountered during geocoding
func GetCoordinates(cityName string) (lat, lon float64, source string, err error) {
	normalized := strings.ToLower(strings.TrimSpace(cityName))

	if coords, found := cityCache[normalized]; found {
		return coords.Latitude, coords.Longitude, "cache", nil
	}

	runtimeCacheMu.RLock()
	if coords, found := runtimeCache[normalized]; found {
		runtimeCacheMu.RUnlock()
		return coords.Latitude, coords.Longitude, "runtime-cache", nil
	}
	runtimeCacheMu.RUnlock()

	start := time.Now()
	geocoder := osm.Geocoder()
	location, err := geocoder.Geocode(cityName + ", USA")
	elapsed := time.Since(start)

	if err != nil {
		return 0, 0, "", fmt.Errorf("geocoding failed for '%s': %w", cityName, err)
	}

	if location == nil {
		return 0, 0, "", fmt.Errorf("no location found for '%s'", cityName)
	}

	coords := Coordinates{
		Latitude:  location.Lat,
		Longitude: location.Lng,
	}

	runtimeCacheMu.Lock()
	runtimeCache[normalized] = coords
	runtimeCacheMu.Unlock()

	slog.Info("Geocoded city via OpenStreetMap",
		"city", cityName,
		"latitude", coords.Latitude,
		"longitude", coords.Longitude,
		"elapsed", elapsed,
	)

	return coords.Latitude, coords.Longitude, "geocoded", nil
}
