package dashboard

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestWeatherResponseSuccessAndMissingData(t *testing.T) {
	source := WeatherSource{Location: "Seoul", Latitude: 37.5665, Longitude: 126.978}
	data := []byte(`{"current":{"temperature_2m":12.5,"apparent_temperature":11.2,"relative_humidity_2m":63,"weather_code":3}}`)
	weather, err := parseWeather(data, source)
	if err != nil {
		t.Fatal(err)
	}
	if weather.Location != "Seoul" || weather.Temperature != 12.5 || weather.FeelsLike == nil || *weather.FeelsLike != 11.2 || weather.Humidity == nil || *weather.Humidity != 63 || weather.Code == nil || *weather.Code != 3 {
		t.Fatalf("unexpected weather: %+v", weather)
	}
	for _, body := range []string{`{}`, `{"current":{}}`, `{"current":null}`, `not json`} {
		if _, err = parseWeather([]byte(body), source); err == nil {
			t.Fatalf("invalid weather accepted: %s", body)
		}
	}
}

func TestWeatherMissingOptionalValuesStayNullAndUnitsAreRequested(t *testing.T) {
	data, err := parseWeather([]byte(`{"current":{"temperature_2m":0,"apparent_temperature":null,"relative_humidity_2m":null,"weather_code":null}}`), WeatherSource{Location: "Zero", Units: "fahrenheit", Timezone: "America/New_York"})
	if err != nil || data.Temperature != 0 || data.FeelsLike != nil || data.Humidity != nil || data.Code != nil || data.Unit != "fahrenheit" || data.Timezone != "America/New_York" {
		t.Fatalf("missing values fabricated: %+v %v", data, err)
	}
	bad, err := parseWeather([]byte(`{"current":{"temperature_2m":12,"relative_humidity_2m":999,"weather_code":999}}`), WeatherSource{})
	if err != nil || bad.Humidity != nil || bad.Code != nil {
		t.Fatal("invalid measurements retained")
	}
	provider := &Providers{resolve: func(context.Context, string) ([]net.IP, error) { return []net.IP{net.ParseIP("8.8.8.8")}, nil }, proxy: func(*http.Request) (*url.URL, error) { return nil, nil }, client: &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		query := request.URL.Query()
		if request.URL.Host != "api.open-meteo.com" || query.Get("temperature_unit") != "fahrenheit" || query.Get("timezone") != "America/New_York" || query.Get("timeformat") != "unixtime" {
			t.Errorf("unexpected weather request: %s", request.URL)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"current":{"temperature_2m":50,"time":1792612800}}`))}, nil
	})}}
	result, err := provider.Fetch(context.Background(), Source{Type: "weather", Config: raw(WeatherSource{Latitude: 40, Longitude: -74, Location: "New York", Units: "fahrenheit", Timezone: "America/New_York"})})
	if err != nil || result.(WeatherData).ObservedAt == nil || result.(WeatherData).ObservedAt.Location() != time.UTC {
		t.Fatal(result, err)
	}
	for _, source := range []WeatherSource{{Location: "X", Units: "kelvin"}, {Location: "X", Timezone: "Unknown/Place"}} {
		if validateWeatherSource(raw(source)) == nil {
			t.Fatal("invalid settings accepted")
		}
	}
}
func TestLocationSearchFixedHostBoundariesAndQuota(t *testing.T) {
	calls := 0
	provider := &Providers{resolve: func(context.Context, string) ([]net.IP, error) { return []net.IP{net.ParseIP("8.8.8.8")}, nil }, proxy: func(*http.Request) (*url.URL, error) { return nil, nil }, client: &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		calls++
		if request.URL.Host != "geocoding-api.open-meteo.com" || request.URL.Query().Get("count") != "5" {
			t.Error("unbounded location request")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"results":[{"name":"서울","country":"대한민국","admin1":"서울","latitude":37.5665,"longitude":126.978,"timezone":"Asia/Seoul"},{"name":"missing"},{"name":"invalid","latitude":91,"longitude":1,"timezone":"UTC"}]}`))}, nil
	})}}
	service := &DashboardService{providers: provider}
	if _, err := service.SearchLocations(context.Background(), "x"); err == nil || calls != 0 {
		t.Fatal("invalid search reached provider")
	}
	for range 10 {
		places, err := service.SearchLocations(context.Background(), "서울")
		if err != nil || len(places) != 1 || places[0].Timezone != "Asia/Seoul" {
			t.Fatal(places, err)
		}
	}
	if _, err := service.SearchLocations(context.Background(), "서울"); err == nil || calls != 10 {
		t.Fatal("search quota bypassed")
	}
}
