package dashboard

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
)

const MaxBoards = 10

var ErrLimit = errors.New("보드는 최대 10개입니다")
var ErrInvalidInput = errors.New("입력값을 확인하세요")
var ErrProtectedBoard = errors.New("기본 보드는 삭제할 수 없습니다")

func (s *Store) GetBoard(id string) (Board, error) {
	var board Board
	var data []byte
	err := s.db.QueryRow("SELECT document FROM boards WHERE id=? AND owner_id='local'", id).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return board, ErrNotFound
	}
	if err != nil {
		return board, err
	}
	err = json.Unmarshal(data, &board)
	return board, err
}
func (s *Store) Boards() ([]Board, error) {
	rows, err := s.db.Query("SELECT document FROM boards WHERE owner_id='local' ORDER BY CASE WHEN id='home' THEN 0 ELSE 1 END,id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Board{}
	for rows.Next() {
		var data []byte
		var board Board
		if err = rows.Scan(&data); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(data, &board); err != nil {
			return nil, err
		}
		out = append(out, board)
	}
	return out, rows.Err()
}
func (s *Store) CreateBoard(title string, template *Board) (Board, error) {
	board := Board{ID: newID(), Title: strings.TrimSpace(title), Revision: 1, Theme: "light", Widgets: []Widget{}, Sources: []Source{}}
	if template != nil {
		board.Theme = template.Theme
		board.Widgets = append(board.Widgets, template.Widgets...)
		board.Sources = append(board.Sources, template.Sources...)
		for i := range board.Widgets {
			board.Widgets[i].ID = newID()
		}
	}
	if err := board.Validate(); err != nil {
		return Board{}, ErrInvalidInput
	}
	tx, err := s.db.Begin()
	if err != nil {
		return Board{}, err
	}
	defer tx.Rollback()
	var count int
	if err = tx.QueryRow("SELECT count(*) FROM boards WHERE owner_id='local'").Scan(&count); err != nil {
		return Board{}, err
	}
	if count >= MaxBoards {
		return Board{}, ErrLimit
	}
	data, err := json.Marshal(board)
	if err != nil {
		return Board{}, err
	}
	if _, err = tx.Exec("INSERT INTO boards(id,owner_id,revision,document) VALUES(?,'local',?,?)", board.ID, board.Revision, data); err != nil {
		return Board{}, err
	}
	if err = replaceSourceRefs(tx, board.ID, board.Sources); err != nil {
		return Board{}, err
	}
	if err = tx.Commit(); err != nil {
		return Board{}, err
	}
	return board, nil
}
func (s *Store) DeleteBoard(id string, revision int64) error {
	if id == "home" {
		return ErrProtectedBoard
	}
	result, err := s.db.Exec("DELETE FROM boards WHERE id=? AND owner_id='local' AND revision=?", id, revision)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		if _, err = s.GetBoard(id); errors.Is(err, ErrNotFound) {
			return ErrNotFound
		}
		return ErrConflict
	}
	return nil
}
