package dashboard

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestBoardLifecycleRevisionsIsolationAndLimits(t *testing.T) {
	store := testStore(t)
	home, _ := store.Board()
	board, err := store.CreateBoard("Work", &home)
	if err != nil {
		t.Fatal(err)
	}
	if board.ID == "home" || board.Revision != 1 || board.Widgets[0].ID == home.Widgets[0].ID {
		t.Fatal("clone reused board/widget identity")
	}
	board.Title = "Updated"
	saved, err := store.Save(board)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.Save(board); !errors.Is(err, ErrConflict) {
		t.Fatal("stale revision accepted")
	}
	if err = store.DeleteBoard(saved.ID, board.Revision); !errors.Is(err, ErrConflict) {
		t.Fatal("stale delete accepted")
	}
	if err = store.DeleteBoard("home", home.Revision); !errors.Is(err, ErrProtectedBoard) {
		t.Fatal("home deleted")
	}
	if err = store.DeleteBoard(saved.ID, saved.Revision); err != nil {
		t.Fatal(err)
	}
	if _, err = store.GetBoard(saved.ID); !errors.Is(err, ErrNotFound) {
		t.Fatal("deleted board visible")
	}
	foreign := home
	foreign.ID = "foreign"
	data, _ := json.Marshal(foreign)
	store.db.Exec("INSERT INTO boards VALUES(?,?,?,?)", foreign.ID, "other-owner", foreign.Revision, data)
	if _, err = store.GetBoard("foreign"); !errors.Is(err, ErrNotFound) {
		t.Fatal("other owner visible")
	}
	if _, err = store.Save(foreign); !errors.Is(err, ErrNotFound) {
		t.Fatal("other owner modified")
	}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := store.CreateBoard(fmt.Sprint(i), nil)
			if err != nil && !errors.Is(err, ErrLimit) {
				t.Error(err)
			}
		}(i)
	}
	wg.Wait()
	boards, err := store.Boards()
	if err != nil || len(boards) != MaxBoards || boards[0].ID != "home" {
		t.Fatal("non-atomic owner board limit", len(boards), err)
	}
}
func TestBoardAPIRequiresOriginMatchingIDAndOwnerScope(t *testing.T) {
	store := testStore(t)
	handler := NewHandler(store, nil, "")
	call := func(method, path string, body any, origin string) *httptest.ResponseRecorder {
		data, _ := json.Marshal(body)
		request := httptest.NewRequest(method, path, bytes.NewReader(data))
		request.Header.Set("Content-Type", "application/json")
		if origin != "" {
			request.Header.Set("Origin", origin)
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response
	}
	if r := call("POST", "/api/v1/boards", map[string]string{"title": "Work"}, ""); r.Code != 403 {
		t.Fatal(r.Code)
	}
	r := call("POST", "/api/v1/boards", map[string]string{"title": "Work"}, "http://example.com")
	if r.Code != 201 {
		t.Fatal(r.Code, r.Body.String())
	}
	var board Board
	json.Unmarshal(r.Body.Bytes(), &board)
	if r = call("PUT", "/api/v1/boards/home", board, "http://example.com"); r.Code != 400 {
		t.Fatal("mismatched ID accepted")
	}
	if r = call("PUT", "/api/v1/dashboard", board, "http://example.com"); r.Code != 400 {
		t.Fatal("legacy endpoint modified another board")
	}
	if r = call("GET", "/api/v1/boards/missing", nil, ""); r.Code != http.StatusNotFound {
		t.Fatal(r.Code)
	}
}
func TestPruningOneBoardKeepsOtherBoardsCache(t *testing.T) {
	store := testStore(t)
	home, _ := store.Board()
	_, err := store.CreateBoard("shared", &home)
	if err != nil {
		t.Fatal(err)
	}
	key := sourceKey(home.Sources[0])
	if err = store.putSnapshot(key, Snapshot{State: "fresh", Data: raw(FeedData{Title: "keep", Items: []FeedItem{}})}); err != nil {
		t.Fatal(err)
	}
	home.Sources = []Source{}
	home.Widgets = []Widget{}
	if _, err = store.Save(home); err != nil {
		t.Fatal(err)
	}
	if err = store.pruneSnapshots(nil); err != nil {
		t.Fatal(err)
	}
	snapshot, err := store.snapshot(key)
	if err != nil || len(snapshot.Data) == 0 {
		t.Fatal("other board cache pruned")
	}
}
