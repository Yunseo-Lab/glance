package dashboard

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

var ErrConnectionLocked = errors.New("연결 암호화 키를 사용할 수 없습니다. 원래 키 파일을 복구하세요")

type keyMaterial struct {
	Version int    `json:"version"`
	Key     []byte `json:"key"`
}
type Vault struct {
	version int
	aead    cipher.AEAD
}
type secretEnvelope struct {
	Version    int    `json:"v"`
	KeyVersion int    `json:"keyVersion"`
	Nonce      []byte `json:"nonce"`
	Data       []byte `json:"data"`
}

func newVault(material keyMaterial) (*Vault, error) {
	if material.Version < 1 || len(material.Key) != 32 {
		return nil, ErrConnectionLocked
	}
	block, err := aes.NewCipher(material.Key)
	if err != nil {
		return nil, ErrConnectionLocked
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, ErrConnectionLocked
	}
	return &Vault{material.Version, aead}, nil
}
func writeKeyFile(path string, material keyMaterial) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	complete := false
	defer func() {
		file.Close()
		if !complete {
			os.Remove(path)
		}
	}()
	if err = json.NewEncoder(file).Encode(material); err != nil {
		return err
	}
	if err = file.Sync(); err != nil {
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	directory, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	err = directory.Sync()
	directory.Close()
	if err != nil {
		return err
	}
	complete = true
	return nil
}
func loadVault(path string, create bool) (*Vault, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) && create {
		key := make([]byte, 32)
		if _, err = rand.Read(key); err != nil {
			return nil, err
		}
		if err = writeKeyFile(path, keyMaterial{Version: 1, Key: key}); err != nil && !os.IsExist(err) {
			return nil, ErrConnectionLocked
		}
		info, err = os.Lstat(path)
	}
	if err != nil || !info.Mode().IsRegular() {
		return nil, ErrConnectionLocked
	}
	mode := info.Mode().Perm()
	mountedReadOnly := mode == 0444 && filepath.Dir(path) == "/run/secrets"
	if mode != 0600 && mode != 0400 && !mountedReadOnly {
		return nil, ErrConnectionLocked
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, ErrConnectionLocked
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 4097))
	if err != nil || len(data) > 4096 {
		return nil, ErrConnectionLocked
	}
	material := keyMaterial{Version: 1}
	if len(data) == 32 {
		material.Key = data
	} else if err = json.Unmarshal(data, &material); err != nil {
		return nil, ErrConnectionLocked
	}
	return newVault(material)
}
func (v *Vault) binding(id, kind string) []byte {
	return []byte(fmt.Sprintf("glance-connection\x00local\x00%s\x00%s\x00%d", id, kind, v.version))
}
func (v *Vault) encrypt(id, kind string, secret connectionSecret) ([]byte, error) {
	payload, err := json.Marshal(secret)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, v.aead.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return nil, err
	}
	ciphertext := v.aead.Seal(nil, nonce, payload, v.binding(id, kind))
	return json.Marshal(secretEnvelope{Version: 1, KeyVersion: v.version, Nonce: nonce, Data: ciphertext})
}
func (v *Vault) decrypt(id, kind string, data []byte) (connectionSecret, error) {
	var envelope secretEnvelope
	if len(data) > 16384 || json.Unmarshal(data, &envelope) != nil || envelope.Version != 1 || envelope.KeyVersion != v.version || len(envelope.Nonce) != v.aead.NonceSize() {
		return connectionSecret{}, ErrConnectionLocked
	}
	payload, err := v.aead.Open(nil, envelope.Nonce, envelope.Data, v.binding(id, kind))
	if err != nil {
		return connectionSecret{}, ErrConnectionLocked
	}
	var secret connectionSecret
	if err = json.Unmarshal(payload, &secret); err != nil {
		return connectionSecret{}, ErrConnectionLocked
	}
	return secret, nil
}
func (s *Store) connectionVault(create bool) (*Vault, error) {
	s.vaultMu.Lock()
	defer s.vaultMu.Unlock()
	if s.vault != nil {
		return s.vault, nil
	}
	var count int
	if create {
		if err := s.db.QueryRow("SELECT count(*) FROM connections WHERE length(encrypted_secret)>0").Scan(&count); err != nil {
			return nil, err
		}
	}
	path := s.keyPath
	if path == "" {
		path = s.path + ".key"
	}
	// Never generate a replacement key for existing ciphertext after a restore.
	vault, err := loadVault(path, create && count == 0)
	if err != nil {
		return nil, err
	}
	s.vault = vault
	return vault, nil
}
