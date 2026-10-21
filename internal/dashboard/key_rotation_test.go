package dashboard

import (
	"os"
	"path/filepath"
	"testing"
)

func TestKeyRotationPreservesRollbackAndRejectsOldKeyForNewCiphertext(t *testing.T) {
	store := testStore(t)
	connection, err := store.CreateConnection(ConnectionInput{Name: "GitHub", Kind: "github", Token: "fixture-token-that-must-survive"})
	if err != nil {
		t.Fatal(err)
	}
	oldKeyPath := store.keyPath
	previous, err := loadVault(oldKeyPath, false)
	if err != nil {
		t.Fatal(err)
	}
	nextPath := filepath.Join(t.TempDir(), "next.key")
	if err = store.RotateConnectionKey(nextPath); err != nil {
		t.Fatal(err)
	}
	_, secret, err := store.resolveConnection(connection.ID)
	if err != nil || secret.Token != "fixture-token-that-must-survive" {
		t.Fatal("rotation lost credential")
	}
	var ciphertext []byte
	store.db.QueryRow("SELECT encrypted_secret FROM connections WHERE id=?", connection.ID).Scan(&ciphertext)
	if _, err = previous.decrypt(connection.ID, "github", ciphertext); err == nil {
		t.Fatal("old key decrypts new ciphertext")
	}
	backups, _ := filepath.Glob(store.path + ".before-key-*.db")
	if len(backups) != 1 {
		t.Fatal("missing rollback DB")
	}
	restoredPath := filepath.Join(t.TempDir(), "rollback.db")
	if err = restoreDatabase(backups[0], restoredPath); err != nil {
		t.Fatal(err)
	}
	restored, err := OpenStore(restoredPath)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	restored.keyPath = oldKeyPath
	if _, _, err = restored.resolveConnection(connection.ID); err != nil {
		t.Fatal("old DB/key pair cannot restore", err)
	}
	if err = store.RotateConnectionKey(nextPath); err == nil {
		t.Fatal("existing output key overwritten")
	}
}
func TestFailedRotationLeavesOriginalCiphertextAndRemovesNewKey(t *testing.T) {
	store := testStore(t)
	connection, err := store.CreateConnection(ConnectionInput{Name: "GitHub", Kind: "github", Token: "fixture-token-before-failure"})
	if err != nil {
		t.Fatal(err)
	}
	var before []byte
	store.db.QueryRow("SELECT encrypted_secret FROM connections WHERE id=?", connection.ID).Scan(&before)
	store.db.Exec("INSERT INTO connections(id,owner_id,kind,public_config,encrypted_secret) VALUES('broken','local','github','{}','corrupt')")
	nextPath := filepath.Join(t.TempDir(), "should-not-remain.key")
	if err = store.RotateConnectionKey(nextPath); err == nil {
		t.Fatal("corrupt row rotation accepted")
	}
	if _, err = os.Stat(nextPath); !os.IsNotExist(err) {
		t.Fatal("uncommitted output key left behind")
	}
	var after []byte
	store.db.QueryRow("SELECT encrypted_secret FROM connections WHERE id=?", connection.ID).Scan(&after)
	if string(before) != string(after) {
		t.Fatal("partial ciphertext update persisted")
	}
	if _, _, err = store.resolveConnection(connection.ID); err != nil {
		t.Fatal("original pair stopped working")
	}
}

func TestMaintenanceKeyRotationAndRestart(t *testing.T) {
	store := testStore(t)
	connection, err := store.CreateConnection(ConnectionInput{Name: "CLI fixture", Kind: "github", Token: "only-a-fixture-credential-value"})
	if err != nil {
		t.Fatal(err)
	}
	previous := store.keyPath
	next := filepath.Join(t.TempDir(), "cli-next.key")
	if code := maintenance("rotate-connection-key", []string{"--db", store.path, "--key-file", previous, "--output-key", next}); code != 0 {
		t.Fatal("CLI key rotation failed")
	}
	reopened, err := OpenStore(store.path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	reopened.keyPath = next
	if _, _, err = reopened.resolveConnection(connection.ID); err != nil {
		t.Fatal("restarted CLI-rotated connection could not decrypt")
	}
}
