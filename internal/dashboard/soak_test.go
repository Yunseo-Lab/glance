package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type soakLatency struct {
	Count int     `json:"count"`
	P50   float64 `json:"p50_ms"`
	P95   float64 `json:"p95_ms"`
	Max   float64 `json:"max_ms"`
}
type soakMemory struct {
	Second     int    `json:"second"`
	RSS        uint64 `json:"rss_bytes"`
	Heap       uint64 `json:"heap_bytes"`
	Goroutines int    `json:"goroutines"`
}

func processRSS() uint64 {
	data, _ := os.ReadFile("/proc/self/status")
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "VmRSS:") {
			fields := strings.Fields(line)
			value, _ := strconv.ParseUint(fields[1], 10, 64)
			return value * 1024
		}
	}
	return 0
}
func latencySummary(values []float64) soakLatency {
	if len(values) == 0 {
		return soakLatency{}
	}
	sort.Float64s(values)
	return soakLatency{Count: len(values), P50: values[len(values)/2], P95: values[int(float64(len(values)-1)*.95)], Max: values[len(values)-1]}
}
func TestDashboardSoak(t *testing.T) {
	rawDuration := os.Getenv("GLANCE_SOAK_DURATION")
	if rawDuration == "" {
		t.Skip("explicit mixed workload; GLANCE_SOAK_DURATION=60m")
	}
	duration, err := time.ParseDuration(rawDuration)
	if err != nil || duration < time.Minute || duration > 2*time.Hour {
		t.Fatal("soak duration must be 1m..2h")
	}
	store := testStore(t)
	var allSources []Source
	for boardIndex := 0; boardIndex < 4; boardIndex++ {
		var board Board
		if boardIndex == 0 {
			board, _ = store.Board()
		} else {
			board, err = store.CreateBoard(fmt.Sprintf("Soak %d", boardIndex), nil)
			if err != nil {
				t.Fatal(err)
			}
		}
		board.Widgets = []Widget{}
		board.Sources = []Source{}
		for item := 0; item < 32; item++ {
			index := boardIndex*32 + item
			id := fmt.Sprintf("soak-%03d", index)
			source := Source{ID: id, Type: "rss", Config: raw(RSSSource{URL: fmt.Sprintf("https://feed-%03d.example.com/feed", index)})}
			board.Sources = append(board.Sources, source)
			board.Widgets = append(board.Widgets, Widget{ID: "widget-" + id, Type: "rss", Title: id, Size: "small", SourceID: id, Config: raw(RSSConfig{Limit: 5})})
			allSources = append(allSources, source)
		}
		if _, err = store.Save(board); err != nil {
			t.Fatal(err)
		}
	}
	seedSearchBudget(t, store)
	if _, err = store.db.Exec("UPDATE entries SET source_id='soak-'||printf('%03d',CAST(substr(id,8) AS INTEGER)%128)"); err != nil {
		t.Fatal(err)
	}
	if _, err = store.db.Exec("INSERT INTO entry_sources(entry_id,source_id,item_key) SELECT id,source_id,item_key FROM entries"); err != nil {
		t.Fatal(err)
	}
	yes := true
	store.SetEntryState("search-09999", EntryStateInput{Saved: &yes})
	var fetches atomic.Int64
	var fetchMu sync.Mutex
	running := map[string]int{}
	hasBody := map[string]bool{}
	duplicates := 0
	fetch := func(ctx context.Context, source Source) (any, error) {
		fetchMu.Lock()
		running[source.ID]++
		if running[source.ID] > 1 {
			duplicates++
		}
		previous := hasBody[source.ID]
		fetchMu.Unlock()
		defer func() { fetchMu.Lock(); running[source.ID]--; fetchMu.Unlock() }()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(20 * time.Millisecond):
		}
		call := fetches.Add(1)
		if call%9 == 0 {
			return nil, &ProviderError{Code: "rate_limited", RetryAfter: 3 * time.Second}
		}
		if call%5 == 0 && previous {
			return FetchResult{NotModified: true, Validators: &HTTPValidators{ETag: "fixture"}}, nil
		}
		index, _ := strconv.Atoi(strings.TrimPrefix(source.ID, "soak-"))
		now := time.Now().UTC()
		feed := FeedData{Title: "소식 " + source.ID, Items: []FeedItem{}}
		for n := 0; n < 5; n++ {
			entryIndex := index + n*128
			feed.Items = append(feed.Items, FeedItem{ID: fmt.Sprintf("search-%05d", entryIndex), Title: fmt.Sprintf("서울시 프론티어 Go 소식 %d", entryIndex), URL: fmt.Sprintf("https://example.com/%d", entryIndex), Excerpt: "반복 수집에서 읽음과 저장 상태를 보존합니다.", PublishedAt: &now})
		}
		fetchMu.Lock()
		hasBody[source.ID] = true
		fetchMu.Unlock()
		return FetchResult{Data: feed, Validators: &HTTPValidators{ETag: "fixture"}}, nil
	}
	engine := NewRuntime(store, fetch)
	engine.StartScheduler()
	server := httptest.NewServer(NewHandler(store, engine, "soak-owner-fixture-password-32chars"))
	client := server.Client()
	client.Timeout = 5 * time.Second
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()
	var group sync.WaitGroup
	var metricsMu sync.Mutex
	latencies := map[string][]float64{}
	requests, failures, limited := 0, 0, 0
	failureMessages := []string{}
	samples := []soakMemory{}
	for worker := 0; worker < 8; worker++ {
		group.Add(1)
		go func(worker int) {
			defer group.Done()
			ticker := time.NewTicker(200 * time.Millisecond)
			defer ticker.Stop()
			for turn := 0; ; turn++ {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
				}
				method, path, body, category := "GET", "/api/v1/dashboard", "", "board"
				switch (worker + turn) % 10 {
				case 0, 1, 2, 3:
					category = "search"
					query := []string{"프론티어", "서울", "서", "Go"}[turn%4]
					path = "/api/v1/entries?q=" + url.QueryEscape(query) + "&limit=30"
				case 4:
					category = "state"
					method = "PATCH"
					path = fmt.Sprintf("/api/v1/entries/search-%05d/state", (turn+worker)%256)
					body = fmt.Sprintf(`{"read":%t,"saved":%t}`, turn%2 == 0, turn%7 == 0)
				case 5:
					category = "refresh"
					method = "POST"
					path = fmt.Sprintf("/api/v1/sources/soak-%03d/refresh", (turn+worker)%128)
					body = "{}"
				case 6:
					category = "snapshots"
					path = "/api/v1/snapshots"
				case 7:
					category = "diagnostics"
					path = "/api/v1/diagnostics"
				case 8:
					category = "brief"
					path = "/api/v1/brief?timezone=Asia%2FSeoul"
				}
				request, _ := http.NewRequestWithContext(ctx, method, server.URL+path, strings.NewReader(body))
				request.SetBasicAuth("owner", "soak-owner-fixture-password-32chars")
				if method != "GET" {
					request.Header.Set("Origin", server.URL)
					request.Header.Set("Content-Type", "application/json")
				}
				began := time.Now()
				response, requestErr := client.Do(request)
				elapsed := float64(time.Since(began).Microseconds()) / 1000
				status := 0
				if requestErr == nil {
					status = response.StatusCode
					_, requestErr = io.Copy(io.Discard, response.Body)
					response.Body.Close()
				}
				if ctx.Err() != nil {
					return
				}
				metricsMu.Lock()
				requests++
				if len(latencies[category]) < 500000 {
					latencies[category] = append(latencies[category], elapsed)
				}
				if status == 429 && category == "refresh" {
					limited++
				} else if requestErr != nil || status != 200 && status != 202 {
					failures++
					if len(failureMessages) < 10 {
						failureMessages = append(failureMessages, fmt.Sprintf("%s status=%d error=%v", category, status, requestErr))
					}
				}
				metricsMu.Unlock()
			}
		}(worker)
	}
	sampleTicker := time.NewTicker(time.Minute)
	defer sampleTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			goto complete
		case <-sampleTicker.C:
			runtime.GC()
			var memory runtime.MemStats
			runtime.ReadMemStats(&memory)
			sample := soakMemory{Second: int(time.Since(start).Seconds()), RSS: processRSS(), Heap: memory.HeapAlloc, Goroutines: runtime.NumGoroutine()}
			samples = append(samples, sample)
			metricsMu.Lock()
			t.Logf("SOAK_PROGRESS elapsed=%ds requests=%d failures=%d rss=%d heap=%d", sample.Second, requests, failures, sample.RSS, sample.Heap)
			metricsMu.Unlock()
		}
	}
complete:
	group.Wait()
	server.Close()
	engine.Close()
	elapsed := time.Since(start)
	summary := map[string]soakLatency{}
	for category, values := range latencies {
		summary[category] = latencySummary(values)
	}
	var entries, active int
	store.db.QueryRow("SELECT count(*) FROM entries WHERE owner_id='local'").Scan(&entries)
	store.db.QueryRow("SELECT count(*) FROM sources WHERE owner_id='local'").Scan(&active)
	var sentinel bool
	store.db.QueryRow("SELECT is_saved FROM entry_state WHERE entry_id='search-09999'").Scan(&sentinel)
	if failures > 0 || duplicates > 0 || entries != MaxEntries || active != 128 || !sentinel {
		t.Errorf("workload failed: requests=%d failures=%d duplicate=%d entries=%d sources=%d saved=%t messages=%v", requests, failures, duplicates, entries, active, sentinel, failureMessages)
	}
	if summary["search"].P95 > 150 || summary["board"].P95 > 100 {
		t.Errorf("latency budget exceeded: search=%+v board=%+v", summary["search"], summary["board"])
	}
	for _, sample := range samples {
		if sample.RSS > 512<<20 {
			t.Errorf("RSS peak over 512MiB: %+v", sample)
		}
	}
	if len(samples) >= 10 && samples[len(samples)-1].Heap > samples[4].Heap+32<<20 {
		t.Error("retained Go heap grew by more than 32MiB after warm-up")
	}
	if _, err = store.db.Exec("INSERT INTO entry_fts(entry_fts,rank) VALUES('integrity-check',1)"); err != nil {
		t.Error(err)
	}
	backup := filepath.Join(t.TempDir(), "soak-backup.db")
	if err = backupDatabase(store.path, backup); err != nil {
		t.Fatal(err)
	}
	databasePath := store.path
	store.Close()
	reopened, err := OpenStore(databasePath)
	if err != nil {
		t.Fatal("restart", err)
	}
	page, err := reopened.Entries(EntryFilter{Query: "서울"})
	if err != nil || page.Total != MaxEntries {
		t.Fatal("restart reading state", err, page.Total)
	}
	reopened.Close()
	restoredPath := filepath.Join(t.TempDir(), "restored.db")
	if err = restoreDatabase(backup, restoredPath); err != nil {
		t.Fatal(err)
	}
	restored, err := OpenStore(restoredPath)
	if err != nil {
		t.Fatal(err)
	}
	restoredPage, err := restored.Entries(EntryFilter{Status: "saved"})
	restored.Close()
	if err != nil || restoredPage.Total == 0 {
		t.Fatal("restore saved state", err)
	}
	report := struct {
		Seconds        float64                `json:"seconds"`
		Clients        int                    `json:"clients"`
		Sources        int                    `json:"sources"`
		Entries        int                    `json:"entries"`
		Requests       int                    `json:"requests"`
		Failures       int                    `json:"failures"`
		RateLimited    int                    `json:"expected_refresh_429"`
		Fetches        int64                  `json:"provider_fetches"`
		Duplicates     int                    `json:"duplicate_fetches"`
		Latency        map[string]soakLatency `json:"latency"`
		Memory         []soakMemory           `json:"memory"`
		RestartRestore bool                   `json:"restart_restore"`
	}{elapsed.Seconds(), 8, active, entries, requests, failures, limited, fetches.Load(), duplicates, summary, samples, true}
	encoded, _ := json.Marshal(report)
	t.Logf("SOAK_RESULT %s", encoded)
}
