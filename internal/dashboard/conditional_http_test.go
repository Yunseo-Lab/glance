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

func TestConditionalHTTP304KeepsDataAndRefreshesFreshness(t *testing.T) {
	store := testStore(t)
	source := Source{ID: "conditional", Type: "rss", Config: raw(RSSSource{URL: "https://github.com/fixture.atom"})}
	provider := NewProviders(store)
	var calls atomic.Int32
	provider.client = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		n := calls.Add(1)
		headers := http.Header{"Etag": {`"version-1"`}, "Last-Modified": {"Wed, 21 Oct 2026 07:28:00 GMT"}}
		if n == 1 {
			return &http.Response{StatusCode: 200, Header: headers, Body: io.NopCloser(strings.NewReader(`<rss version="2.0"><channel><title>Keep this</title><item><title>Entry</title><link>https://example.com/entry</link></item></channel></rss>`))}, nil
		}
		if request.Header.Get("If-None-Match") != `"version-1"` || request.Header.Get("If-Modified-Since") == "" {
			t.Error("conditional validators missing")
		}
		return &http.Response{StatusCode: 304, Header: headers, Body: io.NopCloser(strings.NewReader(""))}, nil
	})}
	runtime := NewRuntime(store, provider.FetchCached)
	defer runtime.Close()
	runtime.Read(source)
	eventually(t, func() bool { snapshot, _ := runtime.Read(source); return snapshot.State == "fresh" })
	before, _ := store.snapshot(sourceKey(source))
	expired := time.Now().Add(-time.Second)
	before.ExpiresAt = &expired
	if err := store.putSnapshot(sourceKey(source), before); err != nil {
		t.Fatal(err)
	}
	runtime.Read(source)
	eventually(t, func() bool {
		snapshot, _ := runtime.Read(source)
		return calls.Load() == 2 && snapshot.State == "fresh"
	})
	after, _ := runtime.Read(source)
	if string(after.Data) != string(before.Data) || after.UpdatedAt.Before(*before.UpdatedAt) {
		t.Fatal("304 lost or staled good data")
	}
	validators, _ := store.validators(sourceKey(source))
	if validators.ETag != `"version-1"` {
		t.Fatal("validators not persisted")
	}
}
func TestNotModifiedWithoutStoredBodyIsAnError(t *testing.T) {
	store := testStore(t)
	source := Source{ID: "missing-body", Type: "rss", Config: raw(RSSSource{URL: "https://example.com/feed"})}
	runtime := NewRuntime(store, func(context.Context, Source) (any, error) { return FetchResult{NotModified: true}, nil })
	defer runtime.Close()
	runtime.Read(source)
	eventually(t, func() bool {
		snapshot, _ := runtime.Read(source)
		return snapshot.ErrorCode == "invalid_response" && len(snapshot.Data) == 0
	})
}
