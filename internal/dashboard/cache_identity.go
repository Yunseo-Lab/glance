package dashboard

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

type rowQuerier interface{ QueryRow(string, ...any) *sql.Row }

func sourceKeyForGeneration(source Source, generation int) string {
	if source.ConnectionID == "" {
		return sourceKey(source)
	}
	digest := sha256.Sum256([]byte(fmt.Sprintf("%s:generation:%d", sourceKey(source), generation)))
	return hex.EncodeToString(digest[:])
}
func cacheKeyFrom(query rowQuerier, source Source) (string, error) {
	if source.ConnectionID == "" {
		return sourceKey(source), nil
	}
	var generation int
	if err := query.QueryRow("SELECT generation FROM connections WHERE id=? AND owner_id='local'", source.ConnectionID).Scan(&generation); err != nil {
		return "", err
	}
	return sourceKeyForGeneration(source, generation), nil
}
func (s *Store) cacheKey(source Source) (string, error) { return cacheKeyFrom(s.db, source) }

func clearConnectionCaches(tx *sql.Tx, id string, generation int) error {
	rows, err := tx.Query("SELECT document FROM sources WHERE connection_id=? AND owner_id='local'", id)
	if err != nil {
		return err
	}
	var keys []string
	for rows.Next() {
		var document []byte
		var source Source
		if err = rows.Scan(&document); err != nil {
			rows.Close()
			return err
		}
		if err = json.Unmarshal(document, &source); err != nil {
			rows.Close()
			return err
		}
		keys = append(keys, sourceKey(source), sourceKeyForGeneration(source, generation))
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, table := range []string{"snapshots", "http_validators"} {
		if _, err = tx.Exec("DELETE FROM "+table+" WHERE cache_key IN(SELECT cache_key FROM refresh_jobs WHERE source_id IN(SELECT id FROM sources WHERE connection_id=?))", id); err != nil {
			return err
		}
		for _, key := range keys {
			if _, err = tx.Exec("DELETE FROM "+table+" WHERE cache_key=?", key); err != nil {
				return err
			}
		}
	}
	_, err = tx.Exec("DELETE FROM refresh_jobs WHERE source_id IN(SELECT id FROM sources WHERE connection_id=? AND owner_id='local')", id)
	return err
}
