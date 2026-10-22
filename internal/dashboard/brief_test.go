package dashboard

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestCivilDateBoundariesHandleDSTAndSkippedMidnight(t *testing.T) {
	for _, fixture := range []struct {
		date, zone string
		hours      int
	}{{"2026-03-08", "America/New_York", 23}, {"2026-11-01", "America/New_York", 25}, {"2018-11-04", "America/Sao_Paulo", 23}, {"2011-12-29", "Pacific/Apia", 24}} {
		date, _ := time.Parse("2006-01-02", fixture.date)
		location, _ := time.LoadLocation(fixture.zone)
		start, end := civilDateBoundary(date, location), civilDateBoundary(date.AddDate(0, 0, 1), location)
		if start.Format("2006-01-02") != fixture.date || end.Sub(start) != time.Duration(fixture.hours)*time.Hour {
			t.Fatal(fixture, start, end)
		}
	}
}
func TestDailyBriefDeterministicCitationsCountsAndOwnerScope(t *testing.T) {
	store := testStore(t)
	instants := []string{"2026-11-01T03:59:59Z", "2026-11-01T04:00:00Z", "2026-11-02T04:59:59Z", "2026-11-02T05:00:00Z"}
	for index, instant := range instants {
		at, _ := time.Parse(time.RFC3339, instant)
		_, err := store.db.Exec("INSERT INTO entries(id,owner_id,item_key,title,url,excerpt,collected_at,source_title,sort_at) VALUES(?,'local',?,?,?,'인용할 발췌',?,'원본 출처',?)", fmt.Sprintf("brief-%d", index), fmt.Sprintf("brief-%d", index), fmt.Sprintf("기사 %d", index), fmt.Sprintf("https://example.com/%d", index), instant, at.UnixNano())
		if err != nil {
			t.Fatal(err)
		}
	}
	store.db.Exec("INSERT INTO entries(id,owner_id,item_key,title,url,collected_at,sort_at) VALUES('brief-foreign','other','foreign','PRIVATE','https://example.com/private','2026-11-01',?)", time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC).UnixNano())
	yes := true
	store.SetEntryState("brief-1", EntryStateInput{Read: &yes, Saved: &yes})
	first, err := store.DailyBrief("2026-11-01", "America/New_York")
	if err != nil || first.Total != 2 || first.Unread != 1 || first.Saved != 1 || len(first.Items) != 2 || first.Items[0].ID != "brief-2" {
		t.Fatal(first, err)
	}
	second, err := store.DailyBrief("2026-11-01", "America/New_York")
	if err != nil || string(raw(first)) != string(raw(second)) || len(first.Fingerprint) != 64 {
		t.Fatal("brief is not deterministic")
	}
	for _, item := range first.Items {
		if item.SourceTitle != "원본 출처" || !strings.HasPrefix(item.URL, "https://example.com/") {
			t.Fatal("citation missing")
		}
	}
	store.SetEntryState("brief-2", EntryStateInput{Read: &yes})
	changed, _ := store.DailyBrief("2026-11-01", "America/New_York")
	if changed.Fingerprint == first.Fingerprint || changed.Unread != 0 {
		t.Fatal("new state not reflected")
	}
	for _, fixture := range [][2]string{{"2026-02-30", "UTC"}, {"2026-11-01", "Not/AZone"}, {"2011-12-30", "Pacific/Apia"}} {
		if _, err = store.DailyBrief(fixture[0], fixture[1]); err == nil {
			t.Fatal("invalid date accepted", fixture)
		}
	}
}
