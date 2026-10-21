package dashboard

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestFakeClockHonorsPersistedBackoffAndThenRecovers(t *testing.T) {
	store := testStore(t)
	home, _ := store.Board()
	source := home.Sources[0]
	var calls atomic.Int32
	var clock atomic.Int64
	clock.Store(time.Now().UnixNano())
	runtime := NewRuntime(store, func(context.Context, Source) (any, error) {
		if calls.Add(1) == 1 {
			return nil, &ProviderError{Code: "rate_limited", RetryAfter: time.Hour}
		}
		return FeedData{Title: "recovered", Items: []FeedItem{}}, nil
	})
	defer runtime.Close()
	runtime.now = func() time.Time { return time.Unix(0, clock.Load()) }
	runtime.jitter = func(time.Duration) time.Duration { return 0 }
	runtime.Read(source)
	eventually(t, func() bool { snapshot, _ := runtime.Read(source); return snapshot.ErrorCode == "rate_limited" })
	clock.Add(int64(59 * time.Minute))
	for i := 0; i < 20; i++ {
		runtime.Read(source)
	}
	if calls.Load() != 1 {
		t.Fatal("retried inside upstream cooldown")
	}
	clock.Add(int64(2 * time.Minute))
	runtime.Read(source)
	eventually(t, func() bool {
		snapshot, _ := runtime.Read(source)
		return snapshot.State == "fresh" && snapshot.ErrorCode == ""
	})
	if calls.Load() != 2 {
		t.Fatal("unexpected recovery fetch count", calls.Load())
	}
}
func TestProviderResponseBudgetRejectsOversizeBody(t *testing.T) {
	provider := NewProviders()
	provider.client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(strings.Repeat("x", MaxProviderBody+1)))}, nil
	})}
	_, err := provider.Fetch(context.Background(), Source{ID: "large", Type: "rss", Config: raw(RSSSource{URL: "https://github.com/fixture.xml"})})
	if err == nil {
		t.Fatal("oversize provider body accepted")
	}
}
