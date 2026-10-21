package dashboard

import (
	"errors"
	"testing"
	"time"
)

func TestRetentionKeepsSavedItemsAndDeletingBoardDoesNotEraseReadingState(t *testing.T) {
	store := testStore(t)
	home, _ := store.Board()
	sourceID := home.Sources[0].ID
	now := time.Now().UTC()
	old := now.AddDate(0, 0, -EntryRetentionDays-1).Format(time.RFC3339Nano)
	for _, id := range []string{"old-unsaved", "old-saved"} {
		if _, err := store.db.Exec("INSERT INTO entries(id,owner_id,source_id,item_key,title,url,collected_at) VALUES(?,'local',?,?,'Title','https://example.com',?)", id, sourceID, id, old); err != nil {
			t.Fatal(err)
		}
	}
	store.db.Exec("INSERT INTO entry_state(owner_id,entry_id,is_saved,updated_at) VALUES('local','old-saved',1,?)", old)
	if err := store.DeleteSource(sourceID); !errors.Is(err, ErrInUse) {
		t.Fatal("referenced source deleted")
	}
	if err := store.RetainData(now); err != nil {
		t.Fatal(err)
	}
	var count int
	store.db.QueryRow("SELECT count(*) FROM entries WHERE id='old-unsaved'").Scan(&count)
	if count != 0 {
		t.Fatal("unsaved expired item retained")
	}
	store.db.QueryRow("SELECT count(*) FROM entries WHERE id='old-saved'").Scan(&count)
	if count != 1 {
		t.Fatal("saved item was expired")
	}
	home.Sources = []Source{}
	home.Widgets = []Widget{}
	if _, err := store.Save(home); err != nil {
		t.Fatal(err)
	}
	var absent int
	store.db.QueryRow("SELECT count(*) FROM sources WHERE id=?", sourceID).Scan(&absent)
	if absent != 0 {
		t.Fatal("orphan source retained")
	}
	var source any
	if err := store.db.QueryRow("SELECT source_id FROM entries WHERE id='old-saved'").Scan(&source); err != nil || source != nil {
		t.Fatal("saved entry did not survive orphan removal")
	}
}
