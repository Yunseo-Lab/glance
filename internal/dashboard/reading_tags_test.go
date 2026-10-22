package dashboard

import (
	"errors"
	"strings"
	"testing"
)

func TestEntryTagsNormalizationFiltersAndAtomicBulk(t *testing.T) {
	store := testStore(t)
	source := seedReading(t, store, 3)
	page, _ := store.Entries(EntryFilter{})
	first, second := page.Items[0].ID, page.Items[1].ID
	tags := []string{" 중요 ", "Go", "go", "e\u0301"}
	yes := true
	result, err := store.BulkEntryState(EntryBulkInput{IDs: []string{first, second, first}, Read: &yes, Tags: &tags})
	if err != nil || result.Updated != 2 {
		t.Fatal(result, err)
	}
	tagged, err := store.Entries(EntryFilter{Tag: "중요", SourceID: source.ID, BoardID: "home", Status: "read"})
	if err != nil || tagged.Total != 2 {
		t.Fatal(tagged, err)
	}
	for _, entry := range tagged.Items {
		if len(entry.Tags) != 3 || entry.Tags[0] != "Go" {
			t.Fatal("tags not normalized", entry.Tags)
		}
	}
	counts, err := store.EntryTags()
	if err != nil || len(counts) != 3 || counts[0].Count != 2 {
		t.Fatal(counts, err)
	}
	// A missing final ID rolls back all earlier state updates in the batch.
	if _, err = store.BulkEntryState(EntryBulkInput{IDs: []string{first, "missing-entry"}, Saved: &yes}); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
	saved, _ := store.Entries(EntryFilter{Status: "saved"})
	if saved.Total != 0 {
		t.Fatal("partial bulk write")
	}
	clear := []string{}
	state, err := store.SetEntryState(first, EntryStateInput{Tags: &clear})
	if err != nil || len(state.Tags) != 0 || !state.IsRead {
		t.Fatal("tag clear lost read state", state, err)
	}
	tagged, _ = store.Entries(EntryFilter{Tag: "중요"})
	if tagged.Total != 1 {
		t.Fatal(tagged.Total)
	}
	for _, tags := range [][]string{{""}, {"line\nbreak"}, {"comma,tag"}, {strings.Repeat("가", 14)}, make([]string, 11)} {
		if _, err = store.SetEntryState(first, EntryStateInput{Tags: &tags}); !errors.Is(err, ErrInvalidInput) {
			t.Fatal("invalid tags accepted", tags, err)
		}
	}
	if _, err = store.BulkEntryState(EntryBulkInput{IDs: make([]string, 101), Saved: &yes}); !errors.Is(err, ErrInvalidInput) {
		t.Fatal("bulk budget bypassed")
	}
}
