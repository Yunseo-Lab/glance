package dashboard

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestProviderContractMatrix(t *testing.T) {
	fixtures := map[string]struct {
		source Source
		body   string
	}{
		"rss":     {Source{Type: "rss", Config: raw(RSSSource{URL: "https://feeds.example.com/rss"})}, `<rss version="2.0"><channel><title>Feed</title></channel></rss>`},
		"weather": {Source{Type: "weather", Config: raw(WeatherSource{Location: "Seoul", Latitude: 37, Longitude: 127})}, `{"current":{"temperature_2m":12}}`},
		"github":  {Source{Type: "github", Config: raw(GitHubSource{Repository: "owner/repo", View: "issues"})}, `[]`},
		"ical":    {Source{Type: "ical", Config: raw(ICalSource{URL: "https://calendar.example.com/calendar.ics", Timezone: "UTC"})}, "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nEND:VCALENDAR\r\n"},
		"json":    {Source{Type: "json", Config: raw(JSONSource{URL: "https://api.example.com/json", Pointer: "/value"})}, `{"value":0}`},
	}
	for kind := range providerDefinitions {
		if _, ok := fixtures[kind]; !ok {
			t.Fatal("provider has no contract fixture", kind)
		}
	}
	for kind, fixture := range fixtures {
		t.Run(kind, func(t *testing.T) {
			for _, scenario := range []struct {
				name           string
				status         int
				body, code     string
				transportError error
			}{{"success", 200, fixture.body, "", nil}, {"malformed", 200, "not protocol data", "", nil}, {"authentication", 401, "private remote body", "authentication", nil}, {"permission", 403, "private remote body", "forbidden", nil}, {"not-found", 404, "private remote body", "not_found", nil}, {"rate-limit", 429, "private remote body", "rate_limited", nil}, {"upstream", 503, "private remote body", "unavailable", nil}, {"too-large", 200, strings.Repeat("x", MaxProviderBody+1), "", nil}, {"timeout", 0, "", "timeout", context.DeadlineExceeded}} {
				t.Run(scenario.name, func(t *testing.T) {
					provider := NewProviders()
					provider.resolve = func(context.Context, string) ([]net.IP, error) { return []net.IP{net.ParseIP("8.8.8.8")}, nil }
					provider.proxy = func(*http.Request) (*url.URL, error) { return nil, nil }
					provider.client.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
						if request.Method != "GET" || request.Header.Get("Cookie") != "" || request.Header.Get("Authorization") != "" {
							t.Error("public read inherited credentials")
						}
						if scenario.transportError != nil {
							return nil, scenario.transportError
						}
						return &http.Response{StatusCode: scenario.status, Header: http.Header{"Retry-After": []string{"60"}}, Body: io.NopCloser(strings.NewReader(scenario.body))}, nil
					})
					result, err := provider.Fetch(context.Background(), fixture.source)
					if scenario.name == "success" {
						if err != nil || result == nil {
							t.Fatal("success contract", result, err)
						}
						return
					}
					if err == nil {
						t.Fatal("failure accepted")
					}
					if strings.Contains(err.Error(), "private remote body") {
						t.Fatal("raw upstream body escaped")
					}
					if scenario.code != "" && classifyProviderError(err) != scenario.code {
						t.Fatal("wrong failure class", classifyProviderError(err), err)
					}
					if scenario.status == 429 {
						var failure *ProviderError
						if !errors.As(err, &failure) || failure.RetryAfter.Seconds() != 60 {
							t.Fatal("rate limit lost")
						}
					}
				})
			}
		})
	}
}
