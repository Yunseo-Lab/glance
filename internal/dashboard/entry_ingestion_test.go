package dashboard

import (
	"database/sql"
	"fmt"
	"testing"
	"time"
)

func ingestForTest(t *testing.T, store *Store, source Source, feed FeedData, now time.Time) IngestionResult {
	t.Helper()
	tx, err := store.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	result, err := ingestFeedEntries(tx, source, raw(feed), now)
	if err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	return result
}
func TestEntryIngestionStableIDsDedupProvenanceAndState(t *testing.T) {
	store := testStore(t)
	board, _ := store.Board()
	first := board.Sources[0]
	second := Source{ID: "second-feed", Type: "rss", Config: raw(RSSSource{URL: "https://other.example.com/feed"})}
	board.Sources = append(board.Sources, second)
	board.Widgets = append(board.Widgets, Widget{ID: "second-widget", Type: "rss", Title: "Other", Size: "wide", SourceID: second.ID, Config: raw(RSSConfig{Limit: 5})})
	board, err := store.Save(board)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	published := now.Add(-time.Hour)
	feed := FeedData{Title: "Source", Items: []FeedItem{{ID: "stable-guid", Title: "<b>Article</b>", URL: "https://example.com/article?utm_source=one", Excerpt: "<p>본문</p><script>hidden()</script>", PublishedAt: &published}}}
	result := ingestForTest(t, store, first, feed, now)
	if result.Added != 1 {
		t.Fatal(result)
	}
	var id, title, excerpt, collected string
	store.db.QueryRow("SELECT id,title,excerpt,collected_at FROM entries WHERE owner_id='local'").Scan(&id, &title, &excerpt, &collected)
	if title != "Article" || excerpt != "본문" {
		t.Fatal(title, excerpt)
	}
	store.db.Exec("INSERT INTO entry_state(owner_id,entry_id,is_read,is_saved,tags,updated_at) VALUES('local',?,1,1,'[\"중요\"]',?)", id, now.Format(time.RFC3339Nano))
	feed.Items[0].Title = "Updated"
	result = ingestForTest(t, store, first, feed, now.Add(time.Hour))
	if result.Added != 0 || result.Updated != 1 {
		t.Fatal(result)
	}
	feed.Items[0].ID = "other-guid"
	feed.Items[0].URL = "https://example.com/article?utm_source=two"
	ingestForTest(t, store, second, feed, now.Add(2*time.Hour))
	var count int
	store.db.QueryRow("SELECT count(*) FROM entries").Scan(&count)
	if count != 1 {
		t.Fatal("duplicate canonical article", count)
	}
	store.db.QueryRow("SELECT count(*) FROM entry_sources WHERE entry_id=?", id).Scan(&count)
	if count != 2 {
		t.Fatal("lost provenance", count)
	}
	var isRead, isSaved int
	var tags, nextCollected string
	store.db.QueryRow("SELECT is_read,is_saved,tags FROM entry_state WHERE entry_id=?", id).Scan(&isRead, &isSaved, &tags)
	store.db.QueryRow("SELECT collected_at FROM entries WHERE id=?", id).Scan(&nextCollected)
	if isRead != 1 || isSaved != 1 || tags != "[\"중요\"]" || nextCollected != collected {
		t.Fatal("state or first collection changed")
	}
	// A provider replacing its GUID for the same URL remains the same reading entry.
	feed.Items[0].ID = "replacement-guid"
	if result = ingestForTest(t, store, second, feed, now); result.Added != 0 {
		t.Fatal("GUID replacement duplicated entry")
	}
}
func TestEntryCapacityRefusesNewItemsWithoutDeletingSavedData(t *testing.T) {
	store := testStore(t)
	board, _ := store.Board()
	source := board.Sources[0]
	now := time.Now().UTC()
	tx, _ := store.db.Begin()
	statement, err := tx.Prepare("INSERT INTO entries(id,owner_id,item_key,title,url,collected_at) VALUES(?,'local',?,'Existing',?,?)")
	if err != nil {
		t.Fatal(err)
	}
	for index := range MaxEntries {
		key := fmt.Sprintf("capacity-%d", index)
		if _, err = statement.Exec(key, key, fmt.Sprintf("https://example.com/%d", index), now.Format(time.RFC3339Nano)); err != nil {
			t.Fatal(err)
		}
	}
	statement.Close()
	tx.Exec("INSERT INTO entry_state(owner_id,entry_id,is_saved,updated_at) VALUES('local','capacity-0',1,?)", now.Format(time.RFC3339Nano))
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	result := ingestForTest(t, store, source, FeedData{Items: []FeedItem{{ID: "new", Title: "New", URL: "https://example.com/new"}, {ID: "existing", Title: "Update", URL: "https://example.com/0"}}}, now)
	if result.Skipped != 1 || result.Updated != 1 || result.Added != 0 {
		t.Fatal(result)
	}
	var count, saved int
	store.db.QueryRow("SELECT count(*) FROM entries").Scan(&count)
	store.db.QueryRow("SELECT is_saved FROM entry_state WHERE entry_id='capacity-0'").Scan(&saved)
	if count != MaxEntries || saved != 1 {
		t.Fatal("capacity deleted owner data")
	}
}
func TestSnapshotFenceAndIngestionCommitAtomically(t *testing.T) {
	store := testStore(t)
	board, _ := store.Board()
	source := board.Sources[0]
	now := time.Now()
	key := sourceKey(source)
	token, _, claimed, err := store.claimRefresh(source, key, now)
	if err != nil || !claimed {
		t.Fatal(err)
	}
	feed := FeedData{Title: "Feed", Items: []FeedItem{{ID: "a", Title: "Article", URL: "https://example.com/a"}}}
	snapshot := Snapshot{State: "fresh", Data: raw(feed), UpdatedAt: &now}
	if err = store.PauseSource(source.ID, true); err != nil {
		t.Fatal(err)
	}
	written, err := store.writeSnapshot(key, snapshot, nil, &refreshFence{SourceID: source.ID, Token: token, Now: now})
	if err != nil || written {
		t.Fatal("obsolete snapshot committed", err)
	}
	var count int
	store.db.QueryRow("SELECT count(*) FROM entries").Scan(&count)
	if count != 0 {
		t.Fatal("obsolete response ingested entries")
	}
	store.PauseSource(source.ID, false)
	token, _, claimed, err = store.claimRefresh(source, key, now)
	if err != nil || !claimed {
		t.Fatal("reclaim", err)
	}
	_, err = store.db.Exec("CREATE TRIGGER fail_ingestion BEFORE INSERT ON entries BEGIN SELECT RAISE(ABORT,'fixture failure'); END")
	if err != nil {
		t.Fatal(err)
	}
	written, err = store.writeSnapshot(key, snapshot, nil, &refreshFence{SourceID: source.ID, Token: token, Now: now})
	if err == nil || written {
		t.Fatal("ingestion failure committed snapshot")
	}
	cached, _ := store.snapshot(key)
	if cached.Data != nil {
		t.Fatal("snapshot was not rolled back")
	}
	store.db.Exec("DROP TRIGGER fail_ingestion")
	written, err = store.writeSnapshot(key, snapshot, &HTTPValidators{ETag: "same-transaction"}, &refreshFence{SourceID: source.ID, Token: token, Now: now})
	if err != nil || !written {
		t.Fatal("valid ingestion failed", err)
	}
	store.db.QueryRow("SELECT count(*) FROM entries").Scan(&count)
	if count != 1 {
		t.Fatal("entry missing")
	}
	var sourceID sql.NullString
	if err = store.db.QueryRow("SELECT source_id FROM entries").Scan(&sourceID); err != nil || sourceID.String != source.ID {
		t.Fatal(sourceID, err)
	}
}
