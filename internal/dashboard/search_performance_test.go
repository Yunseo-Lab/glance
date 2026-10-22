package dashboard

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestKoreanShortAndLiteralSearch(t *testing.T) {
	store := testStore(t)
	board, _ := store.Board()
	now := time.Now()
	ingestForTest(t, store, board.Sources[0], FeedData{Title: "한글 소스", Items: []FeedItem{{ID: "seoul", Title: "서울시 프론티어 Go École", URL: "https://example.com/seoul", Excerpt: "한글 검색 본문"}}}, now)
	for _, query := range []string{"서울시", "서울", "서", "go", "GO", "é", "É", "프론티어 서울", "서울"} {
		page, err := store.Entries(EntryFilter{Query: query})
		if err != nil || page.Total != 1 {
			t.Fatal(query, page, err)
		}
	}
	for _, query := range []string{`서울 OR missing`, `서울*`, `" OR "`, strings.Repeat("가", 201), "nul\x00value"} {
		page, err := store.Entries(EntryFilter{Query: query})
		if err == nil && page.Total != 0 {
			t.Fatal("search syntax escaped literal semantics", query)
		}
	}
}
func seedSearchBudget(t *testing.T, store *Store) {
	t.Helper()
	tx, err := store.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	statement, err := tx.Prepare("INSERT INTO entries(id,owner_id,item_key,title,url,excerpt,collected_at,source_title,sort_at) VALUES(?,'local',?,?,?,?,?,'기술 뉴스',?)")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	excerpt := strings.Repeat("서울의 프론티어 기술과 Go 개발 소식. ", 30)
	for index := range MaxEntries {
		id := fmt.Sprintf("search-%05d", index)
		if _, err = statement.Exec(id, id, fmt.Sprintf("서울시 프론티어 Go École 소식 %d", index), fmt.Sprintf("https://example.com/%d", index), excerpt, now.Format(time.RFC3339Nano), now.UnixNano()-int64(index)); err != nil {
			t.Fatal(err)
		}
	}
	statement.Close()
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
}
func TestSearch10000Performance(t *testing.T) {
	if os.Getenv("GLANCE_SEARCH_BENCH") != "1" {
		t.Skip("explicit 10000-entry timing run")
	}
	store := testStore(t)
	seedSearchBudget(t, store)
	type measurement struct {
		Query   string  `json:"query"`
		Samples int     `json:"samples"`
		P50     float64 `json:"p50_ms"`
		P95     float64 `json:"p95_ms"`
		Matches int     `json:"matches"`
	}
	report := struct {
		Entries      int           `json:"entries"`
		Runtime      string        `json:"runtime"`
		SQLite       string        `json:"sqlite"`
		Measurements []measurement `json:"measurements"`
	}{Entries: MaxEntries, Runtime: runtime.Version(), Measurements: []measurement{}}
	store.db.QueryRow("SELECT sqlite_version()").Scan(&report.SQLite)
	for _, query := range []string{"프론티어", "서울", "서", "Go", "é", "없는검색어"} {
		samples := []float64{}
		matches := 0
		for iteration := 0; iteration < 31; iteration++ {
			start := time.Now()
			page, err := store.Entries(EntryFilter{Query: query, Limit: 30})
			elapsed := float64(time.Since(start).Microseconds()) / 1000
			if err != nil {
				t.Fatal(query, err)
			}
			matches = page.Total
			if iteration > 0 {
				samples = append(samples, elapsed)
			}
		}
		sort.Float64s(samples)
		result := measurement{Query: query, Samples: len(samples), P50: samples[len(samples)/2], P95: samples[int(float64(len(samples)-1)*.95)], Matches: matches}
		report.Measurements = append(report.Measurements, result)
		if result.P95 > 150 {
			t.Errorf("bounded search p95 over 150ms: %+v", result)
		}
	}
	encoded, _ := json.Marshal(report)
	t.Logf("SEARCH_BUDGET %s", encoded)
	var id, parent, unused int
	var detail string
	rows, err := store.db.Query("EXPLAIN QUERY PLAN SELECT rowid FROM entry_fts WHERE entry_fts MATCH ?", `"프론티어"`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	indexed := false
	for rows.Next() {
		rows.Scan(&id, &parent, &unused, &detail)
		t.Log(detail)
		if strings.Contains(detail, "VIRTUAL TABLE INDEX") {
			indexed = true
		}
	}
	if !indexed {
		t.Fatal("FTS index not used")
	}
}
