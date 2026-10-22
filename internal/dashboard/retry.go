package dashboard

import (
	"encoding/json"
	"errors"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func parseRetryAfter(value string, now time.Time) time.Duration {
	value = strings.TrimSpace(value)
	if seconds, err := strconv.ParseInt(value, 10, 64); err == nil {
		if seconds < 0 {
			return 0
		}
		if seconds > 31536000 {
			seconds = 31536000
		}
		return time.Duration(seconds) * time.Second
	}
	if date, err := http.ParseTime(value); err == nil && date.After(now) {
		delay := date.Sub(now)
		if delay > 365*24*time.Hour {
			return 365 * 24 * time.Hour
		}
		return delay
	}
	return 0
}
func retryDelay(failures int, err error, jitter func(time.Duration) time.Duration) time.Duration {
	if failures < 1 {
		failures = 1
	}
	if failures > 5 {
		failures = 5
	}
	base := time.Duration(1<<failures) * 15 * time.Second
	delay := base + jitter(base/5)
	var provider *ProviderError
	if errors.As(err, &provider) && provider.RetryAfter > delay {
		delay = provider.RetryAfter
	}
	return delay
}
func randomJitter(max time.Duration) time.Duration {
	if max <= 0 {
		return 0
	}
	return time.Duration(rand.Int64N(int64(max) + 1))
}
func sourceURLHost(source Source) string {
	var config struct {
		URL string `json:"url"`
	}
	_ = json.Unmarshal(source.Config, &config)
	u, err := url.Parse(config.URL)
	if err != nil || u.Hostname() == "" {
		return source.Type
	}
	return strings.ToLower(u.Hostname())
}
func fixedHost(host string) func(Source) string { return func(Source) string { return host } }
