package dashboard

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

const refreshLeaseDuration = 30 * time.Second

func (s *Store) claimRefresh(source Source, key string, now time.Time) (token string, registered, claimed bool, err error) {
	var data []byte
	var paused int
	err = s.db.QueryRow("SELECT document,paused FROM sources WHERE id=? AND owner_id='local'", source.ID).Scan(&data, &paused)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, true, nil
	}
	if err != nil {
		return "", false, false, err
	}
	registered = true
	var stored Source
	if err = json.Unmarshal(data, &stored); err != nil {
		return
	}
	if paused != 0 || sourceKey(stored) != key {
		return "", true, false, nil
	}
	_, err = s.db.Exec("INSERT OR IGNORE INTO refresh_jobs(cache_key,source_id,next_run) VALUES(?,?,0)", key, source.ID)
	if err != nil {
		return
	}
	token = newID()
	var result sql.Result
	result, err = s.db.Exec("UPDATE refresh_jobs SET source_id=?,lease_token=?,lease_until=?,source_generation=(SELECT generation FROM sources WHERE id=?) WHERE cache_key=? AND lease_until<=? AND next_run<=?", source.ID, token, now.Add(refreshLeaseDuration).UnixNano(), source.ID, key, now.UnixNano(), now.UnixNano())
	if err != nil {
		return
	}
	var rows int64
	rows, err = result.RowsAffected()
	claimed = rows == 1
	return
}
func (s *Store) finishRefresh(key, token string, next time.Time) error {
	_, err := s.db.Exec("UPDATE refresh_jobs SET next_run=?,lease_until=0,lease_token='' WHERE cache_key=? AND lease_token=?", next.UnixNano(), key, token)
	return err
}
