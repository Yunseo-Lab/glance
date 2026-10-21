package dashboard

import (
	"database/sql"
	"encoding/json"
	"errors"
)

var ErrSourceConflict = errors.New("데이터 소스가 변경되었습니다. 새 소스로 저장하거나 다시 불러오세요")

type boardSources struct {
	boardID string
	sources []Source
}

func normalizeSources(tx *sql.Tx) error {
	rows, err := tx.Query("SELECT id,document FROM boards WHERE owner_id='local'")
	if err != nil {
		return err
	}
	var boards []boardSources
	for rows.Next() {
		var id string
		var data []byte
		var board Board
		if err = rows.Scan(&id, &data); err != nil {
			rows.Close()
			return err
		}
		if err = json.Unmarshal(data, &board); err != nil {
			rows.Close()
			return err
		}
		boards = append(boards, boardSources{id, board.Sources})
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, board := range boards {
		if err = replaceSourceRefs(tx, board.boardID, board.sources); err != nil {
			return err
		}
	}
	return nil
}
func replaceSourceRefs(tx *sql.Tx, boardID string, sources []Source) error {
	if _, err := tx.Exec("DELETE FROM board_sources WHERE board_id=?", boardID); err != nil {
		return err
	}
	for _, source := range sources {
		var previous []byte
		err := tx.QueryRow("SELECT document FROM sources WHERE id=? AND owner_id='local'", source.ID).Scan(&previous)
		if err == nil {
			var stored Source
			if err = json.Unmarshal(previous, &stored); err != nil {
				return err
			}
			// A changed query gets a new source ID. Shared queries cannot silently change
			// under another board's independent revision.
			if sourceKey(stored) != sourceKey(source) {
				return ErrSourceConflict
			}
		} else if errors.Is(err, sql.ErrNoRows) {
			document, err := json.Marshal(source)
			if err != nil {
				return err
			}
			if _, err = tx.Exec("INSERT INTO sources(id,owner_id,document) VALUES(?,'local',?)", source.ID, document); err != nil {
				return err
			}
		} else {
			return err
		}
		if _, err = tx.Exec("INSERT INTO board_sources(board_id,source_id) VALUES(?,?)", boardID, source.ID); err != nil {
			return err
		}
	}
	var active int
	if err := tx.QueryRow("SELECT count(DISTINCT r.source_id) FROM board_sources r JOIN boards b ON b.id=r.board_id WHERE b.owner_id='local'").Scan(&active); err != nil {
		return err
	}
	if active > MaxActiveSources {
		return ErrSourceLimit
	}
	// Removing a layout reference never deletes an owner's saved entry. SQLite
	// preserves entries and state while dropping only the orphan source pointer.
	_, err := tx.Exec("DELETE FROM sources WHERE owner_id='local' AND NOT EXISTS(SELECT 1 FROM board_sources r WHERE r.source_id=sources.id)")
	return err
}
func (s *Store) Sources() ([]Source, error) {
	rows, err := s.db.Query("SELECT s.document FROM sources s WHERE s.owner_id='local' AND EXISTS(SELECT 1 FROM board_sources r JOIN boards b ON b.id=r.board_id WHERE r.source_id=s.id AND b.owner_id='local') ORDER BY s.id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Source{}
	for rows.Next() {
		var data []byte
		var source Source
		if err = rows.Scan(&data); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(data, &source); err != nil {
			return nil, err
		}
		out = append(out, source)
	}
	return out, rows.Err()
}
