package dashboard

import (
	"database/sql"
	"encoding/json"
)

// Connection is the only connection response type. Ciphertext and credentials
// are deliberately absent; request secrets use a separate private type.
type Connection struct {
	ID           string `json:"id"`
	Kind         string `json:"kind"`
	Name         string `json:"name"`
	Endpoint     string `json:"endpoint"`
	HasSecret    bool   `json:"hasSecret"`
	Generation   int    `json:"generation"`
	Status       string `json:"status"`
	CreatedAt    string `json:"createdAt"`
	LastTestedAt string `json:"lastTestedAt,omitempty"`
}
type connectionSecret struct {
	Token string `json:"token,omitempty"`
	URL   string `json:"url,omitempty"`
}
type connectionPublic struct {
	Endpoint string `json:"endpoint"`
}

func (s *Store) Connections() ([]Connection, error) {
	rows, err := s.db.Query("SELECT id,kind,name,public_config,length(encrypted_secret)>0,generation,status,created_at,last_tested_at FROM connections WHERE owner_id='local' ORDER BY created_at,id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Connection{}
	for rows.Next() {
		connection, err := scanConnection(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, connection)
	}
	return out, rows.Err()
}

type connectionScanner interface{ Scan(...any) error }

func scanConnection(row connectionScanner) (Connection, error) {
	var out Connection
	var config []byte
	var hasSecret sql.NullBool
	err := row.Scan(&out.ID, &out.Kind, &out.Name, &config, &hasSecret, &out.Generation, &out.Status, &out.CreatedAt, &out.LastTestedAt)
	if err != nil {
		return out, err
	}
	var public connectionPublic
	if err = json.Unmarshal(config, &public); err != nil {
		return out, err
	}
	out.Endpoint = public.Endpoint
	out.HasSecret = hasSecret.Valid && hasSecret.Bool
	return out, nil
}
