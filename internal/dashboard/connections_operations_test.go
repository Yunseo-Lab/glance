package dashboard

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestConnectionCreateTestAndRevokeNeverStoresPlaintext(t *testing.T) {
	store := testStore(t)
	token := "only-a-test-token-for-fixtures-12345"
	connection, err := store.CreateConnection(ConnectionInput{Name: "GitHub", Kind: "github", Token: token})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(store.path)
	if err != nil {
		t.Fatal(err)
	}
	wal, _ := os.ReadFile(store.path + "-wal")
	if strings.Contains(string(data)+string(wal), token) {
		t.Fatal("plaintext credential in SQLite")
	}
	provider := NewProviders()
	provider.client = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Header.Get("Authorization") != "Bearer "+token {
			t.Error("missing credential header")
		}
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"id":1,"login":"fixture-user"}`))}, nil
	})}
	report, err := store.TestConnection(context.Background(), connection.ID, provider)
	if err != nil || !report.Connected || report.Identity != "fixture-user" {
		t.Fatal("connection fixture failed", err)
	}
	_, err = store.TestConnection(context.Background(), connection.ID, provider)
	var cooldown *CooldownError
	if !errors.As(err, &cooldown) {
		t.Fatal("connection test cooldown bypassed")
	}
	if err = store.RevokeConnection(connection.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err = store.resolveConnection(connection.ID); err == nil {
		t.Fatal("revoked secret still available")
	}
	public, err := store.connection(connection.ID)
	if err != nil || public.HasSecret || public.Status != "revoked" {
		t.Fatal("revocation did not clear secret")
	}
}
func TestPrivateCalendarURLIsEncryptedAndOnlyOriginIsPublic(t *testing.T) {
	store := testStore(t)
	url := "https://example.com/private-calendar-token/feed.ics?secret=fixture"
	connection, err := store.CreateConnection(ConnectionInput{Name: "Schedule", Kind: "calendar", Endpoint: url})
	if err != nil {
		t.Fatal(err)
	}
	if connection.Endpoint != "https://example.com" {
		t.Fatal("private URL leaked in public metadata")
	}
	_, secret, err := store.resolveConnection(connection.ID)
	if err != nil || secret.URL != url {
		t.Fatal("private calendar URL lost")
	}
}
