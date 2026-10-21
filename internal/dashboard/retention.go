package dashboard

import (
	"database/sql"
	"errors"
	"time"
)

var ErrInUse = errors.New("다른 항목에서 사용 중입니다. 먼저 참조를 제거하세요")

func purgeUnusedSources(tx *sql.Tx) error {
	_, err := tx.Exec("DELETE FROM sources WHERE owner_id='local' AND NOT EXISTS(SELECT 1 FROM board_sources r WHERE r.source_id=sources.id)")
	return err
}
func (s *Store) RetainData(now time.Time) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec("DELETE FROM entries WHERE owner_id='local' AND collected_at<? AND NOT EXISTS(SELECT 1 FROM entry_state st WHERE st.entry_id=entries.id AND st.owner_id='local' AND st.is_saved=1)", now.AddDate(0, 0, -EntryRetentionDays).UTC().Format(time.RFC3339Nano)); err != nil {
		return err
	}
	if err = purgeUnusedSources(tx); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	return s.pruneSnapshots(nil)
}
func (s *Store) DeleteSource(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var count int
	if err = tx.QueryRow("SELECT count(*) FROM sources WHERE id=? AND owner_id='local'", id).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		return ErrNotFound
	}
	if err = tx.QueryRow("SELECT count(*) FROM board_sources WHERE source_id=?", id).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return ErrInUse
	}
	if _, err = tx.Exec("DELETE FROM sources WHERE id=? AND owner_id='local'", id); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *Store) deleteBoard(id string, revision int64) error {
	if id == "home" {
		return ErrProtectedBoard
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.Exec("DELETE FROM boards WHERE id=? AND owner_id='local' AND revision=?", id, revision)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		var exists int
		err = tx.QueryRow("SELECT 1 FROM boards WHERE id=? AND owner_id='local'", id).Scan(&exists)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		return ErrConflict
	}
	if err = purgeUnusedSources(tx); err != nil {
		return err
	}
	return tx.Commit()
}
