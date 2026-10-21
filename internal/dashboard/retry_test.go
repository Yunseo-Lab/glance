package dashboard

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestRetryAfterAndBoundedJitter(t *testing.T) {
	now := time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		value string
		want  time.Duration
	}{{"3600", time.Hour}, {"259200", 72 * time.Hour}, {"-1", 0}, {"invalid", 0}, {"Wed, 21 Oct 2026 01:00:00 GMT", time.Hour}} {
		if actual := parseRetryAfter(test.value, now); actual != test.want {
			t.Errorf("%s: %s != %s", test.value, actual, test.want)
		}
	}
	deterministic := func(max time.Duration) time.Duration { return max }
	if got := retryDelay(1, &ProviderError{Code: "rate_limited", RetryAfter: time.Hour}, deterministic); got != time.Hour {
		t.Fatal("Retry-After ignored")
	}
	if got := retryDelay(999, context.DeadlineExceeded, deterministic); got > 10*time.Minute || got < 8*time.Minute {
		t.Fatal("backoff not bounded", got)
	}
}
func TestOneHostCannotOccupyAllFetchSlots(t *testing.T) {
	store := testStore(t)
	release := make(chan struct{})
	var active atomic.Int32
	runtime := NewRuntime(store, func(ctx context.Context, source Source) (any, error) {
		active.Add(1)
		defer active.Add(-1)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-release:
			return FeedData{}, nil
		}
	})
	defer runtime.Close()
	for i := 0; i < 20; i++ {
		runtime.Read(Source{ID: newID(), Type: "rss", Config: raw(RSSSource{URL: "https://example.com/" + newID()})})
	}
	eventually(t, func() bool { return active.Load() == 2 })
	runtime.Read(Source{ID: "another-host", Type: "rss", Config: raw(RSSSource{URL: "https://different.example.com/feed"})})
	eventually(t, func() bool { return active.Load() == 3 })
	close(release)
}
