package dashboard

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

type migration struct{ name, sql string }

// Never edit an applied migration. Add a new one; checksums detect drift.
var migrations = []migration{
	{"initial dashboard", `CREATE TABLE IF NOT EXISTS boards(id TEXT PRIMARY KEY,owner_id TEXT NOT NULL,revision INTEGER NOT NULL,document BLOB NOT NULL);
CREATE TABLE IF NOT EXISTS snapshots(cache_key TEXT PRIMARY KEY,document BLOB NOT NULL);`},
	{"migration history and owner index", `CREATE INDEX IF NOT EXISTS boards_by_owner ON boards(owner_id,id);`},
	{"shared sources and reading storage", `CREATE TABLE connections(id TEXT PRIMARY KEY,owner_id TEXT NOT NULL,kind TEXT NOT NULL,public_config BLOB NOT NULL,encrypted_secret BLOB,key_version INTEGER NOT NULL DEFAULT 1);
CREATE TABLE sources(id TEXT PRIMARY KEY,owner_id TEXT NOT NULL,document BLOB NOT NULL,connection_id TEXT REFERENCES connections(id),generation INTEGER NOT NULL DEFAULT 1,paused INTEGER NOT NULL DEFAULT 0,updated_at TEXT NOT NULL DEFAULT '');
CREATE TABLE board_sources(board_id TEXT NOT NULL REFERENCES boards(id) ON DELETE CASCADE,source_id TEXT NOT NULL REFERENCES sources(id),PRIMARY KEY(board_id,source_id));
CREATE INDEX board_sources_by_source ON board_sources(source_id);
CREATE TABLE entries(id TEXT PRIMARY KEY,owner_id TEXT NOT NULL,source_id TEXT REFERENCES sources(id) ON DELETE SET NULL,item_key TEXT NOT NULL,title TEXT NOT NULL,url TEXT NOT NULL,excerpt TEXT NOT NULL DEFAULT '',published_at TEXT,collected_at TEXT NOT NULL,UNIQUE(owner_id,source_id,item_key));
CREATE INDEX entries_by_owner_time ON entries(owner_id,collected_at DESC,id);
CREATE TABLE entry_state(owner_id TEXT NOT NULL,entry_id TEXT NOT NULL REFERENCES entries(id) ON DELETE CASCADE,is_read INTEGER NOT NULL DEFAULT 0,is_saved INTEGER NOT NULL DEFAULT 0,tags BLOB NOT NULL DEFAULT '[]',updated_at TEXT NOT NULL,PRIMARY KEY(owner_id,entry_id));`},
	{"bounded source and reading query indexes", `CREATE INDEX sources_by_owner ON sources(owner_id,id);
CREATE INDEX entries_by_source ON entries(source_id);
CREATE INDEX saved_entries_by_owner ON entry_state(owner_id,is_saved,entry_id);`},
	{"persistent refresh leases", `CREATE TABLE refresh_jobs(cache_key TEXT PRIMARY KEY,source_id TEXT NOT NULL REFERENCES sources(id) ON DELETE CASCADE,next_run INTEGER NOT NULL,lease_until INTEGER NOT NULL DEFAULT 0,lease_token TEXT NOT NULL DEFAULT '');
CREATE INDEX due_refresh_jobs ON refresh_jobs(next_run,lease_until);`},
	{"HTTP conditional request validators", `CREATE TABLE http_validators(cache_key TEXT PRIMARY KEY REFERENCES snapshots(cache_key) ON DELETE CASCADE,etag TEXT NOT NULL,last_modified TEXT NOT NULL,updated_at TEXT NOT NULL);`},
	{"refresh source generation fence", `ALTER TABLE refresh_jobs ADD COLUMN source_generation INTEGER NOT NULL DEFAULT 1;`},
	{"manual refresh budgets", `ALTER TABLE refresh_jobs ADD COLUMN manual_next_allowed INTEGER NOT NULL DEFAULT 0;
CREATE TABLE manual_refresh_log(id INTEGER PRIMARY KEY,requested_at INTEGER NOT NULL);`},
	{"safe connection metadata", `ALTER TABLE connections ADD COLUMN name TEXT NOT NULL DEFAULT '';
ALTER TABLE connections ADD COLUMN generation INTEGER NOT NULL DEFAULT 1;
ALTER TABLE connections ADD COLUMN status TEXT NOT NULL DEFAULT 'unverified';
ALTER TABLE connections ADD COLUMN created_at TEXT NOT NULL DEFAULT '';
ALTER TABLE connections ADD COLUMN last_tested_at TEXT NOT NULL DEFAULT '';`},
	{"reading provenance and normalized order", `ALTER TABLE entries ADD COLUMN identity_scope TEXT NOT NULL DEFAULT 'public';
ALTER TABLE entries ADD COLUMN source_title TEXT NOT NULL DEFAULT '';
ALTER TABLE entries ADD COLUMN sort_at INTEGER NOT NULL DEFAULT 0;
UPDATE entries SET sort_at=COALESCE(CAST(strftime('%s',published_at) AS INTEGER),CAST(strftime('%s',collected_at) AS INTEGER),0)*1000000000;
CREATE TABLE entry_sources(entry_id TEXT NOT NULL REFERENCES entries(id) ON DELETE CASCADE,source_id TEXT NOT NULL REFERENCES sources(id) ON DELETE CASCADE,item_key TEXT NOT NULL,PRIMARY KEY(entry_id,source_id),UNIQUE(source_id,item_key));
INSERT INTO entry_sources(entry_id,source_id,item_key) SELECT id,source_id,item_key FROM entries WHERE source_id IS NOT NULL;
CREATE INDEX entry_sources_by_source ON entry_sources(source_id,entry_id);
CREATE INDEX entries_by_owner_order ON entries(owner_id,sort_at DESC,id DESC);
CREATE INDEX entries_by_owner_url ON entries(owner_id,identity_scope,url);`},
}

func schemaVersion() int { return len(migrations) }
func migrationChecksum(m migration) string {
	hash := sha256.Sum256([]byte(m.name + "\n" + m.sql))
	return hex.EncodeToString(hash[:])
}
func applyMigrations(db *sql.DB, steps []migration) error {
	var version int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return err
	}
	if version < 0 || version > len(steps) {
		return fmt.Errorf("unsupported database version %d", version)
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations(version INTEGER PRIMARY KEY,name TEXT NOT NULL,checksum TEXT NOT NULL,applied_at TEXT NOT NULL);`); err != nil {
		return err
	}
	// Original schema v1 had no history table. Its recognized baseline is recorded
	// only inside the same atomic transaction as the first upgrade.
	for i := 0; i < version; i++ {
		var sum string
		err = tx.QueryRow("SELECT checksum FROM schema_migrations WHERE version=?", i+1).Scan(&sum)
		if errors.Is(err, sql.ErrNoRows) && version == 1 {
			_, err = tx.Exec("INSERT INTO schema_migrations VALUES(?,?,?,?)", i+1, steps[i].name, migrationChecksum(steps[i]), "legacy-v1")
		} else if err == nil && sum != migrationChecksum(steps[i]) {
			return fmt.Errorf("migration checksum mismatch at version %d", i+1)
		}
		if err != nil {
			return err
		}
	}
	for i := version; i < len(steps); i++ {
		if _, err = tx.Exec(steps[i].sql); err != nil {
			return fmt.Errorf("migration %d failed: %w", i+1, err)
		}
		if i+1 == 3 {
			if err = normalizeSources(tx); err != nil {
				return err
			}
		}
		if _, err = tx.Exec("INSERT INTO schema_migrations VALUES(?,?,?,?)", i+1, steps[i].name, migrationChecksum(steps[i]), time.Now().UTC().Format(time.RFC3339)); err != nil {
			return err
		}
		if _, err = tx.Exec(fmt.Sprintf("PRAGMA user_version=%d", i+1)); err != nil {
			return err
		}
	}
	return tx.Commit()
}
