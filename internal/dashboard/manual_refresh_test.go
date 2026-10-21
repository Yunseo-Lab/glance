package dashboard

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestManualRefreshBudgetIsAtomicAndRespectsProviderCooldown(t *testing.T) {
	store := testStore(t)
	home, _ := store.Board()
	source := home.Sources[0]
	now := time.Now()
	var accepted atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, err := store.RequestRefresh(source.ID, now)
			if err == nil {
				accepted.Add(1)
			} else {
				var cooldown *CooldownError
				if !errors.As(err, &cooldown) {
					t.Error(err)
				}
			}
		}()
	}
	wg.Wait()
	if accepted.Load() != 1 {
		t.Fatal("parallel tabs bypassed cooldown", accepted.Load())
	}
	key := sourceKey(source)
	snapshot, _ := store.snapshot(key)
	retry := now.Add(2 * time.Hour)
	snapshot.RetryAt = &retry
	store.putSnapshot(key, snapshot)
	_, _, err := store.RequestRefresh(source.ID, now.Add(time.Minute))
	var cooldown *CooldownError
	if !errors.As(err, &cooldown) || !cooldown.Until.Equal(retry) {
		t.Fatal("manual refresh bypassed upstream cooldown", err)
	}
}

func TestManualRefreshHasAnOwnerWideBudget(t *testing.T) {
	store := testStore(t)
	home, _ := store.Board()
	home.Sources = []Source{}
	home.Widgets = []Widget{}
	for i := 0; i < 11; i++ {
		home.Sources = append(home.Sources, Source{ID: newID(), Type: "rss", Config: raw(RSSSource{URL: "https://example.com/" + newID()})})
	}
	if _, err := store.Save(home); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	for i, source := range home.Sources {
		_, _, err := store.RequestRefresh(source.ID, now)
		if i < 10 && err != nil {
			t.Fatal(err)
		}
		if i == 10 {
			var cooldown *CooldownError
			if !errors.As(err, &cooldown) {
				t.Fatal("owner-wide budget bypassed")
			}
		}
	}
}
