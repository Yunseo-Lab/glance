package dashboard

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOnlineBackupRestoreAndOverwriteProtection(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "live.db")
	store, err := OpenStore(source)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	b, _ := store.Board()
	b.Title = "Live WAL backup"
	saved, err := store.Save(b)
	if err != nil {
		t.Fatal(err)
	}
	backup := filepath.Join(dir, "backup.db")
	if err = backupDatabase(source, backup); err != nil {
		t.Fatal(err)
	}
	if err = backupDatabase(source, backup); err == nil {
		t.Fatal("existing backup overwritten")
	}
	restored := filepath.Join(dir, "restored.db")
	if err = restoreDatabase(backup, restored); err != nil {
		t.Fatal(err)
	}
	copy, err := OpenStore(restored)
	if err != nil {
		t.Fatal(err)
	}
	defer copy.Close()
	actual, err := copy.Board()
	if err != nil {
		t.Fatal(err)
	}
	if actual.Title != saved.Title || actual.Revision != saved.Revision || actual.Widgets[0].ID != saved.Widgets[0].ID {
		t.Fatal("backup lost committed WAL data")
	}
	if err = restoreDatabase(backup, restored); err == nil {
		t.Fatal("restore overwrote database")
	}
	corrupt := filepath.Join(dir, "corrupt.db")
	if err = os.WriteFile(corrupt, []byte("not a SQLite database"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = restoreDatabase(corrupt, filepath.Join(dir, "invalid.db")); err == nil {
		t.Fatal("corrupt backup restored")
	}
}

func TestBackupRestoresEveryBoardAndReadingState(t *testing.T) {
	dir := t.TempDir()
	store, err := OpenStore(filepath.Join(dir, "live.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	home, _ := store.Board()
	board, err := store.CreateBoard("preserved second board", &home)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.db.Exec("INSERT INTO entries(id,owner_id,item_key,title,url,collected_at) VALUES('saved','local','saved','Remember','https://example.com','2026-10-21')")
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.db.Exec("INSERT INTO entry_state(owner_id,entry_id,is_saved,updated_at) VALUES('local','saved',1,'2026-10-21')")
	if err != nil {
		t.Fatal(err)
	}
	backup := filepath.Join(dir, "backup.db")
	if err = backupDatabase(store.path, backup); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "restored.db")
	if err = restoreDatabase(backup, target); err != nil {
		t.Fatal(err)
	}
	restored, err := OpenStore(target)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	boards, err := restored.Boards()
	if err != nil || len(boards) != 2 {
		t.Fatal("second board lost", err)
	}
	actual, err := restored.GetBoard(board.ID)
	if err != nil || actual.Title != board.Title || actual.Widgets[0].ID != board.Widgets[0].ID {
		t.Fatal("second board changed")
	}
	var saved int
	if err = restored.db.QueryRow("SELECT is_saved FROM entry_state WHERE entry_id='saved'").Scan(&saved); err != nil || saved != 1 {
		t.Fatal("saved reading state lost")
	}
}
