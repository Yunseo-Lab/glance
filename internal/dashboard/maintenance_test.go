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
