package dashboard

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

var ErrConflict = errors.New("다른 창에서 대시보드가 변경되었습니다. 새로고침 후 다시 편집하세요")

type Store struct{ db *sql.DB }

func OpenStore(path string) (*Store, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if err = os.MkdirAll(filepath.Dir(abs), 0700); err != nil {
		return nil, err
	}
	// Pre-create privately; SQLite inherits these permissions for WAL files.
	f, err := os.OpenFile(abs, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err = f.Close(); err != nil {
		return nil, err
	}
	u := url.URL{Scheme: "file", Path: abs, RawQuery: "_busy_timeout=5000&_journal_mode=WAL&_foreign_keys=on"}
	db, err := sql.Open("sqlite3", u.String())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &Store{db: db}
	if err = s.init(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) init() error {
	var version int
	if err := s.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return err
	}
	if version > 1 {
		return fmt.Errorf("unsupported database version %d", version)
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.Exec(`CREATE TABLE IF NOT EXISTS boards(id TEXT PRIMARY KEY,owner_id TEXT NOT NULL,revision INTEGER NOT NULL,document BLOB NOT NULL);
CREATE TABLE IF NOT EXISTS snapshots(cache_key TEXT PRIMARY KEY,document BLOB NOT NULL);
PRAGMA user_version=1;`)
	if err != nil {
		return err
	}
	b := defaultBoard()
	data, err := json.Marshal(b)
	if err != nil {
		return err
	}
	if _, err = tx.Exec("INSERT OR IGNORE INTO boards VALUES(?,?,?,?)", b.ID, "local", b.Revision, data); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) Close() error { return s.db.Close() }
func (s *Store) Board() (Board, error) {
	var b Board
	var data []byte
	err := s.db.QueryRow("SELECT document FROM boards WHERE id='home' AND owner_id='local'").Scan(&data)
	if err != nil {
		return b, err
	}
	err = json.Unmarshal(data, &b)
	return b, err
}
func (s *Store) Save(b Board) (Board, error) {
	if err := b.Validate(); err != nil {
		return b, err
	}
	expected := b.Revision
	b.Revision++
	data, err := json.Marshal(b)
	if err != nil {
		return b, err
	}
	res, err := s.db.Exec("UPDATE boards SET revision=?,document=? WHERE id=? AND owner_id='local' AND revision=?", b.Revision, data, b.ID, expected)
	if err != nil {
		return b, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return b, err
	}
	if n != 1 {
		return b, ErrConflict
	}
	return b, nil
}
func (s *Store) snapshot(key string) (Snapshot, error) {
	var snap Snapshot
	var data []byte
	err := s.db.QueryRow("SELECT document FROM snapshots WHERE cache_key=?", key).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return snap, nil
	}
	if err != nil {
		return snap, err
	}
	err = json.Unmarshal(data, &snap)
	return snap, err
}
func (s *Store) putSnapshot(key string, snap Snapshot) error {
	data, err := json.Marshal(snap)
	if err != nil {
		return err
	}
	_, err = s.db.Exec("INSERT INTO snapshots(cache_key,document) VALUES(?,?) ON CONFLICT(cache_key) DO UPDATE SET document=excluded.document", key, data)
	return err
}

// Source configurations are public in this first, single-owner release.
// On save, discard orphaned caches so deleted feeds do not accumulate forever.
func (s *Store) pruneSnapshots(sources []Source) error {
	keys := map[string]bool{}
	for _, source := range sources {
		keys[sourceKey(source)] = true
	}
	rows, err := s.db.Query("SELECT cache_key FROM snapshots")
	if err != nil {
		return err
	}
	var remove []string
	for rows.Next() {
		var key string
		if err = rows.Scan(&key); err != nil {
			rows.Close()
			return err
		}
		if !keys[key] {
			remove = append(remove, key)
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, key := range remove {
		if _, err = s.db.Exec("DELETE FROM snapshots WHERE cache_key=?", key); err != nil {
			return err
		}
	}
	return nil
}
