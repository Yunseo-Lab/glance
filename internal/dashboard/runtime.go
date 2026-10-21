package dashboard

import (
	"bytes"
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
	ManualRefreshAfter *time.Time      `json:"manualRefreshAfter,omitempty"`
	ErrorCode          string          `json:"errorCode,omitempty"`
	AttemptedAt        *time.Time      `json:"attemptedAt,omitempty"`
	Data               json.RawMessage `json:"data,omitempty"`
	State              string          `json:"state"`
	UpdatedAt          *time.Time      `json:"updatedAt,omitempty"`
	ExpiresAt          *time.Time      `json:"expiresAt,omitempty"`
	RetryAt            *time.Time      `json:"retryAt,omitempty"`
	Error              string          `json:"error,omitempty"`
}

type FetchFunc func(context.Context, Source) (any, error)
type taskState struct {
	sourceID   string
	cancel     context.CancelFunc
	leaseToken string
	hostSlot   chan struct{}
	running    bool
	failures   int
}

// Runtime owns fetching, concurrency and freshness. Providers never render HTML.
type Runtime struct {
	schedulerStarted  bool
	requireRegistered bool
	hostSlots         map[string]chan struct{}
	jitter            func(time.Duration) time.Duration
	now               func() time.Time
	store             *Store
	fetch             FetchFunc
	ctx               context.Context
	cancel            context.CancelFunc
	mu                sync.Mutex
	tasks             map[string]*taskState
	slots             chan struct{}
	wg                sync.WaitGroup
	closed            bool
}

func NewRuntime(store *Store, fetch FetchFunc) *Runtime {
	ctx, cancel := context.WithCancel(context.Background())
	return &Runtime{jitter: randomJitter, hostSlots: map[string]chan struct{}{}, now: time.Now, store: store, fetch: fetch, ctx: ctx, cancel: cancel, tasks: map[string]*taskState{}, slots: make(chan struct{}, 4)}
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
	paused, known, pauseErr := r.store.sourcePaused(source.ID)
	if pauseErr != nil {
		return snap, pauseErr
	}
	if known && paused {
		snap = snapshotState(snap, r.now())
		snap.State = "paused"
		return snap, nil
	}
	now := r.now()
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
	now = r.now()
	latest = snapshotState(latest, now)
	if latest.State == "fresh" || latest.RetryAt != nil && latest.RetryAt.After(now) {
		return latest, nil
	}
	select {
	case r.slots <- struct{}{}:
	default:
		return snap, nil
	}
	host := source.Type
	if definition, ok := providerDefinitions[source.Type]; ok && definition.host != nil {
		host = definition.host(source)
	}
	hostSlot := r.hostSlots[host]
	if hostSlot == nil {
		if len(r.hostSlots) >= MaxActiveSources {
			for key, slot := range r.hostSlots {
				if len(slot) == 0 {
					delete(r.hostSlots, key)
				}
			}
		}
		hostSlot = make(chan struct{}, 2)
		r.hostSlots[host] = hostSlot
	}
	select {
	case hostSlot <- struct{}{}:
	default:
		<-r.slots
		return snap, nil
	}
	token, registered, claimed, claimErr := r.store.claimRefresh(source, key, now)
	if claimErr != nil {
		<-hostSlot
		<-r.slots
		return snap, claimErr
	}
	if (registered && !claimed) || (!registered && r.requireRegistered) {
		<-hostSlot
		<-r.slots
		return snap, nil
	}
	state.leaseToken = token
	state.hostSlot = hostSlot
	state.sourceID = source.ID
	ctx, cancel := context.WithTimeout(r.ctx, 10*time.Second)
	state.cancel = cancel
	state.running = true
	r.wg.Add(1)
	go r.refresh(ctx, key, source, state)
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

func (r *Runtime) refresh(ctx context.Context, key string, source Source, state *taskState) {
	defer r.wg.Done()
	defer func() { <-state.hostSlot; <-r.slots; r.mu.Lock(); state.running = false; r.mu.Unlock() }()
	leaseToken := state.leaseToken
	nextRun := r.now().Add(30 * time.Second)
	defer func() {
		if leaseToken != "" {
			if err := r.store.finishRefresh(key, leaseToken, nextRun); err != nil {
				log.Print("dashboard refresh lease release failed")
			}
		}
	}()
	defer state.cancel()
	data, err := r.callProvider(ctx, source)
	now := r.now()
	if ctx.Err() != nil && r.ctx.Err() != nil {
		return
	}
	snap := Snapshot{AttemptedAt: &now}
	var validators *HTTPValidators
	if result, ok := data.(FetchResult); ok && err == nil {
		validators = result.Validators
		if result.NotModified {
			previous, readErr := r.store.snapshot(key)
			if readErr != nil {
				err = readErr
			} else if len(previous.Data) == 0 {
				err = &ProviderError{Code: "invalid_response"}
			} else {
				data = json.RawMessage(previous.Data)
			}
		} else {
			data = result.Data
		}
	}

	if err == nil {
		snap.Data, err = json.Marshal(data)
		if err == nil && (len(snap.Data) == 0 || bytes.Equal(snap.Data, []byte("null"))) {
			err = &ProviderError{Code: "invalid_response"}
		}
	}
	if err == nil {
		ttl := time.Duration(providerDefinitions[source.Type].FreshForSeconds) * time.Second
		if ttl <= 0 {
			ttl = 15 * time.Minute
		}
		expires := now.Add(ttl)
		snap.UpdatedAt = &now
		snap.ExpiresAt = &expires
		nextRun = expires
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
		retry := now.Add(retryDelay(failures, err, r.jitter))
		snap.RetryAt = &retry
		nextRun = retry
		snap.AttemptedAt = &now
		snap.ErrorCode = classifyProviderError(err)
		snap.Error = "정보를 가져오지 못했습니다. 잠시 후 자동으로 다시 시도합니다."
		snap.State = "error"
		// Errors may embed URLs, query credentials or response bodies.
		log.Printf("dashboard provider=%s outcome=fetch_failed", source.Type)
	}
	if _, err := r.store.writeSnapshot(key, snap, validators, &refreshFence{SourceID: source.ID, Token: leaseToken, Now: r.now()}); err != nil {
		log.Printf("dashboard cache write: %v", err)
	}
}
