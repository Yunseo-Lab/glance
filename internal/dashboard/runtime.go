package dashboard

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"
)

type Snapshot struct {
	Data      json.RawMessage `json:"data,omitempty"`
	State     string          `json:"state"`
	UpdatedAt *time.Time      `json:"updatedAt,omitempty"`
	ExpiresAt *time.Time      `json:"expiresAt,omitempty"`
	RetryAt   *time.Time      `json:"retryAt,omitempty"`
	Error     string          `json:"error,omitempty"`
}

type FetchFunc func(context.Context, Source) (any, error)
type taskState struct {
	running  bool
	failures int
}

// Runtime owns fetching, concurrency and freshness. Providers never render HTML.
type Runtime struct {
	store  *Store
	fetch  FetchFunc
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.Mutex
	tasks  map[string]*taskState
	slots  chan struct{}
	wg     sync.WaitGroup
	closed bool
}

func NewRuntime(store *Store, fetch FetchFunc) *Runtime {
	ctx, cancel := context.WithCancel(context.Background())
	return &Runtime{store: store, fetch: fetch, ctx: ctx, cancel: cancel, tasks: map[string]*taskState{}, slots: make(chan struct{}, 4)}
}
func sourceKey(source Source) string {
	// Scope is explicit even for the initial local owner. Future authenticated
	// connections must include their identity in this key before cache sharing.
	var config any
	_ = json.Unmarshal(source.Config, &config)
	canonical, _ := json.Marshal(config)
	h := sha256.Sum256(append([]byte("local:"+source.Type+":"), canonical...))
	return hex.EncodeToString(h[:])
}
func (r *Runtime) Close() { r.mu.Lock(); r.closed = true; r.cancel(); r.mu.Unlock(); r.wg.Wait() }

func snapshotState(snap Snapshot, now time.Time) Snapshot {
	if snap.UpdatedAt != nil && now.Sub(*snap.UpdatedAt) > 24*time.Hour {
		snap.Data = nil
	}
	if snap.ExpiresAt != nil && snap.ExpiresAt.After(now) && snap.Data != nil {
		snap.State = "fresh"
		return snap
	}
	snap.State = "loading"
	if snap.Data != nil {
		snap.State = "stale"
	}
	if snap.Error != "" && snap.Data == nil {
		snap.State = "error"
	}
	return snap
}

func (r *Runtime) Read(source Source) (Snapshot, error) {
	key := sourceKey(source)
	snap, err := r.store.snapshot(key)
	if err != nil {
		return snap, err
	}
	now := time.Now()
	snap = snapshotState(snap, now)
	if snap.State == "fresh" || snap.RetryAt != nil && snap.RetryAt.After(now) {
		return snap, nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return snap, nil
	}
	state := r.tasks[key]
	if state == nil {
		// Keep bookkeeping bounded when configurations are repeatedly edited.
		if len(r.tasks) >= 128 {
			for k, s := range r.tasks {
				if !s.running {
					delete(r.tasks, k)
				}
			}
		}
		state = &taskState{}
		r.tasks[key] = state
	}
	if state.running {
		return snap, nil
	}
	latest, readErr := r.store.snapshot(key)
	if readErr != nil {
		return snap, readErr
	}
	// A refresh may have completed between the first read and acquiring mu.
	// Recheck both successful freshness and failed-fetch backoff before scheduling.
	now = time.Now()
	latest = snapshotState(latest, now)
	if latest.State == "fresh" || latest.RetryAt != nil && latest.RetryAt.After(now) {
		return latest, nil
	}
	select {
	case r.slots <- struct{}{}:
	default:
		return snap, nil
	}
	state.running = true
	r.wg.Add(1)
	go r.refresh(key, source, state)
	return snap, nil
}

func (r *Runtime) callProvider(ctx context.Context, source Source) (data any, err error) {
	defer func() {
		if recover() != nil {
			err = errors.New("provider panic")
		}
	}()
	return r.fetch(ctx, source)
}

func (r *Runtime) refresh(key string, source Source, state *taskState) {
	defer r.wg.Done()
	defer func() { <-r.slots; r.mu.Lock(); state.running = false; r.mu.Unlock() }()
	ctx, cancel := context.WithTimeout(r.ctx, 10*time.Second)
	defer cancel()
	data, err := r.callProvider(ctx, source)
	now := time.Now()
	if ctx.Err() != nil && r.ctx.Err() != nil {
		return
	}
	snap := Snapshot{}
	if err == nil {
		snap.Data, err = json.Marshal(data)
	}
	if err == nil {
		ttl := 15 * time.Minute
		if source.Type == "rss" {
			ttl = 10 * time.Minute
		}
		expires := now.Add(ttl)
		snap.UpdatedAt = &now
		snap.ExpiresAt = &expires
		snap.State = "fresh"
		r.mu.Lock()
		state.failures = 0
		r.mu.Unlock()
	} else {
		var readErr error
		snap, readErr = r.store.snapshot(key)
		if readErr != nil {
			log.Printf("dashboard cache read: %v", readErr)
			return
		}
		r.mu.Lock()
		state.failures++
		failures := state.failures
		r.mu.Unlock()
		if failures > 5 {
			failures = 5
		}
		retry := now.Add(time.Duration(1<<failures) * 15 * time.Second)
		snap.RetryAt = &retry
		snap.Error = "정보를 가져오지 못했습니다. 잠시 후 자동으로 다시 시도합니다."
		snap.State = "error"
		// Errors may embed URLs, query credentials or response bodies.
		log.Printf("dashboard provider=%s outcome=fetch_failed", source.Type)
	}
	if err := r.store.putSnapshot(key, snap); err != nil {
		log.Printf("dashboard cache write: %v", err)
	}
}
