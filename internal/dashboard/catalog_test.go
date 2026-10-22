package dashboard

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"testing"
)

func legacyBoard(t *testing.T) Board {
	t.Helper()
	data, err := os.ReadFile("testdata/legacy-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var file struct {
		Board Board `json:"board"`
	}
	if err = json.Unmarshal(data, &file); err != nil {
		t.Fatal(err)
	}
	return file.Board
}
func TestCatalogPreservesLegacyWidgetAndProviderContracts(t *testing.T) {
	board := legacyBoard(t)
	if err := board.Validate(); err != nil {
		t.Fatal(err)
	}
	catalog := Catalog()
	if len(catalog.Widgets) < 6 || len(catalog.Providers) < 2 {
		t.Fatal("missing legacy definition")
	}
	for _, widget := range board.Widgets {
		definition, ok := widgetDefinitions[widget.Type]
		if !ok || definition.validate == nil {
			t.Fatal(widget.Type)
		}
		malformed := widget
		malformed.Config = raw(map[string]any{"unrecognized": true})
		copy := board
		copy.Widgets = []Widget{malformed}
		if copy.Validate() == nil {
			t.Fatal("unknown config accepted:", widget.Type)
		}
	}
	handler := NewHandler(nil, nil, "owner-password-16-characters")
	request := httptest.NewRequest("GET", "/api/v1/catalog", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != 401 {
		t.Fatal("catalog must respect owner authentication")
	}
	request.SetBasicAuth("owner", "owner-password-16-characters")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != 200 {
		t.Fatal(response.Code)
	}
	var document CatalogDocument
	if err := json.Unmarshal(response.Body.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	if document.Widgets[0].ConfigSchema["additionalProperties"] != false {
		t.Fatal("settings schema must reject unknown fields")
	}
}

func TestConfigVersionCompatibilityAndFutureRejection(t *testing.T) {
	board := legacyBoard(t)
	for _, version := range []int{0, 1} {
		for i := range board.Widgets {
			board.Widgets[i].ConfigVersion = version
		}
		for i := range board.Sources {
			board.Sources[i].ConfigVersion = version
		}
		if err := board.Validate(); err != nil {
			t.Fatal("legacy or current version rejected", err)
		}
	}
	for _, version := range []int{-1, 2, 999} {
		board.Widgets[0].ConfigVersion = version
		if board.Validate() == nil {
			t.Fatal("future widget config accepted")
		}
		board.Widgets[0].ConfigVersion = 1
		board.Sources[0].ConfigVersion = version
		if board.Validate() == nil {
			t.Fatal("future provider config accepted")
		}
		board.Sources[0].ConfigVersion = 1
	}
	if err := decodeConfig(json.RawMessage(`{} {}`), &struct{}{}); err == nil {
		t.Fatal("trailing config document accepted")
	}
}
