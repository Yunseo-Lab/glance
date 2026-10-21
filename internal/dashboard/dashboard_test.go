package dashboard

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	s, err := OpenStore(filepath.Join(t.TempDir(), "dashboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}
func eventually(t *testing.T, check func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if check() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition did not complete")
}

func TestStorePersistenceAndOptimisticRevision(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dashboard.db")
	store, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	b, err := store.Board()
	if err != nil {
		t.Fatal(err)
	}
	stale := b
	b.Title = "저장된 대시보드"
	saved, err := store.Save(b)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Revision != b.Revision+1 {
		t.Fatal("revision did not advance")
	}
	if _, err = store.Save(stale); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
	if err = store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	restored, err := store.Board()
	if err != nil {
		t.Fatal(err)
	}
	if restored.Title != saved.Title || restored.Revision != saved.Revision || restored.Widgets[0].ID != saved.Widgets[0].ID {
		t.Fatal("document or stable IDs lost on restart")
	}
}

func TestRuntimeIndependentSourcesAndDeduplication(t *testing.T) {
	store := testStore(t)
	slow := Source{ID: "slow", Type: "rss", Config: raw(RSSSource{URL: "https://example.com/slow"})}
	fast := Source{ID: "fast", Type: "rss", Config: raw(RSSSource{URL: "https://example.com/fast"})}
	release := make(chan struct{})
	var slowCalls atomic.Int32
	runtime := NewRuntime(store, func(ctx context.Context, s Source) (any, error) {
		if s.ID == "slow" {
			slowCalls.Add(1)
			select {
			case <-release:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		return FeedData{Title: s.ID, Items: []FeedItem{}}, nil
	})
	defer runtime.Close()
	start := time.Now()
	if _, err := runtime.Read(slow); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Read(fast); err != nil {
		t.Fatal(err)
	}
	if time.Since(start) > 250*time.Millisecond {
		t.Fatal("read blocked on provider")
	}
	for i := 0; i < 10; i++ {
		alias := slow
		alias.ID = "same-data"
		if _, err := runtime.Read(alias); err != nil {
			t.Fatal(err)
		}
	}
	eventually(t, func() bool { s, err := runtime.Read(fast); return err == nil && s.State == "fresh" })
	if slowCalls.Load() != 1 {
		t.Fatalf("duplicate fetches: %d", slowCalls.Load())
	}
	close(release)
	eventually(t, func() bool { s, err := runtime.Read(slow); return err == nil && s.State == "fresh" })
}

func TestRuntimeKeepsLastGoodDataAndBacksOff(t *testing.T) {
	store := testStore(t)
	source := Source{ID: "feed", Type: "rss", Config: raw(RSSSource{URL: "https://example.com/feed"})}
	updated := time.Now().Add(-time.Hour)
	expired := time.Now().Add(-time.Minute)
	old := Snapshot{Data: raw(FeedData{Title: "last success", Items: []FeedItem{}}), UpdatedAt: &updated, ExpiresAt: &expired}
	if err := store.putSnapshot(sourceKey(source), old); err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	runtime := NewRuntime(store, func(context.Context, Source) (any, error) {
		calls.Add(1)
		return nil, errors.New("upstream unavailable")
	})
	defer runtime.Close()
	snap, err := runtime.Read(source)
	if err != nil || snap.State != "stale" || snap.Data == nil {
		t.Fatalf("stale data not served: %+v %v", snap, err)
	}
	eventually(t, func() bool {
		s, err := runtime.Read(source)
		return err == nil && s.Error != "" && s.Data != nil && s.RetryAt != nil
	})
	for i := 0; i < 10; i++ {
		_, _ = runtime.Read(source)
	}
	if calls.Load() != 1 {
		t.Fatal("provider retried before backoff")
	}
	tooOld := time.Now().Add(-25 * time.Hour)
	old.UpdatedAt = &tooOld
	future := time.Now().Add(time.Hour)
	old.RetryAt = &future
	if err := store.putSnapshot(sourceKey(source), old); err != nil {
		t.Fatal(err)
	}
	snap, err = runtime.Read(source)
	if err != nil || snap.Data != nil {
		t.Fatal("data older than maximum stale age exposed")
	}
}

func TestHTTPRevisionValidationAndCSRF(t *testing.T) {
	store := testStore(t)
	runtime := NewRuntime(store, func(context.Context, Source) (any, error) { return nil, errors.New("unused") })
	defer runtime.Close()
	handler := NewHandler(store, runtime, "")
	b, _ := store.Board()
	b.Title = "웹에서 수정"
	body, _ := json.Marshal(b)
	put := func(payload []byte, origin string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("PUT", "http://localhost/api/v1/dashboard", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		return res
	}
	if res := put(body, ""); res.Code != 403 {
		t.Fatalf("missing origin: %d", res.Code)
	}
	if res := put(body, "http://evil.example"); res.Code != 403 {
		t.Fatalf("cross origin: %d", res.Code)
	}
	if res := put(body, "http://localhost"); res.Code != 200 {
		t.Fatalf("save: %d %s", res.Code, res.Body.String())
	}
	if res := put(body, "http://localhost"); res.Code != 409 {
		t.Fatalf("stale revision: %d", res.Code)
	}
	bad, _ := store.Board()
	bad.Widgets[0].Type = "unknown"
	payload, _ := json.Marshal(bad)
	if res := put(payload, "http://localhost"); res.Code != 400 {
		t.Fatalf("invalid widget: %d", res.Code)
	}
	if res := put(append(body, []byte(" {}")...), "http://localhost"); res.Code != 400 {
		t.Fatalf("trailing JSON: %d", res.Code)
	}
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, httptest.NewRequest("GET", "http://localhost/api/v1/dashboard", nil))
	var saved Board
	if err := json.Unmarshal(res.Body.Bytes(), &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Title != b.Title {
		t.Fatal("saved document unavailable")
	}
}

func TestAuthenticationProtectsDataButAllowsLoginShell(t *testing.T) {
	store := testStore(t)
	runtime := NewRuntime(store, nil)
	defer runtime.Close()
	handler := NewHandler(store, runtime, "a-private-dashboard-token")
	for _, path := range []string{"/api/v1/dashboard", "/api/healthz", "/api/v1/snapshots"} {
		req := httptest.NewRequest("GET", path, nil)
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		if res.Code != 401 {
			t.Fatalf("unprotected %s: %d", path, res.Code)
		}
	}
	req := httptest.NewRequest("GET", "/api/v1/dashboard", nil)
	req.SetBasicAuth("owner", "a-private-dashboard-token")
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != 200 {
		t.Fatal(res.Code)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestProviderParsesFeedsAndRejectsPrivateDestinations(t *testing.T) {
	for _, ip := range []string{"127.0.0.1", "10.0.0.1", "169.254.169.254", "100.64.0.1", "::1", "::ffff:127.0.0.1", "fd00::1", "198.18.0.1"} {
		if publicIP(net.ParseIP(ip)) {
			t.Errorf("private address accepted: %s", ip)
		}
	}
	for _, address := range []string{"http://example.com", "https://user:password@example.com", "https://example.com:8443"} {
		if _, err := publicURL(address); err == nil {
			t.Errorf("unsafe URL accepted: %s", address)
		}
	}
	var called atomic.Int32
	provider := &Providers{proxyHosts: map[string]bool{"8.8.8.8": true}, client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		called.Add(1)
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`<?xml version="1.0"?><rss version="2.0"><channel><title>Test feed</title><item><title>A real item</title><link>https://example.com/article</link></item><item><title>Unsafe link</title><link>javascript:alert(1)</link></item></channel></rss>`))}, nil
	})}}
	_, err := provider.Fetch(context.Background(), Source{Type: "rss", Config: raw(RSSSource{URL: "https://127.0.0.1/private"})})
	if err == nil || called.Load() != 0 {
		t.Fatal("private fetch reached transport")
	}
	data, err := provider.Fetch(context.Background(), Source{Type: "rss", Config: raw(RSSSource{URL: "https://8.8.8.8/feed"})})
	if err != nil {
		t.Fatal(err)
	}
	feed := data.(FeedData)
	if len(feed.Items) != 1 || feed.Items[0].Title != "A real item" {
		t.Fatalf("unexpected normalized feed: %+v", feed)
	}
}

func TestBoardRejectsDanglingSourcesAndInvalidConfig(t *testing.T) {
	b := defaultBoard()
	if err := b.Validate(); err != nil {
		t.Fatal(err)
	}
	b.Sources = []Source{}
	if err := b.Validate(); err == nil {
		t.Fatal("dangling data source accepted")
	}
	b = defaultBoard()
	b.Widgets[0].Config = raw(ClockConfig{Timezone: "Not/A_Timezone"})
	if err := b.Validate(); err == nil {
		t.Fatal("invalid timezone accepted")
	}
}

func TestValidationPreviewDoesNotPersistOrFetch(t *testing.T) {
	store := testStore(t)
	var calls atomic.Int32
	runtime := NewRuntime(store, func(context.Context, Source) (any, error) { calls.Add(1); return nil, errors.New("unexpected fetch") })
	defer runtime.Close()
	handler := NewHandler(store, runtime, "")
	original, err := store.Board()
	if err != nil {
		t.Fatal(err)
	}
	preview := original
	preview.Title = "Preview only"
	for _, tc := range []struct {
		origin  string
		invalid bool
		status  int
	}{{"http://localhost", false, 200}, {"http://evil.example", false, 403}, {"http://localhost", true, 400}} {
		candidate := preview
		if tc.invalid {
			candidate.ID = "unknown"
		}
		body, _ := json.Marshal(candidate)
		req := httptest.NewRequest("POST", "http://localhost/api/v1/dashboard/validate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", tc.origin)
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		if res.Code != tc.status {
			t.Fatalf("preview returned %d, expected %d: %s", res.Code, tc.status, res.Body.String())
		}
	}
	after, err := store.Board()
	if err != nil {
		t.Fatal(err)
	}
	if after.Title != original.Title || after.Revision != original.Revision || calls.Load() != 0 {
		t.Fatal("preview changed storage or fetched external data")
	}
}
