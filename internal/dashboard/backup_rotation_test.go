package dashboard

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestBackupRotationRetainsGoodCopies(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	store, err := OpenStore(filepath.Join(t.TempDir(), "source.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	for i := 1; i <= 5; i++ {
		if err := backupDatabase(store.path, filepath.Join(dir, fmt.Sprintf("dashboard-202610%02dT120000Z-1.db", i))); err != nil {
			t.Fatal(err)
		}
	}
	broken := filepath.Join(dir, "dashboard-20261009T120000Z-1.db")
	os.WriteFile(broken, []byte("broken"), 0600)
	link := filepath.Join(dir, "dashboard-20261010T120000Z-1.db")
	os.Symlink(store.path, link)
	dry, err := pruneBackups(dir, 2, false)
	if err != nil || dry.Valid != 5 || dry.Invalid != 1 || len(dry.Removed) != 3 {
		t.Fatal(dry, err)
	}
	files, _ := os.ReadDir(dir)
	if len(files) != 7 {
		t.Fatal("dry run deleted files")
	}
	applied, err := pruneBackups(dir, 2, true)
	if err != nil || applied.Retained != 2 {
		t.Fatal(applied, err)
	}
	files, _ = os.ReadDir(dir)
	if len(files) != 4 {
		t.Fatal("unexpected deletion")
	}
	if _, err := os.Lstat(link); err != nil {
		t.Fatal("symlink removed")
	}
	if _, err := os.Stat(broken); err != nil {
		t.Fatal("corrupt file removed")
	}
	if _, err := pruneBackups(dir, 1, true); err == nil {
		t.Fatal("unsafe retention accepted")
	}
	if err := os.Chmod(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := pruneBackups(dir, 2, true); err == nil {
		t.Fatal("public backup directory accepted")
	}
}

func TestBackupAndSeparatedKeyRestore(t *testing.T) {
	store := testStore(t)
	connection, err := store.CreateConnection(ConnectionInput{Name: "Restoration fixture", Kind: "github", Token: "offline-recovery-fixture-token"})
	if err != nil {
		t.Fatal(err)
	}
	offsiteDB := filepath.Join(t.TempDir(), "database-only.db")
	if err := backupDatabase(store.path, offsiteDB); err != nil {
		t.Fatal(err)
	}
	material, err := os.ReadFile(store.keyPath)
	if err != nil {
		t.Fatal(err)
	}
	separateKey := filepath.Join(t.TempDir(), "key-only.json")
	if err = os.WriteFile(separateKey, material, 0600); err != nil {
		t.Fatal(err)
	}
	if err = verifyBackupKey(offsiteDB, separateKey); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "restored.db")
	if err = restoreDatabase(offsiteDB, target); err != nil {
		t.Fatal(err)
	}
	restored, err := OpenStore(target)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	if _, _, err = restored.resolveConnection(connection.ID); err == nil {
		t.Fatal("DB-only restoration silently created a key")
	}
	restored.keyPath = separateKey
	_, secret, err := restored.resolveConnection(connection.ID)
	if err != nil || secret.Token != "offline-recovery-fixture-token" {
		t.Fatal("separate DB/key recovery failed", err)
	}
	wrong := filepath.Join(t.TempDir(), "wrong.key")
	if _, err = loadVault(wrong, true); err != nil {
		t.Fatal(err)
	}
	if err = verifyBackupKey(offsiteDB, wrong); err == nil {
		t.Fatal("wrong key accepted")
	}
}
