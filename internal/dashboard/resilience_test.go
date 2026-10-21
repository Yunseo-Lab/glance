package dashboard

import (
	"context"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestProviderPanicIsIsolated(t *testing.T) {
	store := testStore(t)
	runtime := NewRuntime(store, func(context.Context, Source) (any, error) { panic("sensitive remote response must not be logged") })
	defer runtime.Close()
	source := Source{ID: "panic", Type: "rss", Config: raw(RSSSource{URL: "https://example.com/feed"})}
	if _, err := runtime.Read(source); err != nil {
		t.Fatal(err)
	}
	eventually(t, func() bool { snapshot, err := runtime.Read(source); return err == nil && snapshot.State == "error" })
	handler := NewHandler(store, runtime, "")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("GET", "/api/healthz", nil))
	if response.Code != 200 {
		t.Fatal("provider panic stopped health checks")
	}
}

func TestHTTPRecoveryLimitsAndReadiness(t *testing.T) {
	response := httptest.NewRecorder()
	NewHandler(nil, nil, "").ServeHTTP(response, httptest.NewRequest("GET", "/api/healthz", nil))
	if response.Code != 500 || strings.Contains(response.Body.String(), "panic") {
		t.Fatal("request panic must become a generic 500")
	}
	store := testStore(t)
	handler := NewHandler(store, nil, "")
	request := httptest.NewRequest("POST", "/api/v1/dashboard/validate", strings.NewReader(strings.Repeat(" ", 1<<20)+"{}"))
	request.Header.Set("Origin", "http://example.com")
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != 400 {
		t.Fatalf("oversize body: %d", response.Code)
	}
	store.Close()
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("GET", "/api/readyz", nil))
	if response.Code != 503 {
		t.Fatalf("closed database readiness: %d", response.Code)
	}
}

func TestRuntimeBoundsWorkAndCancelsOnClose(t *testing.T) {
	store := testStore(t)
	var active, peak atomic.Int32
	runtime := NewRuntime(store, func(ctx context.Context, source Source) (any, error) {
		n := active.Add(1)
		defer active.Add(-1)
		for old := peak.Load(); n > old; old = peak.Load() {
			if peak.CompareAndSwap(old, n) {
				break
			}
		}
		<-ctx.Done()
		return nil, ctx.Err()
	})
	defer runtime.Close()
	for i := range 20 {
		source := Source{ID: "bounded", Type: "weather", Config: raw(WeatherSource{Latitude: float64(i), Longitude: 0, Location: "Test"})}
		if _, err := runtime.Read(source); err != nil {
			t.Fatal(err)
		}
	}
	eventually(t, func() bool { return active.Load() == 4 })
	done := make(chan struct{})
	go func() { runtime.Close(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("shutdown did not cancel provider work")
	}
	if active.Load() != 0 || peak.Load() > 4 {
		t.Fatal("provider concurrency exceeded bounds")
	}
}
