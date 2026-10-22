package dashboard

import (
	"compress/gzip"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestCompressedStatic(t *testing.T) {
	body := strings.Repeat("const publicShell = true;", 100)
	handler := compressedStatic(fstest.MapFS{"assets/app.js": {Data: []byte(body)}})
	for _, encoding := range []string{"gzip", "br, gzip;q=0.8", "*", "gzip;q=0, *;q=1", "gzip;q=invalid", "identity"} {
		req := httptest.NewRequest("GET", "/assets/app.js", nil)
		req.Header.Set("Accept-Encoding", encoding)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		got := response.Body.String()
		if response.Header().Get("Content-Encoding") == "gzip" {
			reader, err := gzip.NewReader(response.Body)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := io.ReadAll(reader)
			if err != nil {
				t.Fatal(err)
			}
			got = string(decoded)
		}
		if got != body || response.Code != 200 {
			t.Fatalf("%s: invalid body/status", encoding)
		}
		if encoding == "gzip;q=0, *;q=1" && response.Header().Get("Content-Encoding") != "" {
			t.Fatal("explicit gzip exclusion ignored")
		}
		if response.Header().Get("Vary") != "Accept-Encoding" {
			t.Fatal("missing Vary")
		}
	}
	req := httptest.NewRequest("HEAD", "/assets/app.js", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	if response.Body.Len() != 0 || response.Header().Get("Content-Length") == "" {
		t.Fatal("HEAD must describe the compressed representation without body")
	}
}
