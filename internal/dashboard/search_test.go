package dashboard

import (
	"database/sql"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFTS5UpgradeRebuildAndEntryTriggers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "before-search.db")
	db, err := sql.Open("sqlite3", path+"?_foreign_keys=on")
	if err != nil {
		t.Fatal(err)
	}
	if err = applyMigrations(db, migrations[:10]); err != nil {
		t.Fatal(err)
	}
	board := defaultBoard()
	if _, err = db.Exec("INSERT INTO boards(id,owner_id,revision,document) VALUES('home','local',1,?)", raw(board)); err != nil {
		t.Fatal(err)
	}
	tx, _ := db.Begin()
	if err = replaceSourceRefs(tx, board.ID, board.Sources); err != nil {
		t.Fatal(err)
	}
	tx.Commit()
	_, err = db.Exec("INSERT INTO entries(id,owner_id,item_key,title,url,collected_at,sort_at) VALUES('legacy-search','local','legacy','프론티어 검색 기록','https://example.com/legacy',?,?)", time.Now().UTC().Format(time.RFC3339Nano), time.Now().UnixNano())
	if err != nil {
		t.Fatal(err)
	}
	db.Close()
	store, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	page, err := store.Entries(EntryFilter{Query: "프론티어"})
	if err != nil || page.Total != 1 {
		t.Fatal("migration did not index existing entry", page, err)
	}
	store.db.Exec("UPDATE entries SET title='아키텍처 연구' WHERE id='legacy-search'")
	page, err = store.Entries(EntryFilter{Query: "아키텍처"})
	if err != nil || page.Total != 1 {
		t.Fatal("update missing from FTS", page, err)
	}
	page, _ = store.Entries(EntryFilter{Query: "프론티어"})
	if page.Total != 0 {
		t.Fatal("old token retained")
	}
	store.db.Exec("DELETE FROM entries WHERE id='legacy-search'")
	page, _ = store.Entries(EntryFilter{Query: "아키텍처"})
	if page.Total != 0 {
		t.Fatal("delete missing from FTS")
	}
	if _, err = store.db.Exec("INSERT INTO entry_fts(entry_fts,rank) VALUES('integrity-check',1)"); err != nil {
		t.Fatal("FTS/content mismatch", err)
	}
}
func TestFTS5LiteralQueryAndCapabilities(t *testing.T) {
	store := testStore(t)
	seedReading(t, store, 2)
	page, err := store.Entries(EntryFilter{Query: "Article"})
	if err != nil || page.Total != 2 {
		t.Fatal(page, err)
	}
	for _, query := range []string{`Article" OR "missing`, strings.Repeat("가", 201)} {
		page, err = store.Entries(EntryFilter{Query: query})
		if err == nil && page.Total != 0 {
			t.Fatal("operator syntax escaped its literal phrase", page)
		}
	}
	store.db.Exec("INSERT INTO entries(id,owner_id,item_key,title,url,collected_at) VALUES('foreign-search','other','f','PRIVATE searchable','https://example.com/f','2026-10-22')")
	page, err = store.Entries(EntryFilter{Query: "PRIVATE"})
	if err != nil || page.Total != 0 {
		t.Fatal("foreign search result exposed", page, err)
	}
	response := httptest.NewRecorder()
	NewHandler(store, nil, "").ServeHTTP(response, httptest.NewRequest("GET", "/api/v1/capabilities", nil))
	if response.Code != 200 || !strings.Contains(response.Body.String(), `"fts5":true`) {
		t.Fatal("capability not reported", response.Body.String())
	}
}
