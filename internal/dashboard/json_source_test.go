package dashboard

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestJSONPointerScalarOnlyAndRFC6901Escapes(t *testing.T) {
	fixture := []byte(`{"a/b":{"~key":[42,false,null,"<script>text only</script>"]},"":7}`)
	for pointer, want := range map[string]string{"/a~1b/~0key/0": "42", "/a~1b/~0key/1": "false", "/a~1b/~0key/2": "정보 없음", "/a~1b/~0key/3": "<script>text only</script>", "/": "7"} {
		result, err := extractJSONValue(fixture, pointer)
		if err != nil || result.Value != want {
			t.Fatal(pointer, result, err)
		}
	}
	for _, pointer := range []string{"a/b", "/a~2b", "/a~", "/missing", "/a~1b/~0key/01", "/a~1b/~0key/-", "/a~1b", "/a~1b/~0key/99", strings.Repeat("/a", 17)} {
		if _, err := extractJSONValue(fixture, pointer); err == nil {
			t.Fatal("unsafe/missing pointer accepted", pointer)
		}
	}
	if result, err := extractJSONValue([]byte(`true`), ""); err != nil || result.Value != "true" {
		t.Fatal(result, err)
	}
	for _, body := range []string{`{broken`, `1 2`, `{"value":[]}`, `{"value":{}}`, `{"value":"` + strings.Repeat("x", 4097) + `"}`} {
		if _, err := extractJSONValue([]byte(body), "/value"); err == nil {
			t.Fatal("invalid/oversized value accepted")
		}
	}
	for _, config := range []string{`{"url":"https://example.com","pointer":"/value","script":"return document.cookie"}`, `{"url":"https://example.com","pointer":"/value","headers":{"Authorization":"secret"}}`, `{"url":"http://127.0.0.1","pointer":"/value"}`} {
		if validateJSONSource([]byte(config)) == nil {
			t.Fatal("executable/unsafe config accepted")
		}
	}
}
func TestPrivateJSONEndpointAndTokenRemainServerSide(t *testing.T) {
	store := testStore(t)
	connection, err := store.CreateConnection(ConnectionInput{Name: "Private JSON", Kind: "json", Endpoint: "https://api.example.com/private?key=fixture-private-url", Token: "fixture-private-bearer"})
	if err != nil {
		t.Fatal(err)
	}
	provider := &Providers{store: store, resolve: func(context.Context, string) ([]net.IP, error) { return []net.IP{net.ParseIP("8.8.8.8")}, nil }, proxy: func(*http.Request) (*url.URL, error) { return nil, nil }, client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != "GET" || r.URL.Host != "api.example.com" || r.Header.Get("Authorization") != "Bearer fixture-private-bearer" || r.Context().Value(sensitiveRequestKey{}) != true {
			t.Fatal("incorrect private request")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"reading":{"value":12.5}}`))}, nil
	})}}
	source := Source{ID: "json", Type: "json", ConnectionID: connection.ID, Config: raw(JSONSource{Pointer: "/reading/value"})}
	data, err := provider.Fetch(context.Background(), source)
	if err != nil || data.(JSONValueData).Value != "12.5" {
		t.Fatal(data, err)
	}
	if strings.Contains(string(raw(source)), "fixture-private") {
		t.Fatal("credential exported")
	}
}
