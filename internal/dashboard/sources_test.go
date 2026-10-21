package dashboard

import (
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
)

func TestSharedSourcesAreAtomicAndCannotRewriteAnotherBoard(t *testing.T) {
	store := testStore(t)
	home, _ := store.Board()
	copy, err := store.CreateBoard("shared", &home)
	if err != nil {
		t.Fatal(err)
	}
	sources, err := store.Sources()
	if err != nil || len(sources) != len(home.Sources) {
		t.Fatal("source duplicated", err)
	}
	changed := home
	changed.Sources = append([]Source{}, home.Sources...)
	changed.Sources[0].Config = raw(RSSSource{URL: "https://example.com/new-feed"})
	if _, err = store.Save(changed); !errors.Is(err, ErrSourceConflict) {
		t.Fatal("shared query silently rewritten", err)
	}
	actual, _ := store.Board()
	if actual.Revision != home.Revision {
		t.Fatal("failed save advanced revision")
	}
	other, _ := store.GetBoard(copy.ID)
	if string(other.Sources[0].Config) != string(home.Sources[0].Config) {
		t.Fatal("other board changed")
	}
	if _, err = store.db.Exec("DELETE FROM sources WHERE id=?", home.Sources[0].ID); err == nil {
		t.Fatal("referenced source deleted")
	}
	// Changing query identity must explicitly create a new source.
	previous := changed.Sources[0].ID
	changed.Sources[0].ID = newID()
	for i := range changed.Widgets {
		if changed.Widgets[i].SourceID == previous {
			changed.Widgets[i].SourceID = changed.Sources[0].ID
		}
	}
	if _, err = store.Save(changed); err != nil {
		t.Fatal(err)
	}
	sources, _ = store.Sources()
	if len(sources) != len(home.Sources)+1 {
		t.Fatal("new source did not remain independent")
	}
	// Reading state is not stored in a board or its source cache.
	_, err = store.db.Exec("INSERT INTO entries(id,owner_id,source_id,item_key,title,url,collected_at) VALUES('entry','local',?,'item','Title','https://example.com','2026-10-21')", previous)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.db.Exec("INSERT INTO entry_state(owner_id,entry_id,is_read,is_saved,updated_at) VALUES('local','entry',1,1,'2026-10-21')")
	if err != nil {
		t.Fatal(err)
	}
	if err = store.DeleteBoard(copy.ID, copy.Revision); err != nil {
		t.Fatal(err)
	}
	var saved int
	if err = store.db.QueryRow("SELECT is_saved FROM entry_state WHERE entry_id='entry'").Scan(&saved); err != nil || saved != 1 {
		t.Fatal("board deletion erased saved state")
	}
}
func TestMigrationRejectsConflictingLegacySourceIDsAtomically(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.db")
	board := legacyDatabase(t, path)
	db, err := existingDatabase(path)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()
	// Open writable legacy data without the application upgrade for a collision fixture.
	storeDB, err := openFixtureDB(path)
	if err != nil {
		t.Fatal(err)
	}
	board.ID = "another"
	board.Sources[0].Config = raw(RSSSource{URL: "https://example.com/conflict"})
	data, _ := json.Marshal(board)
	if _, err = storeDB.Exec("INSERT INTO boards VALUES(?,?,?,?)", board.ID, "local", board.Revision, data); err != nil {
		t.Fatal(err)
	}
	storeDB.Close()
	if store, err := OpenStore(path); err == nil {
		store.Close()
		t.Fatal("ambiguous migration succeeded")
	}
	storeDB, err = openFixtureDB(path)
	if err != nil {
		t.Fatal(err)
	}
	defer storeDB.Close()
	var version int
	storeDB.QueryRow("PRAGMA user_version").Scan(&version)
	if version != 1 {
		t.Fatal("failed normalization changed schema")
	}
}

func openFixtureDB(path string) (*sql.DB, error) { return sql.Open("sqlite3", path) }
