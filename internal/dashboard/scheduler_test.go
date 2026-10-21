package dashboard

import (
	"context"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestSchedulerRefreshesWithoutBrowserAndPersistsItsDueTime(t *testing.T) {
	path := filepath.Join(t.TempDir(), "jobs.db")
	store, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	fetch := func(context.Context, Source) (any, error) {
		calls.Add(1)
		return FeedData{Title: "scheduled", Items: []FeedItem{}}, nil
	}
	runtime := NewRuntime(store, fetch)
	runtime.startScheduler(10 * time.Millisecond)
	eventually(t, func() bool {
		var count int
		store.db.QueryRow("SELECT count(*) FROM refresh_jobs WHERE lease_until=0 AND next_run>0").Scan(&count)
		return count == 1
	})
	runtime.Close()
	store.Close()
	store, err = OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	restarted := NewRuntime(store, fetch)
	defer restarted.Close()
	board, _ := store.Board()
	snapshot, err := restarted.Read(board.Sources[0])
	if err != nil || snapshot.State != "fresh" || calls.Load() != 1 {
		t.Fatal("restart ignored persisted freshness", calls.Load(), err)
	}
}
func TestRefreshLeaseIsExclusiveAndRecoversAfterCrash(t *testing.T) {
	store := testStore(t)
	board, _ := store.Board()
	source := board.Sources[0]
	key := sourceKey(source)
	now := time.Now()
	first, known, claimed, err := store.claimRefresh(source, key, now)
	if err != nil || !known || !claimed {
		t.Fatal("claim failed", err)
	}
	_, _, claimed, err = store.claimRefresh(source, key, now.Add(time.Second))
	if err != nil || claimed {
		t.Fatal("duplicate lease granted")
	}
	second, _, claimed, err := store.claimRefresh(source, key, now.Add(refreshLeaseDuration+time.Second))
	if err != nil || !claimed || second == first {
		t.Fatal("expired lease not recovered")
	}
	if err = store.finishRefresh(key, first, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	var token string
	store.db.QueryRow("SELECT lease_token FROM refresh_jobs WHERE cache_key=?", key).Scan(&token)
	if token != second {
		t.Fatal("old worker overwrote new lease")
	}
}
