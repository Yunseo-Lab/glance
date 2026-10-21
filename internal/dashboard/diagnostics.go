package dashboard

import (
	"database/sql"
	"errors"
	"time"
)

type SourceStatus struct {
	Source       Source     `json:"source"`
	BoardIDs     []string   `json:"boardIds"`
	Paused       bool       `json:"paused"`
	Generation   int        `json:"generation"`
	Snapshot     Snapshot   `json:"snapshot"`
	Running      bool       `json:"running"`
	NextRunAt    *time.Time `json:"nextRunAt,omitempty"`
	NextManualAt *time.Time `json:"nextManualAt,omitempty"`
}

func (s *Store) sourcePaused(id string) (bool, bool, error) {
	var paused int
	err := s.db.QueryRow("SELECT paused FROM sources WHERE id=? AND owner_id='local'", id).Scan(&paused)
	if errors.Is(err, sql.ErrNoRows) {
		return false, false, nil
	}
	return paused != 0, err == nil, err
}
func (s *Store) SourceStatuses(now time.Time) ([]SourceStatus, error) {
	sources, err := s.Sources()
	if err != nil {
		return nil, err
	}
	out := []SourceStatus{}
	for _, source := range sources {
		status := SourceStatus{Source: source, BoardIDs: []string{}}
		var paused int
		err = s.db.QueryRow("SELECT paused,generation FROM sources WHERE id=? AND owner_id='local'", source.ID).Scan(&paused, &status.Generation)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return nil, err
		}
		status.Paused = paused != 0
		rows, err := s.db.Query("SELECT r.board_id FROM board_sources r JOIN boards b ON b.id=r.board_id WHERE r.source_id=? AND b.owner_id='local' ORDER BY r.board_id", source.ID)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var id string
			if err = rows.Scan(&id); err != nil {
				rows.Close()
				return nil, err
			}
			status.BoardIDs = append(status.BoardIDs, id)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, err
		}
		key := sourceKey(source)
		snapshot, err := s.snapshot(key)
		if err != nil {
			return nil, err
		}
		status.Snapshot = snapshotState(snapshot, now)
		status.Snapshot.Data = nil
		if status.Paused {
			status.Snapshot.State = "paused"
		}
		var next, lease, manual int64
		err = s.db.QueryRow("SELECT next_run,lease_until,manual_next_allowed FROM refresh_jobs WHERE cache_key=?", key).Scan(&next, &lease, &manual)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		status.Running = lease > now.UnixNano()
		if next > 0 {
			date := time.Unix(0, next)
			status.NextRunAt = &date
		}
		if manual > 0 {
			date := time.Unix(0, manual)
			status.NextManualAt = &date
		}
		out = append(out, status)
	}
	return out, nil
}
func (s *Store) PauseSource(id string, paused bool) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var before bool
	err = tx.QueryRow("SELECT paused FROM sources WHERE id=? AND owner_id='local' AND EXISTS(SELECT 1 FROM board_sources r WHERE r.source_id=sources.id)", id).Scan(&before)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if before == paused {
		return nil
	}
	if _, err = tx.Exec("UPDATE sources SET paused=?,generation=generation+1 WHERE id=? AND owner_id='local'", paused, id); err != nil {
		return err
	}
	if _, err = tx.Exec("UPDATE refresh_jobs SET lease_until=0,lease_token='',next_run=0 WHERE source_id=?", id); err != nil {
		return err
	}
	return tx.Commit()
}
