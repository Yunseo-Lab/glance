package dashboard

import (
	"database/sql"
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"
)

func legacyDatabase(t *testing.T, path string) Board {
	t.Helper()
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(migrations[0].sql + "PRAGMA user_version=1;"); err != nil {
		t.Fatal(err)
	}
	board := legacyBoard(t)
	data, _ := json.Marshal(board)
	if _, err = db.Exec("INSERT INTO boards VALUES(?,?,?,?)", board.ID, "local", board.Revision, data); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("INSERT INTO snapshots VALUES(?,?)", "legacy-cache", `{"state":"fresh","data":{"title":"preserved"}}`); err != nil {
		t.Fatal(err)
	}
	return board
}
func TestMigrationPreservesV1AndCreatesVerifiedBackup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	before := legacyDatabase(t, path)
	store, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	after, err := store.Board()
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("legacy document changed", err)
	}
	snapshot, err := store.snapshot("legacy-cache")
	if err != nil || len(snapshot.Data) == 0 {
		t.Fatal("legacy cache lost")
	}
	backups, err := filepath.Glob(path + ".before-v*.db")
	if err != nil || len(backups) != 1 {
		t.Fatal("missing backup", err)
	}
	if err = verifyDatabase(backups[0]); err != nil {
		t.Fatal(err)
	}
	var version int
	store.db.QueryRow("PRAGMA user_version").Scan(&version)
	if version != schemaVersion() {
		t.Fatal("wrong schema version")
	}
	if err = applyMigrations(store.db, migrations); err != nil {
		t.Fatal("not repeatable", err)
	}
}
func TestMigrationFailureRollsBackAndChecksumsRejectDrift(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	legacyDatabase(t, path)
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	broken := []migration{migrations[0], {"broken", `CREATE TABLE should_rollback(x); THIS IS NOT SQL;`}}
	if applyMigrations(db, broken) == nil {
		t.Fatal("broken migration accepted")
	}
	var version, count int
	db.QueryRow("PRAGMA user_version").Scan(&version)
	db.QueryRow("SELECT count(*) FROM sqlite_master WHERE name IN ('should_rollback','schema_migrations')").Scan(&count)
	if version != 1 || count != 0 {
		t.Fatal("partial migration persisted")
	}
	if err = applyMigrations(db, migrations); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("UPDATE schema_migrations SET checksum='tampered' WHERE version=1"); err != nil {
		t.Fatal(err)
	}
	if applyMigrations(db, migrations) == nil {
		t.Fatal("migration drift accepted")
	}
}
func TestFutureDatabaseIsRejectedWithoutRewritingIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "future.db")
	legacyDatabase(t, path)
	db, _ := sql.Open("sqlite3", path)
	db.Exec("PRAGMA user_version=999")
	db.Close()
	if store, err := OpenStore(path); err == nil {
		store.Close()
		t.Fatal("future schema opened")
	}
	db, _ = sql.Open("sqlite3", path)
	defer db.Close()
	var version int
	db.QueryRow("PRAGMA user_version").Scan(&version)
	if version != 999 {
		t.Fatal("future version changed")
	}
}
