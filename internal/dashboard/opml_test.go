package dashboard

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOPMLDeduplicatesPublicFeedsAndPreservesFolders(t *testing.T) {
	document := `<?xml version="1.0"?><opml version="2.0"><head><title>Feeds</title></head><body><outline text="개발"><outline text="RSSHub" xmlUrl="https://rsshub.app/github/issue/owner/repo?a=1&amp;b=2"/><outline text="Same" xmlUrl="https://rsshub.app/github/issue/owner/repo?b=2&amp;a=1"/><outline text="Bridge" xmlUrl="https://bridge.example.com/?action=display&amp;bridge=Example&amp;format=Atom"/></outline></body></opml>`
	preview, err := parseOPML(document)
	if err != nil || len(preview.Feeds) != 2 || preview.DuplicateCount != 1 || preview.Feeds[0].Folder != "개발" {
		t.Fatal(preview, err)
	}
	for _, document := range []string{`<!DOCTYPE opml [<!ENTITY x SYSTEM "https://private.example.com">]><opml/>`, `<opml><body><outline xmlUrl="http://example.com/rss"/></body></opml>`, `<opml><outline xmlUrl="https://127.0.0.1/private"/></opml>`, `<opml><outline xmlUrl="javascript:alert(1)"/></opml>`, `<opml><outline xmlUrl="https://user:password@example.com/feed"/></opml>`, `<broken`, strings.Repeat("x", 524289), `<opml>` + strings.Repeat(`<outline>`, 33) + strings.Repeat(`</outline>`, 33) + `</opml>`} {
		if _, err := parseOPML(document); err == nil {
			t.Fatal("unsafe OPML accepted")
		}
	}
}
func TestOPMLPreviewDoesNotWriteBoardOrSources(t *testing.T) {
	store := testStore(t)
	before, _ := store.Board()
	handler := NewHandler(store, nil, "")
	document := `<opml version="2.0"><body><outline text="Example" xmlUrl="https://example.com/feed"/></body></opml>`
	payload, _ := json.Marshal(map[string]string{"document": document})
	request := httptest.NewRequest("POST", "http://localhost/api/v1/imports/opml/preview", bytes.NewReader(payload))
	request.Header.Set("Origin", "http://localhost")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != 200 {
		t.Fatal(response.Code, response.Body.String())
	}
	after, _ := store.Board()
	if string(raw(before)) != string(raw(after)) {
		t.Fatal("preview mutated board")
	}
	sources, _ := store.Sources()
	if len(sources) != len(before.Sources) {
		t.Fatal("preview persisted sources")
	}
}
