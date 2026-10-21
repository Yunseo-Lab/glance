package dashboard

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestActiveSourceLimitIsTransactional(t *testing.T) {
	store := testStore(t)
	home, _ := store.Board()
	home.Widgets = []Widget{}
	home.Sources = []Source{}
	if _, err := store.Save(home); err != nil {
		t.Fatal(err)
	}
	for batch := 0; batch < 4; batch++ {
		board := Board{Theme: "light", Widgets: []Widget{}, Sources: []Source{}}
		for i := 0; i < 32; i++ {
			source := Source{ID: newID(), Type: "rss", Config: raw(RSSSource{URL: fmt.Sprintf("https://example.com/%d/%d", batch, i)})}
			board.Sources = append(board.Sources, source)
			board.Widgets = append(board.Widgets, Widget{ID: newID(), Type: "rss", Title: "Feed", Size: "small", SourceID: source.ID, Config: raw(RSSConfig{Limit: 5})})
		}
		if _, err := store.CreateBoard(fmt.Sprint(batch), &board); err != nil {
			t.Fatal(err)
		}
	}
	before, _ := store.Boards()
	template := defaultBoard()
	if _, err := store.CreateBoard("over limit", &template); !errors.Is(err, ErrSourceLimit) {
		t.Fatal("source limit not enforced", err)
	}
	after, _ := store.Boards()
	sources, _ := store.Sources()
	if len(after) != len(before) || len(sources) != MaxActiveSources {
		t.Fatal("failed creation left partial data")
	}
}
func TestOwnerSourceQueryUsesIndexes(t *testing.T) {
	store := testStore(t)
	rows, err := store.db.Query("EXPLAIN QUERY PLAN SELECT document FROM sources WHERE owner_id='local' ORDER BY id")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var details string
	for rows.Next() {
		var id, parent, unused int
		var detail string
		if err = rows.Scan(&id, &parent, &unused, &detail); err != nil {
			t.Fatal(err)
		}
		details += detail
	}
	if !strings.Contains(details, "sources_by_owner") {
		t.Fatal("owner source query no longer uses its index", details)
	}
}
