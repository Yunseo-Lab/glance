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

func TestGitHubViewsAndRepositoryValidation(t *testing.T) {
	fixtures := map[string]string{"repository": `{"full_name":"owner/repo","html_url":"https://github.com/owner/repo","description":"Demo","stargazers_count":10}`, "releases": `[{"id":1,"tag_name":"v1","html_url":"https://github.com/owner/repo/releases/tag/v1","published_at":"2026-10-01T00:00:00Z"},{"id":2,"name":"Hidden","draft":true,"html_url":"https://github.com/owner/repo/releases/2"}]`, "issues": `[{"id":3,"title":"Issue","html_url":"https://github.com/owner/repo/issues/3"},{"id":4,"title":"PR","html_url":"https://github.com/owner/repo/pull/4","pull_request":{}}]`, "pulls": `[{"id":4,"title":"PR","html_url":"https://github.com/owner/repo/pull/4"}]`}
	for view, fixture := range fixtures {
		config := GitHubSource{Repository: "owner/repo", View: view}
		if err := validateGitHubSource(raw(config)); err != nil {
			t.Fatal(err)
		}
		feed, err := parseGitHub([]byte(fixture), config)
		if err != nil || len(feed.Items) != 1 || len(feed.Items[0].ID) != 64 {
			t.Fatal(view, feed, err)
		}
	}
	for _, repo := range []string{"owner/..", "owner/.", "owner/repo/path", "https://github.com/owner/repo", "owner/repo?x=1", "../repo"} {
		if validateGitHubSource(raw(GitHubSource{Repository: repo, View: "issues"})) == nil {
			t.Fatal("invalid repository", repo)
		}
	}
	if _, err := parseGitHub([]byte(`{"message":"private response"}`), GitHubSource{Repository: "owner/repo", View: "issues"}); err == nil {
		t.Fatal("error object accepted")
	}
}
func TestGitHubUsesFixedReadOnlyEndpointAndRateLimitMetadata(t *testing.T) {
	called := 0
	provider := &Providers{resolve: func(context.Context, string) ([]net.IP, error) { return []net.IP{net.ParseIP("8.8.8.8")}, nil }, proxy: func(*http.Request) (*url.URL, error) { return nil, nil }, client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		called++
		if r.Method != "GET" || r.URL.Host != "api.github.com" || r.URL.Path != "/repos/owner/repo/issues" || r.URL.Query().Get("per_page") != "20" || r.Header.Get("Authorization") != "" || r.Header.Get("X-GitHub-Api-Version") == "" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		}
		if called == 1 {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`[]`))}, nil
		}
		return &http.Response{StatusCode: 403, Header: http.Header{"X-Ratelimit-Remaining": []string{"0"}, "Retry-After": []string{"60"}}, Body: io.NopCloser(strings.NewReader(`private response`))}, nil
	})}}
	source := Source{Type: "github", Config: raw(GitHubSource{Repository: "owner/repo", View: "issues"})}
	if _, err := provider.Fetch(context.Background(), source); err != nil {
		t.Fatal(err)
	}
	_, err := provider.Fetch(context.Background(), source)
	var failure *ProviderError
	if !errors.As(err, &failure) || failure.Code != "rate_limited" || failure.RetryAfter.Seconds() != 60 {
		t.Fatalf("rate limit lost: %#v", err)
	}
}
