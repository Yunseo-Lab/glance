package dashboard

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestVaultBindsCredentialsToOwnerRecordAndRejectsTampering(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)
	vault, err := newVault(keyMaterial{1, key})
	if err != nil {
		t.Fatal(err)
	}
	secret := connectionSecret{Token: "private-token-must-not-appear-in-ciphertext"}
	encrypted, err := vault.encrypt("one", "github", secret)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encrypted, []byte(secret.Token)) {
		t.Fatal("credential not encrypted")
	}
	restored, err := vault.decrypt("one", "github", encrypted)
	if err != nil || restored.Token != secret.Token {
		t.Fatal("decrypt failed")
	}
	if _, err = vault.decrypt("another", "github", encrypted); err == nil {
		t.Fatal("ciphertext moved to another record")
	}
	if _, err = vault.decrypt("one", "calendar", encrypted); err == nil {
		t.Fatal("ciphertext moved to another provider")
	}
	var envelope secretEnvelope
	json.Unmarshal(encrypted, &envelope)
	envelope.Data[0] ^= 1
	tampered, _ := json.Marshal(envelope)
	if _, err = vault.decrypt("one", "github", tampered); err == nil {
		t.Fatal("tampering accepted")
	}
	rand.Read(key)
	wrong, _ := newVault(keyMaterial{1, key})
	if _, err = wrong.decrypt("one", "github", encrypted); err == nil {
		t.Fatal("wrong key accepted")
	}
}
func TestKeyFileIsExclusivePrivateAndNeverReplacedForExistingCiphertext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "master.key")
	vault, err := loadVault(path, true)
	if err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0600 {
		t.Fatal("unsafe generated key mode")
	}
	ciphertext, _ := vault.encrypt("one", "github", connectionSecret{Token: "saved"})
	again, err := loadVault(path, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = again.decrypt("one", "github", ciphertext); err != nil {
		t.Fatal("existing key replaced")
	}
	os.Chmod(path, 0644)
	if _, err = loadVault(path, false); err == nil {
		t.Fatal("publicly readable host key accepted")
	}
	store := testStore(t)
	store.keyPath = filepath.Join(t.TempDir(), "missing.key")
	store.db.Exec("INSERT INTO connections(id,owner_id,kind,public_config,encrypted_secret) VALUES('one','local','github','{}',?)", ciphertext)
	if _, err = store.connectionVault(true); err == nil {
		t.Fatal("new key generated for existing ciphertext")
	}
	if _, err = os.Stat(store.keyPath); !os.IsNotExist(err) {
		t.Fatal("replacement key was written")
	}
}
