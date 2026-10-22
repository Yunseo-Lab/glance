package dashboard

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"
	"time"
)

type LocationResult struct {
	Name      string  `json:"name"`
	Country   string  `json:"country"`
	Region    string  `json:"region"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Timezone  string  `json:"timezone"`
}

func (s *DashboardService) SearchLocations(ctx context.Context, query string) ([]LocationResult, error) {
	query = strings.TrimSpace(query)
	if n := len([]rune(query)); n < 2 || n > 80 {
		return nil, ErrInvalidInput
	}
	now := time.Now()
	s.geoMu.Lock()
	valid := s.geoQueries[:0]
	for _, at := range s.geoQueries {
		if at.After(now.Add(-time.Minute)) {
			valid = append(valid, at)
		}
	}
	s.geoQueries = valid
	if len(valid) >= 10 {
		until := valid[0].Add(time.Minute)
		s.geoMu.Unlock()
		return nil, &CooldownError{Until: until}
	}
	s.geoQueries = append(s.geoQueries, now)
	s.geoMu.Unlock()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	data, err := s.providers.get(ctx, "https://geocoding-api.open-meteo.com/v1/search?"+url.Values{"name": {query}, "count": {"5"}, "language": {"ko"}, "format": {"json"}}.Encode())
	if err != nil {
		return nil, &ProviderError{Code: "unavailable"}
	}
	var body struct {
		Results []struct {
			Name      string   `json:"name"`
			Country   string   `json:"country"`
			Region    string   `json:"admin1"`
			Latitude  *float64 `json:"latitude"`
			Longitude *float64 `json:"longitude"`
			Timezone  string   `json:"timezone"`
		} `json:"results"`
	}
	if err := json.Unmarshal(data, &body); err != nil {
		return nil, &ProviderError{Code: "invalid_response"}
	}
	result := []LocationResult{}
	for _, place := range body.Results {
		if place.Latitude == nil || place.Longitude == nil || *place.Latitude < -90 || *place.Latitude > 90 || *place.Longitude < -180 || *place.Longitude > 180 || place.Name == "" {
			continue
		}
		if _, err := time.LoadLocation(place.Timezone); err != nil {
			continue
		}
		result = append(result, LocationResult{Name: plainText(place.Name, 100), Country: plainText(place.Country, 100), Region: plainText(place.Region, 100), Latitude: *place.Latitude, Longitude: *place.Longitude, Timezone: place.Timezone})
		if len(result) == 5 {
			break
		}
	}
	return result, nil
}
