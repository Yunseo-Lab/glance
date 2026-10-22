package dashboard

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPrivateGitHubCacheIdentityGenerationRevocationAndRotation(t *testing.T) {
	store := testStore(t)
	a, err := store.CreateConnection(ConnectionInput{Name: "A", Kind: "github", Token: "fixture-private-github-token-a"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := store.CreateConnection(ConnectionInput{Name: "B", Kind: "github", Token: "fixture-private-github-token-b"})
	if err != nil {
		t.Fatal(err)
	}
	source := Source{ID: "private-a", Type: "github", ConnectionID: a.ID, Config: raw(GitHubSource{Repository: "owner/private", View: "issues"})}
	other := source
	other.ID = "private-b"
	other.ConnectionID = b.ID
	public := source
	public.ConnectionID = ""
	keyA, err := store.cacheKey(source)
	if err != nil {
		t.Fatal(err)
	}
	keyB, _ := store.cacheKey(other)
	if keyA == keyB || keyA == sourceKey(public) {
		t.Fatal("private cache identity collision")
	}
	board, _ := store.Board()
	board.Widgets = []Widget{{ID: "a", Type: "github", Title: "A", Size: "wide", SourceID: source.ID, Config: raw(RSSConfig{Limit: 5})}, {ID: "b", Type: "github", Title: "B", Size: "wide", SourceID: other.ID, Config: raw(RSSConfig{Limit: 5})}}
	board.Sources = []Source{source, other}
	board, err = store.Save(board)
	if err != nil {
		t.Fatal(err)
	}
	provider := &Providers{store: store, resolve: func(context.Context, string) ([]net.IP, error) { return []net.IP{net.ParseIP("8.8.8.8")}, nil }, proxy: func(*http.Request) (*url.URL, error) { return nil, nil }, client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != "api.github.com" || r.Header.Get("Authorization") != "Bearer fixture-private-github-token-a" {
			t.Error("credential went to wrong endpoint")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`[]`))}, nil
	})}}
	if _, err = provider.Fetch(context.Background(), source); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	for _, key := range []string{keyA, keyB} {
		if _, err = store.writeSnapshot(key, Snapshot{State: "fresh", Data: raw(map[string]any{"private": "body"}), UpdatedAt: &now}, &HTTPValidators{ETag: "private-validator"}, nil); err != nil {
			t.Fatal(err)
		}
	}
	ingestForTest(t, store, source, FeedData{Items: []FeedItem{{ID: "private-unsaved", Title: "Unsaved", URL: "https://github.com/owner/private/issues/1"}, {ID: "private-saved", Title: "Saved", URL: "https://github.com/owner/private/issues/2"}}}, now)
	var savedEntryID string
	store.db.QueryRow("SELECT id FROM entries WHERE title='Saved'").Scan(&savedEntryID)
	store.db.Exec("INSERT INTO entry_state(owner_id,entry_id,is_saved,updated_at) VALUES('local',?,1,?)", savedEntryID, now.Format(time.RFC3339Nano))
	if err = store.RevokeConnection(a.ID); err != nil {
		t.Fatal(err)
	}
	snapshot, _ := store.snapshot(keyA)
	validators, _ := store.validators(keyA)
	if snapshot.Data != nil || validators.ETag != "" {
		t.Fatal("revocation retained private body/validator")
	}
	snapshot, _ = store.snapshot(keyB)
	if snapshot.Data == nil {
		t.Fatal("revocation cleared another connection")
	}
	if _, err = provider.Fetch(context.Background(), source); err == nil {
		t.Fatal("revoked credential used")
	}
	var privateCount int
	store.db.QueryRow("SELECT count(*) FROM entries WHERE identity_scope=?", "connection:"+a.ID).Scan(&privateCount)
	if privateCount != 1 {
		t.Fatal("revocation did not preserve only explicitly saved reading data")
	}
	board.Title = "Revoked widget remains editable"
	if _, err = store.Save(board); err != nil {
		t.Fatal("revoked source blocks unrelated board edits", err)
	}
	if err = store.RotateConnectionKey(filepath.Join(t.TempDir(), "rotated.key")); err != nil {
		t.Fatal(err)
	}
	nextKey, err := store.cacheKey(other)
	if err != nil || nextKey == keyB {
		t.Fatal("rotation reused identity")
	}
	snapshot, _ = store.snapshot(keyB)
	validators, _ = store.validators(keyB)
	if snapshot.Data != nil || validators.ETag != "" {
		t.Fatal("rotation retained private cache")
	}
}
