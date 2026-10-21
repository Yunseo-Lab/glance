package dashboard

import (
	"crypto/rand"
	"fmt"
	"os"
	"time"
)

// RotateConnectionKey must be used during a maintenance stop. The old key and a
// verified pre-rotation database backup are retained for rollback.
func (s *Store) RotateConnectionKey(nextPath string) error {
	if nextPath == "" {
		return ErrInvalidInput
	}
	if _, err := os.Lstat(nextPath); err == nil {
		return os.ErrExist
	} else if !os.IsNotExist(err) {
		return err
	}
	s.vaultMu.Lock()
	defer s.vaultMu.Unlock()
	oldPath := s.keyPath
	if oldPath == "" {
		oldPath = s.path + ".key"
	}
	previous, err := loadVault(oldPath, false)
	if err != nil {
		return err
	}
	if previous.version >= 1000000 {
		return ErrConnectionLocked
	}
	backup := fmt.Sprintf("%s.before-key-v%d-%s.db", s.path, previous.version+1, time.Now().UTC().Format("20060102T150405.000000000"))
	if err = backupDatabase(s.path, backup); err != nil {
		return err
	}
	material := keyMaterial{Version: previous.version + 1, Key: make([]byte, 32)}
	if _, err = rand.Read(material.Key); err != nil {
		return err
	}
	next, err := newVault(material)
	if err != nil {
		return err
	}
	if err = writeKeyFile(nextPath, material); err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			os.Remove(nextPath)
		}
	}()
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	rows, err := tx.Query("SELECT id,kind,encrypted_secret FROM connections WHERE owner_id='local' AND length(encrypted_secret)>0")
	if err != nil {
		return err
	}
	type record struct {
		id, kind   string
		ciphertext []byte
	}
	var records []record
	for rows.Next() {
		var item record
		if err = rows.Scan(&item.id, &item.kind, &item.ciphertext); err != nil {
			rows.Close()
			return err
		}
		records = append(records, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, item := range records {
		secret, err := previous.decrypt(item.id, item.kind, item.ciphertext)
		if err != nil {
			return err
		}
		ciphertext, err := next.encrypt(item.id, item.kind, secret)
		if err != nil {
			return err
		}
		if _, err = tx.Exec("UPDATE connections SET encrypted_secret=?,key_version=?,generation=generation+1,status='unverified',last_tested_at='' WHERE id=? AND owner_id='local'", ciphertext, next.version, item.id); err != nil {
			return err
		}
	}
	if _, err = tx.Exec("UPDATE sources SET generation=generation+1 WHERE connection_id IS NOT NULL AND owner_id='local'"); err != nil {
		return err
	}
	if _, err = tx.Exec("UPDATE refresh_jobs SET lease_token='',lease_until=0,next_run=0 WHERE source_id IN(SELECT id FROM sources WHERE connection_id IS NOT NULL AND owner_id='local')"); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	committed = true
	s.vault = next
	s.keyPath = nextPath
	return nil
}
