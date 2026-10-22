package dashboard

import (
	"database/sql"
	"errors"
	"net/url"
	"strings"
	"time"
)

const MaxConnections = 32

type ConnectionInput struct {
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Endpoint string `json:"endpoint"`
	Token    string `json:"token,omitempty"`
}

func (s *Store) connection(id string) (Connection, error) {
	row := s.db.QueryRow("SELECT id,kind,name,public_config,length(encrypted_secret)>0,generation,status,created_at,last_tested_at FROM connections WHERE id=? AND owner_id='local'", id)
	connection, err := scanConnection(row)
	if errors.Is(err, sql.ErrNoRows) {
		return connection, ErrNotFound
	}
	return connection, err
}
func (s *Store) CreateConnection(input ConnectionInput) (Connection, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" || len(input.Name) > 100 || len(input.Token) > 4096 || strings.ContainsAny(input.Token, "\r\n") {
		return Connection{}, ErrInvalidInput
	}
	secret := connectionSecret{Token: input.Token}
	endpoint := input.Endpoint
	switch input.Kind {
	case "github":
		if len(input.Token) < 16 {
			return Connection{}, ErrInvalidInput
		}
		if endpoint != "" && endpoint != "https://api.github.com" {
			return Connection{}, ErrInvalidInput
		}
		endpoint = "https://api.github.com"
	case "calendar", "json":
		u, err := publicURL(endpoint)
		if err != nil {
			return Connection{}, ErrInvalidInput
		}
		if u.Fragment != "" {
			return Connection{}, ErrInvalidInput
		}
		secret.URL = endpoint
		endpoint = (&url.URL{Scheme: u.Scheme, Host: u.Host}).String()
	default:
		return Connection{}, ErrInvalidInput
	}
	vault, err := s.connectionVault(true)
	if err != nil {
		return Connection{}, err
	}
	id := newID()
	ciphertext, err := vault.encrypt(id, input.Kind, secret)
	if err != nil {
		return Connection{}, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return Connection{}, err
	}
	defer tx.Rollback()
	var count int
	if err = tx.QueryRow("SELECT count(*) FROM connections WHERE owner_id='local'").Scan(&count); err != nil {
		return Connection{}, err
	}
	if count >= MaxConnections {
		return Connection{}, ErrInvalidInput
	}
	public := raw(connectionPublic{Endpoint: endpoint})
	created := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err = tx.Exec("INSERT INTO connections(id,owner_id,kind,public_config,encrypted_secret,key_version,name,created_at) VALUES(?,'local',?,?,?,?,?,?)", id, input.Kind, public, ciphertext, vault.version, input.Name, created); err != nil {
		return Connection{}, err
	}
	if err = tx.Commit(); err != nil {
		return Connection{}, err
	}
	return s.connection(id)
}
func (s *Store) resolveConnection(id string) (Connection, connectionSecret, error) {
	connection, err := s.connection(id)
	if err != nil {
		return connection, connectionSecret{}, err
	}
	if connection.Status == "revoked" || !connection.HasSecret {
		return connection, connectionSecret{}, ErrInvalidInput
	}
	var ciphertext []byte
	if err = s.db.QueryRow("SELECT encrypted_secret FROM connections WHERE id=? AND owner_id='local'", id).Scan(&ciphertext); err != nil {
		return connection, connectionSecret{}, err
	}
	vault, err := s.connectionVault(false)
	if err != nil {
		return connection, connectionSecret{}, err
	}
	secret, err := vault.decrypt(id, connection.Kind, ciphertext)
	return connection, secret, err
}
func (s *Store) RevokeConnection(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var previousGeneration int
	if err = tx.QueryRow("SELECT generation FROM connections WHERE id=? AND owner_id='local'", id).Scan(&previousGeneration); errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	if err = clearConnectionCaches(tx, id, previousGeneration); err != nil {
		return err
	}
	result, err := tx.Exec("UPDATE connections SET encrypted_secret=NULL,status='revoked',generation=generation+1 WHERE id=? AND owner_id='local'", id)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return ErrNotFound
	}
	if _, err = tx.Exec("UPDATE sources SET paused=1,generation=generation+1 WHERE connection_id=? AND owner_id='local'", id); err != nil {
		return err
	}
	if _, err = tx.Exec("UPDATE refresh_jobs SET lease_token='',lease_until=0 WHERE source_id IN(SELECT id FROM sources WHERE connection_id=? AND owner_id='local')", id); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *Store) DeleteConnection(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var referenced int
	if err = tx.QueryRow("SELECT count(*) FROM sources WHERE connection_id=?", id).Scan(&referenced); err != nil {
		return err
	}
	if referenced > 0 {
		return ErrInUse
	}
	result, err := tx.Exec("DELETE FROM connections WHERE id=? AND owner_id='local'", id)
	if err != nil {
		return err
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		return ErrNotFound
	}
	return tx.Commit()
}
