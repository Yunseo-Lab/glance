package dashboard

import (
	"errors"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func seedReading(t *testing.T, store *Store, count int) Source {
	t.Helper()
	board, _ := store.Board()
	source := board.Sources[0]
	now := time.Now().UTC()
	feed := FeedData{Title: "Fixture", Items: []FeedItem{}}
	for index := range count {
		at := now.Add(-time.Duration(index) * time.Minute)
		feed.Items = append(feed.Items, FeedItem{ID: fmt.Sprintf("item-%d", index), Title: fmt.Sprintf("Article %d", index), URL: fmt.Sprintf("https://example.com/article/%d", index), PublishedAt: &at})
	}
	ingestForTest(t, store, source, feed, now)
	return source
}
func TestReadingPaginationIndependentStateAndOwnerIsolation(t *testing.T) {
	store := testStore(t)
	source := seedReading(t, store, 5)
	page, err := store.Entries(EntryFilter{Limit: 2})
	if err != nil || len(page.Items) != 2 || page.Total != 5 || page.NextCursor == "" {
		t.Fatal(page, err)
	}
	first := page.Items[0]
	if len(first.SourceIDs) != 1 || first.SourceIDs[0] != source.ID {
		t.Fatal("missing provenance", first)
	}
	seen := map[string]bool{}
	for _, item := range page.Items {
		seen[item.ID] = true
	}
	for page.NextCursor != "" {
		page, err = store.Entries(EntryFilter{Limit: 2, Cursor: page.NextCursor})
		if err != nil {
			t.Fatal(err)
		}
		for _, item := range page.Items {
			if seen[item.ID] {
				t.Fatal("duplicate keyset entry")
			}
			seen[item.ID] = true
		}
	}
	if len(seen) != 5 {
		t.Fatal("keyset skipped entries")
	}
	yes, no := true, false
	state, err := store.SetEntryState(first.ID, EntryStateInput{Read: &yes})
	if err != nil || !state.IsRead || state.IsSaved {
		t.Fatal(state, err)
	}
	state, err = store.SetEntryState(first.ID, EntryStateInput{Saved: &yes})
	if err != nil || !state.IsRead || !state.IsSaved {
		t.Fatal("saved write lost independent read state", state, err)
	}
	state, err = store.SetEntryState(first.ID, EntryStateInput{Read: &no})
	if err != nil || state.IsRead || !state.IsSaved {
		t.Fatal(state, err)
	}
	saved, _ := store.Entries(EntryFilter{Status: "saved"})
	if saved.Total != 1 || saved.Items[0].ID != first.ID {
		t.Fatal(saved)
	}
	unread, _ := store.Entries(EntryFilter{Status: "unread"})
	if unread.Total != 5 {
		t.Fatal(unread.Total)
	}
	store.db.Exec("INSERT INTO entries(id,owner_id,item_key,title,url,collected_at) VALUES('foreign','other','foreign','PRIVATE','https://example.com/private','2026-10-22')")
	all, _ := store.Entries(EntryFilter{})
	if all.Total != 5 {
		t.Fatal("foreign entry exposed")
	}
	if _, err = store.SetEntryState("foreign", EntryStateInput{Saved: &yes}); !errors.Is(err, ErrNotFound) {
		t.Fatal("foreign state writable", err)
	}
	for _, filter := range []EntryFilter{{Limit: 51}, {Limit: -1}, {Status: "unknown"}, {Cursor: "not-base64"}, {BoardID: "../other"}} {
		if _, err = store.Entries(filter); err == nil {
			t.Fatal("invalid filter accepted", filter)
		}
	}
	if _, err = store.SetEntryState(first.ID, EntryStateInput{}); !errors.Is(err, ErrInvalidInput) {
		t.Fatal("empty update accepted")
	}
}
func TestReadingStateHTTPPersistenceAndStrictPayload(t *testing.T) {
	store := testStore(t)
	seedReading(t, store, 1)
	page, _ := store.Entries(EntryFilter{})
	id := page.Items[0].ID
	handler := NewHandler(store, nil, "")
	for _, test := range []struct {
		body   string
		status int
	}{{`{"saved":true}`, 200}, {`{"read":true}`, 200}, {`{"owner":"other","saved":false}`, 400}, {`{}`, 400}} {
		request := httptest.NewRequest("PATCH", "http://localhost/api/v1/entries/"+id+"/state", strings.NewReader(test.body))
		request.Header.Set("Origin", "http://localhost")
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != test.status {
			t.Fatal(test.body, response.Code, response.Body.String())
		}
	}
	page, _ = store.Entries(EntryFilter{Status: "saved"})
	if len(page.Items) != 1 || !page.Items[0].IsRead {
		t.Fatal("HTTP state did not persist", page)
	}
}
