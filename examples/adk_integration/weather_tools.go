package main

import (
	"context"
	"fmt"
	"strconv"

	"github.com/icodealot/noaa"
	mcpio "github.com/robbyt/mcp-io"
)

// GeocodeInput defines the input schema for the geocode_city tool.
type GeocodeInput struct {
	CityName string `json:"city_name" jsonschema:"US city name"`
}

// GeocodeOutput defines the output schema for the geocode_city tool.
type GeocodeOutput struct {
	Latitude  float64 `json:"latitude" jsonschema:"Latitude coordinate"`
	Longitude float64 `json:"longitude" jsonschema:"Longitude coordinate"`
	Source    string  `json:"source" jsonschema:"Source of coordinates"`
}

// geocodeTool converts a city name to geographic coordinates.
// It uses a cache-first approach with fallback to OpenStreetMap geocoding.
func geocodeTool(_ context.Context, _ mcpio.RequestContext, input GeocodeInput) (GeocodeOutput, error) {
	if input.CityName == "" {
		return GeocodeOutput{}, fmt.Errorf("city name is required but was empty")
	}

	lat, lon, source, err := GetCoordinates(input.CityName)
	if err != nil {
		return GeocodeOutput{}, fmt.Errorf("failed to geocode '%s': %w", input.CityName, err)
	}

	return GeocodeOutput{
		Latitude:  lat,
		Longitude: lon,
		Source:    source,
	}, nil
}

// WeatherInput defines the input schema for the get_weather tool.
type WeatherInput struct {
	Latitude  float64 `json:"latitude" jsonschema:"Latitude coordinate"`
	Longitude float64 `json:"longitude" jsonschema:"Longitude coordinate"`
}

// ForecastPeriod represents a single forecast time period.
type ForecastPeriod struct {
	Name        string  `json:"name" jsonschema:"Period name"`
	Temperature float64 `json:"temperature" jsonschema:"Temperature value"`
	Unit        string  `json:"unit" jsonschema:"Temperature unit"`
	Windspeed   string  `json:"windspeed" jsonschema:"Wind speed description"`
	Description string  `json:"description" jsonschema:"Detailed weather description"`
}

// WeatherOutput defines the output schema for the get_weather tool.
type WeatherOutput struct {
	Location    string           `json:"location" jsonschema:"Location description"`
	CurrentTemp float64          `json:"current_temp" jsonschema:"Current temperature"`
	Unit        string           `json:"unit" jsonschema:"Temperature unit"`
	Conditions  string           `json:"conditions" jsonschema:"Current conditions summary"`
	Forecast    []ForecastPeriod `json:"forecast" jsonschema:"Extended forecast periods"`
}

// getWeatherTool fetches weather forecast data from NOAA for the given coordinates.
func getWeatherTool(_ context.Context, _ mcpio.RequestContext, input WeatherInput) (WeatherOutput, error) {
	latStr := strconv.FormatFloat(input.Latitude, 'f', 4, 64)
	lonStr := strconv.FormatFloat(input.Longitude, 'f', 4, 64)

	forecast, err := noaa.Forecast(latStr, lonStr)
	if err != nil {
		return WeatherOutput{}, fmt.Errorf("NOAA API error for coords (%s, %s): %w", latStr, lonStr, err)
	}

	if len(forecast.Periods) == 0 {
		return WeatherOutput{}, fmt.Errorf("no forecast data available for coords (%s, %s)", latStr, lonStr)
	}

	currentPeriod := forecast.Periods[0]

	periods := make([]ForecastPeriod, 0, len(forecast.Periods))
	for _, p := range forecast.Periods {
		periods = append(periods, ForecastPeriod{
			Name:        p.Name,
			Temperature: p.Temperature,
			Unit:        p.TemperatureUnit,
			Windspeed:   p.WindSpeed,
			Description: p.Details,
		})
	}

	return WeatherOutput{
		Location:    fmt.Sprintf("%.4f, %.4f", input.Latitude, input.Longitude),
		CurrentTemp: currentPeriod.Temperature,
		Unit:        currentPeriod.TemperatureUnit,
		Conditions:  currentPeriod.Summary,
		Forecast:    periods,
	}, nil
}
