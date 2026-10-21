package dashboard

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

type CooldownError struct{ Until time.Time }

func (e *CooldownError) Error() string {
	return "잠시 후 다시 시도해 주세요. 정보 제공자의 대기 시간과 새로고침 한도를 적용합니다."
}

type RefreshReceipt struct {
	Accepted      bool      `json:"accepted"`
	NextAllowedAt time.Time `json:"nextAllowedAt"`
}

func (s *Store) RequestRefresh(id string, now time.Time) (Source, RefreshReceipt, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return Source{}, RefreshReceipt{}, err
	}
	defer tx.Rollback()
	var document []byte
	var paused int
	err = tx.QueryRow("SELECT document,paused FROM sources WHERE id=? AND owner_id='local' AND EXISTS(SELECT 1 FROM board_sources r WHERE r.source_id=sources.id)", id).Scan(&document, &paused)
	if errors.Is(err, sql.ErrNoRows) {
		return Source{}, RefreshReceipt{}, ErrNotFound
	}
	if err != nil {
		return Source{}, RefreshReceipt{}, err
	}
	if paused != 0 {
		return Source{}, RefreshReceipt{}, ErrInvalidInput
	}
	var source Source
	if err = json.Unmarshal(document, &source); err != nil {
		return source, RefreshReceipt{}, err
	}
	key := sourceKey(source)
	if _, err = tx.Exec("INSERT OR IGNORE INTO refresh_jobs(cache_key,source_id,next_run) VALUES(?,?,0)", key, source.ID); err != nil {
		return source, RefreshReceipt{}, err
	}
	var next int64
	if err = tx.QueryRow("SELECT manual_next_allowed FROM refresh_jobs WHERE cache_key=?", key).Scan(&next); err != nil {
		return source, RefreshReceipt{}, err
	}
	if next > now.UnixNano() {
		return source, RefreshReceipt{}, &CooldownError{Until: time.Unix(0, next)}
	}
	var snapshot Snapshot
	var data []byte
	err = tx.QueryRow("SELECT document FROM snapshots WHERE cache_key=?", key).Scan(&data)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return source, RefreshReceipt{}, err
	}
	if len(data) > 0 {
		if err = json.Unmarshal(data, &snapshot); err != nil {
			return source, RefreshReceipt{}, err
		}
	}
	if snapshot.RetryAt != nil && snapshot.RetryAt.After(now) {
		return source, RefreshReceipt{}, &CooldownError{Until: *snapshot.RetryAt}
	}
	if _, err = tx.Exec("DELETE FROM manual_refresh_log WHERE requested_at<=?", now.Add(-time.Minute).UnixNano()); err != nil {
		return source, RefreshReceipt{}, err
	}
	var count int
	var oldest sql.NullInt64
	if err = tx.QueryRow("SELECT count(*),min(requested_at) FROM manual_refresh_log").Scan(&count, &oldest); err != nil {
		return source, RefreshReceipt{}, err
	}
	if count >= 10 {
		return source, RefreshReceipt{}, &CooldownError{Until: time.Unix(0, oldest.Int64).Add(time.Minute)}
	}
	if _, err = tx.Exec("INSERT INTO manual_refresh_log(requested_at) VALUES(?)", now.UnixNano()); err != nil {
		return source, RefreshReceipt{}, err
	}
	nextAllowed := now.Add(30 * time.Second)
	if _, err = tx.Exec("UPDATE refresh_jobs SET manual_next_allowed=?,next_run=0 WHERE cache_key=?", nextAllowed.UnixNano(), key); err != nil {
		return source, RefreshReceipt{}, err
	}
	snapshot.ExpiresAt = &now
	snapshot.RetryAt = nil
	snapshot.Error = ""
	snapshot.ErrorCode = ""
	snapshot.State = "loading"
	data, err = json.Marshal(snapshot)
	if err != nil {
		return source, RefreshReceipt{}, err
	}
	if _, err = tx.Exec("INSERT INTO snapshots(cache_key,document) VALUES(?,?) ON CONFLICT(cache_key) DO UPDATE SET document=excluded.document", key, data); err != nil {
		return source, RefreshReceipt{}, err
	}
	if err = tx.Commit(); err != nil {
		return source, RefreshReceipt{}, err
	}
	return source, RefreshReceipt{Accepted: true, NextAllowedAt: nextAllowed}, nil
}

func (s *Store) nextManualRefresh(key string) (*time.Time, error) {
	var until int64
	err := s.db.QueryRow("SELECT manual_next_allowed FROM refresh_jobs WHERE cache_key=?", key).Scan(&until)
	if errors.Is(err, sql.ErrNoRows) || err == nil && until <= 0 {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	date := time.Unix(0, until)
	return &date, nil
}
