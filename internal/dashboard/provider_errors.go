package dashboard

import (
	"context"
	"errors"
	"net"
	"time"
)

// ProviderError contains only a stable safe class; remote bodies/URLs are never
// shown to the browser or included in its Error string.
type ProviderError struct {
	Code       string
	Status     int
	RetryAfter time.Duration
	cause      error
}

func (e *ProviderError) Error() string { return "provider: " + e.Code }
func (e *ProviderError) Unwrap() error { return e.cause }
func classifyProviderError(err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	var timed net.Error
	if errors.As(err, &timed) && timed.Timeout() {
		return "timeout"
	}
	var provider *ProviderError
	if errors.As(err, &provider) {
		return provider.Code
	}
	return "unavailable"
}
func statusError(status int) *ProviderError {
	code := "unavailable"
	switch status {
	case 401:
		code = "authentication"
	case 403:
		code = "forbidden"
	case 404:
		code = "not_found"
	case 429:
		code = "rate_limited"
	}
	return &ProviderError{Code: code, Status: status}
}
