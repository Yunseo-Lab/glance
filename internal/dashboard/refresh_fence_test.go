package dashboard

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestDeletingSourceCancelsWorkAndCannotResurrectCache(t *testing.T) {
	store := testStore(t)
	home, _ := store.Board()
	source := home.Sources[0]
	started := make(chan struct{})
	cancelled := make(chan struct{})
	runtime := NewRuntime(store, func(ctx context.Context, source Source) (any, error) {
		close(started)
		<-ctx.Done()
		close(cancelled)
		return FeedData{Title: "old result"}, nil
	})
	defer runtime.Close()
	runtime.Read(source)
	<-started
	service := DashboardService{store: store, runtime: runtime}
	home.Sources = []Source{}
	home.Widgets = []Widget{}
	if _, err := service.Commit(home); err != nil {
		t.Fatal(err)
	}
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("removed source did not cancel")
	}
	runtime.Close()
	snapshot, err := store.snapshot(sourceKey(source))
	if err != nil || len(snapshot.Data) != 0 {
		t.Fatal("deleted source recreated cache")
	}
}
func TestExpiredOrInvalidatedLeaseCannotCommitAResult(t *testing.T) {
	store := testStore(t)
	home, _ := store.Board()
	source := home.Sources[0]
	key := sourceKey(source)
	now := time.Now()
	token, _, _, err := store.claimRefresh(source, key, now)
	if err != nil {
		t.Fatal(err)
	}
	store.db.Exec("UPDATE sources SET generation=generation+1 WHERE id=?", source.ID)
	written, err := store.writeSnapshot(key, Snapshot{State: "fresh", Data: raw(FeedData{Title: "obsolete"})}, nil, &refreshFence{SourceID: source.ID, Token: token, Now: now})
	if err != nil || written {
		t.Fatal("obsolete generation committed", err)
	}
	store.db.Exec("UPDATE sources SET generation=generation-1 WHERE id=?", source.ID)
	written, err = store.writeSnapshot(key, Snapshot{State: "fresh", Data: raw(FeedData{})}, nil, &refreshFence{SourceID: source.ID, Token: token, Now: now.Add(refreshLeaseDuration + time.Second)})
	if err != nil || written {
		t.Fatal("expired lease committed")
	}
}
func TestProductionRuntimeRejectsUnregisteredSource(t *testing.T) {
	store := testStore(t)
	var calls atomic.Int32
	runtime := NewRuntime(store, func(context.Context, Source) (any, error) { calls.Add(1); return nil, nil })
	defer runtime.Close()
	runtime.mu.Lock()
	runtime.requireRegistered = true
	runtime.mu.Unlock()
	runtime.Read(Source{ID: "deleted", Type: "rss", Config: raw(RSSSource{URL: "https://example.com"})})
	runtime.Close()
	if calls.Load() != 0 {
		t.Fatal("unregistered source fetched")
	}
}
