package dashboard

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestConnectionResponseCannotExposeStoredCredentials(t *testing.T) {
	store := testStore(t)
	secret := "super-sensitive-credential-marker"
	_, err := store.db.Exec("INSERT INTO connections(id,owner_id,kind,public_config,encrypted_secret,name) VALUES('test','local','github',?,?,?)", raw(connectionPublic{Endpoint: "https://api.github.com"}), []byte(secret), "GitHub")
	if err != nil {
		t.Fatal(err)
	}
	connections, err := store.Connections()
	if err != nil || len(connections) != 1 || !connections[0].HasSecret {
		t.Fatal("missing safe metadata", err)
	}
	data, _ := json.Marshal(connections)
	if strings.Contains(string(data), secret) || strings.Contains(string(data), "encrypted_secret") {
		t.Fatal("connection DTO exposed private storage")
	}
	handler := NewHandler(store, nil, "owner-private-password")
	response := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/api/v1/connections", nil)
	handler.ServeHTTP(response, request)
	if response.Code != 401 {
		t.Fatal("connections public")
	}
	request.SetBasicAuth("owner", "owner-private-password")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != 200 || strings.Contains(response.Body.String(), secret) {
		t.Fatal("connection API exposed private storage")
	}
	board, _ := store.Board()
	export, _ := json.Marshal(board)
	if strings.Contains(string(export), secret) {
		t.Fatal("board export included credential")
	}
}
