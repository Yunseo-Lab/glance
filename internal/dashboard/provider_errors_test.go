package dashboard

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestProviderErrorsPreserveLastSuccessWithoutLeakingCause(t *testing.T) {
	store := testStore(t)
	source := Source{ID: "rate", Type: "rss", Config: raw(RSSSource{URL: "https://example.com/feed"})}
	runtime := NewRuntime(store, func(context.Context, Source) (any, error) {
		return nil, &ProviderError{Code: "rate_limited", Status: 429, cause: errors.New("secret-token-sensitive-body")}
	})
	defer runtime.Close()
	runtime.Read(source)
	eventually(t, func() bool {
		snapshot, _ := runtime.Read(source)
		return snapshot.ErrorCode == "rate_limited" && snapshot.AttemptedAt != nil && snapshot.RetryAt != nil
	})
	snapshot, _ := runtime.Read(source)
	if strings.Contains(string(raw(snapshot)), "secret-token") {
		t.Fatal("private cause exposed")
	}
	if classifyProviderError(context.DeadlineExceeded) != "timeout" || statusError(401).Code != "authentication" {
		t.Fatal("incorrect safe error class")
	}
}
