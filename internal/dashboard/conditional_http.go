package dashboard

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

type HTTPValidators struct {
	ETag         string
	LastModified string
}
type FetchResult struct {
	Data        any
	NotModified bool
	Validators  *HTTPValidators
}
type conditionalKey struct{}
type conditionalRequest struct {
	previous HTTPValidators
	received HTTPValidators
}

var errNotModified = errors.New("provider not modified")

func (s *Store) validators(key string) (HTTPValidators, error) {
	var validators HTTPValidators
	err := s.db.QueryRow("SELECT etag,last_modified FROM http_validators WHERE cache_key=?", key).Scan(&validators.ETag, &validators.LastModified)
	if errors.Is(err, sql.ErrNoRows) {
		return validators, nil
	}
	return validators, err
}
func (p *Providers) FetchCached(ctx context.Context, source Source) (any, error) {
	if p.store == nil || source.Type == "ical" {
		return p.Fetch(ctx, source)
	}
	key, err := p.store.cacheKey(source)
	if err != nil {
		return nil, err
	}
	previous, err := p.store.validators(key)
	if err != nil {
		return nil, err
	}
	metadata := &conditionalRequest{previous: previous}
	ctx = context.WithValue(ctx, conditionalKey{}, metadata)
	data, err := p.Fetch(ctx, source)
	if err != nil && !errors.Is(err, errNotModified) {
		return nil, err
	}
	return FetchResult{Data: data, NotModified: errors.Is(err, errNotModified), Validators: &metadata.received}, nil
}

type refreshFence struct {
	SourceID, Token string
	Now             time.Time
}

func (s *Store) writeSnapshot(key string, snapshot Snapshot, validators *HTTPValidators, fence *refreshFence) (bool, error) {
	data, err := json.Marshal(snapshot)
	if err != nil {
		return false, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	if fence != nil && fence.Token != "" {
		var count int
		err = tx.QueryRow(`SELECT count(*) FROM refresh_jobs j JOIN sources s ON s.id=j.source_id WHERE j.cache_key=? AND j.lease_token=? AND j.lease_until>? AND j.source_generation=s.generation AND s.paused=0 AND EXISTS(SELECT 1 FROM board_sources r WHERE r.source_id=s.id)`, key, fence.Token, fence.Now.UnixNano()).Scan(&count)
		if err != nil {
			return false, err
		}
		if count != 1 {
			return false, nil
		}
	}
	if _, err = tx.Exec("INSERT INTO snapshots(cache_key,document) VALUES(?,?) ON CONFLICT(cache_key) DO UPDATE SET document=excluded.document", key, data); err != nil {
		return false, err
	}
	if fence != nil && fence.Token != "" && snapshot.State == "fresh" {
		var document []byte
		var source Source
		if err = tx.QueryRow("SELECT document FROM sources WHERE id=? AND owner_id='local'", fence.SourceID).Scan(&document); err != nil {
			return false, err
		}
		if err = json.Unmarshal(document, &source); err != nil {
			return false, err
		}
		if _, err = ingestFeedEntries(tx, source, snapshot.Data, fence.Now); err != nil {
			return false, err
		}
	}
	if validators != nil {
		if _, err = tx.Exec("INSERT INTO http_validators(cache_key,etag,last_modified,updated_at) VALUES(?,?,?,?) ON CONFLICT(cache_key) DO UPDATE SET etag=excluded.etag,last_modified=excluded.last_modified,updated_at=excluded.updated_at", key, validators.ETag, validators.LastModified, time.Now().UTC().Format(time.RFC3339)); err != nil {
			return false, err
		}
	}
	if err = tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}
