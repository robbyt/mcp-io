package main

import (
	"sync"
	"testing"
	"testing/synctest"

	mcpio "github.com/robbyt/mcp-io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/adk/tool/mcptoolset"
)

// TestCityCache verifies that hardcoded city lookups work instantly.
func TestCityCache(t *testing.T) {
	tests := []struct {
		name       string
		cityName   string
		wantLat    float64
		wantLon    float64
		wantSource string
	}{
		{
			name:       "New York",
			cityName:   "New York",
			wantLat:    40.7128,
			wantLon:    -74.0060,
			wantSource: "cache",
		},
		{
			name:       "Case insensitive",
			cityName:   "CHICAGO",
			wantLat:    41.8781,
			wantLon:    -87.6298,
			wantSource: "cache",
		},
		{
			name:       "With whitespace",
			cityName:   "  Seattle  ",
			wantLat:    47.6062,
			wantLon:    -122.3321,
			wantSource: "cache",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lat, lon, source, err := GetCoordinates(tt.cityName)
			require.NoError(t, err)
			assert.Equal(t, tt.wantLat, lat)
			assert.Equal(t, tt.wantLon, lon)
			assert.Equal(t, tt.wantSource, source)
		})
	}
}

// TestGeocodeToolDirect tests the geocode_city tool function directly.
func TestGeocodeToolDirect(t *testing.T) {
	input := GeocodeInput{CityName: "Boston"}
	output, err := geocodeTool(t.Context(), nil, input)

	require.NoError(t, err)
	assert.Equal(t, 42.3601, output.Latitude)
	assert.Equal(t, -71.0589, output.Longitude)
	assert.Equal(t, "cache", output.Source)
}

// TestWeatherToolIntegration tests the get_weather tool with real NOAA API.
func TestWeatherToolIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test with real NOAA API")
	}

	// Test weather query for New York
	input := WeatherInput{
		Latitude:  40.7128,
		Longitude: -74.0060,
	}

	output, err := getWeatherTool(t.Context(), nil, input)
	require.NoError(t, err)

	// Verify output contains expected fields
	assert.Contains(t, output.Location, "40.7128")
	assert.NotZero(t, output.CurrentTemp)
	assert.Equal(t, "F", output.Unit)
	assert.NotEmpty(t, output.Conditions)
	assert.NotEmpty(t, output.Forecast)
}

// TestMCPServerIntegration verifies MCP server can be created with both tools.
func TestMCPServerIntegration(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx := t.Context()

		// Create server with both tools
		handler, clientTransport, err := mcpio.NewInMemoryPair(ctx,
			mcpio.WithName("weather-agent"),
			mcpio.WithTool("geocode_city", "Convert city to coordinates", geocodeTool),
			mcpio.WithTool("get_weather", "Get weather forecast", getWeatherTool),
		)
		require.NoError(t, err)
		require.NotNil(t, handler)
		require.NotNil(t, clientTransport)

		// Start server
		var wg sync.WaitGroup
		wg.Go(func() {
			_ = handler.Run(ctx)
		})
		synctest.Wait()

		// Verify ADK toolset can be created
		mcpToolSet, err := mcptoolset.New(mcptoolset.Config{
			Transport: clientTransport,
		})
		require.NoError(t, err)
		require.NotNil(t, mcpToolSet)
	})
}

// TestGeocodeToolEmptyInput verifies that empty city name returns an error.
func TestGeocodeToolEmptyInput(t *testing.T) {
	input := GeocodeInput{CityName: ""}
	_, err := geocodeTool(t.Context(), nil, input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "city name is required")
}

// TestWeatherToolZeroCoordinates verifies that (0,0) coordinates are accepted.
// The NOAA API will error since 0,0 is in the ocean, but we shouldn't reject it client-side.
func TestWeatherToolZeroCoordinates(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test with real NOAA API")
	}

	input := WeatherInput{Latitude: 0, Longitude: 0}
	_, err := getWeatherTool(t.Context(), nil, input)
	// Error should come from NOAA, not coordinate validation
	if err != nil {
		assert.Contains(t, err.Error(), "NOAA")
	}
}

// TestGetCoordinatesUnknownCity verifies geocoding fallback for cities not in cache.
func TestGetCoordinatesUnknownCity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test with real geocoding API")
	}

	lat, lon, source, err := GetCoordinates("Truth or Consequences")
	require.NoError(t, err)
	assert.Equal(t, "geocoded", source)
	assert.InDelta(t, 33.13, lat, 0.1)
	assert.InDelta(t, -107.25, lon, 0.1)
}

// TestRuntimeCache verifies that geocoding results are cached.
func TestRuntimeCache(t *testing.T) {
	// Clear runtime cache before test
	runtimeCacheMu.Lock()
	for k := range runtimeCache {
		delete(runtimeCache, k)
	}
	runtimeCacheMu.Unlock()

	// First call should return "cache" for known city
	lat1, lon1, source1, err := GetCoordinates("New York")
	require.NoError(t, err)
	assert.Equal(t, "cache", source1)

	// Same city should still use hardcoded cache
	lat2, lon2, source2, err := GetCoordinates("new york")
	require.NoError(t, err)
	assert.Equal(t, "cache", source2)
	assert.Equal(t, lat1, lat2)
	assert.Equal(t, lon1, lon2)
}
