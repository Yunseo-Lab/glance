package dashboard

import (
	"context"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestSourcePauseDiagnosticsAndSemanticAliases(t *testing.T) {
	store := testStore(t)
	home, _ := store.Board()
	source := home.Sources[0]
	alias := source
	alias.ID = newID()
	home.Sources = append(home.Sources, alias)
	if _, err := store.Save(home); err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	runtime := NewRuntime(store, func(context.Context, Source) (any, error) {
		calls.Add(1)
		return FeedData{Title: "private-body-must-not-be-in-diagnostics", Items: []FeedItem{}}, nil
	})
	defer runtime.Close()
	if err := store.PauseSource(source.ID, true); err != nil {
		t.Fatal(err)
	}
	snapshot, err := runtime.Read(source)
	if err != nil || snapshot.State != "paused" || calls.Load() != 0 {
		t.Fatal("paused source fetched")
	}
	runtime.Read(alias)
	eventually(t, func() bool { snapshot, _ := runtime.Read(alias); return snapshot.State == "fresh" })
	statuses, err := store.SourceStatuses(time.Now())
	if err != nil || len(statuses) != 2 || len(statuses[0].BoardIDs) != 1 {
		t.Fatal("missing source status", err)
	}
	handler := NewHandler(store, runtime, "")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("GET", "/api/v1/diagnostics", nil))
	if response.Code != 200 || strings.Contains(response.Body.String(), "private-body") {
		t.Fatal("diagnostics leaked payload")
	}
	for _, status := range statuses {
		if status.Snapshot.Data != nil {
			t.Fatal("status exposed body")
		}
	}
	if err = store.PauseSource(source.ID, false); err != nil {
		t.Fatal(err)
	}
	snapshot, err = runtime.Read(source)
	if err != nil || snapshot.State != "fresh" {
		t.Fatal("resumed alias lost shared cache")
	}
}
