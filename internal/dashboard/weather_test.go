package dashboard

import "testing"

func TestWeatherResponseSuccessAndMissingData(t *testing.T) {
	source := WeatherSource{Location: "Seoul", Latitude: 37.5665, Longitude: 126.978}
	data := []byte(`{"current":{"temperature_2m":12.5,"apparent_temperature":11.2,"relative_humidity_2m":63,"weather_code":3}}`)
	weather, err := parseWeather(data, source)
	if err != nil {
		t.Fatal(err)
	}
	if weather.Location != "Seoul" || weather.Temperature != 12.5 || weather.FeelsLike != 11.2 || weather.Humidity != 63 || weather.Code != 3 {
		t.Fatalf("unexpected weather: %+v", weather)
	}
	for _, body := range []string{`{}`, `{"current":{}}`, `{"current":null}`, `not json`} {
		if _, err = parseWeather([]byte(body), source); err == nil {
			t.Fatalf("invalid weather accepted: %s", body)
		}
	}
}
