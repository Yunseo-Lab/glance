package dashboard

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"
)

type IngestionResult struct {
	Added   int
	Updated int
	Skipped int
}

// Called inside the same transaction and source-generation fence as snapshots.
// Repeated 304s, overlapping feeds and restarts preserve owner state and IDs.
func ingestFeedEntries(tx *sql.Tx, source Source, body json.RawMessage, now time.Time) (IngestionResult, error) {
	result := IngestionResult{}
	if source.Type != "rss" && source.Type != "github" {
		return result, nil
	}
	var feed FeedData
	if len(body) > MaxProviderBody || json.Unmarshal(body, &feed) != nil {
		return result, ErrInvalidInput
	}
	if len(feed.Items) > 50 {
		feed.Items = feed.Items[:50]
	}
	var count int
	if err := tx.QueryRow("SELECT count(*) FROM entries WHERE owner_id='local'").Scan(&count); err != nil {
		return result, err
	}
	scope := "public"
	if source.ConnectionID != "" {
		scope = "connection:" + source.ConnectionID
	}
	for _, item := range feed.Items {
		address, err := canonicalArticleURL(item.URL, "")
		if err != nil {
			result.Skipped++
			continue
		}
		key := item.ID
		if key == "" {
			key = address
		}
		var id string
		err = tx.QueryRow("SELECT entry_id FROM entry_sources WHERE source_id=? AND item_key=?", source.ID, key).Scan(&id)
		if errors.Is(err, sql.ErrNoRows) {
			if lookupErr := tx.QueryRow("SELECT id FROM entries WHERE owner_id='local' AND identity_scope=? AND url=? LIMIT 1", scope, address).Scan(&id); errors.Is(lookupErr, sql.ErrNoRows) {
				digest := sha256.Sum256([]byte("local:" + scope + ":" + address))
				id = hex.EncodeToString(digest[:])
			} else if lookupErr != nil {
				return result, lookupErr
			}
		} else if err != nil {
			return result, err
		}
		var exists int
		if err = tx.QueryRow("SELECT count(*) FROM entries WHERE id=? AND owner_id='local'", id).Scan(&exists); err != nil {
			return result, err
		}
		if exists == 0 && count >= MaxEntries {
			result.Skipped++
			continue
		}
		var published any
		sortAt := now.UnixNano()
		if date := item.PublishedAt; date != nil && date.Year() >= 1900 && date.Year() <= 2100 && !date.After(now.Add(24*time.Hour)) {
			published = date.UTC().Format(time.RFC3339Nano)
			sortAt = date.UnixNano()
		}
		title := plainText(item.Title, 300)
		if title == "" {
			title = address
		}
		excerpt := plainText(item.Excerpt, 600)
		if exists != 0 {
			if _, err = tx.Exec("UPDATE entries SET title=?,url=?,excerpt=?,published_at=COALESCE(?,published_at),source_title=?,sort_at=CASE WHEN ? IS NULL THEN sort_at ELSE ? END,source_id=COALESCE(source_id,?) WHERE id=? AND owner_id='local'", title, address, excerpt, published, plainText(feed.Title, 200), published, sortAt, source.ID, id); err != nil {
				return result, err
			}
			result.Updated++
		} else {
			if _, err = tx.Exec("INSERT INTO entries(id,owner_id,source_id,item_key,title,url,excerpt,published_at,collected_at,identity_scope,source_title,sort_at) VALUES(?,'local',?,?,?,?,?,?,?,?,?,?)", id, source.ID, key, title, address, excerpt, published, now.UTC().Format(time.RFC3339Nano), scope, plainText(feed.Title, 200), sortAt); err != nil {
				return result, err
			}
			count++
			result.Added++
		}
		if _, err = tx.Exec("INSERT INTO entry_sources(entry_id,source_id,item_key) VALUES(?,?,?) ON CONFLICT(entry_id,source_id) DO UPDATE SET item_key=excluded.item_key ON CONFLICT(source_id,item_key) DO UPDATE SET entry_id=excluded.entry_id", id, source.ID, key); err != nil {
			return result, err
		}
	}
	return result, nil
}
