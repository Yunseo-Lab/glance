package dashboard

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOriginMustMatchConfiguredHTTPSAndIgnoresForwardedHeaders(t *testing.T) {
	t.Setenv("GLANCE_DASHBOARD_PUBLIC_URL", "https://dashboard.example.com")
	for _, test := range []struct {
		origin string
		want   bool
	}{{"https://dashboard.example.com", true}, {"https://dashboard.example.com:443", true}, {"http://dashboard.example.com", false}, {"https://evil@dashboard.example.com", false}, {"https://dashboard.example.com/path", false}, {"https://dashboard.example.com?q=x", false}, {"null", false}} {
		request := httptest.NewRequest("POST", "http://dashboard.example.com/api/v1/auth/login", nil)
		request.Header.Set("Origin", test.origin)
		request.Header.Set("X-Forwarded-Proto", "https")
		request.Header.Set("X-Forwarded-Host", "evil.example")
		if got := sameOrigin(request); got != test.want {
			t.Errorf("%s: %v", test.origin, got)
		}
	}
	request := httptest.NewRequest("POST", "http://evil.example/api/v1/dashboard", nil)
	request.Header.Set("Origin", "https://dashboard.example.com")
	request.Header.Set("X-Forwarded-Host", "dashboard.example.com")
	if sameOrigin(request) {
		t.Fatal("forged forwarded host trusted")
	}
}
func TestProductionHostGuardRejectsDNSRebindingHost(t *testing.T) {
	handler := guardHost(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }), "https://dashboard.example.com")
	for _, test := range []struct {
		host string
		want int
	}{{"dashboard.example.com", 204}, {"127.0.0.1:8080", 204}, {"localhost:8090", 204}, {"[::1]:8090", 204}, {"rebind.evil.example", 400}} {
		request := httptest.NewRequest("GET", "http://"+test.host+"/", nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != test.want {
			t.Fatal(test.host, response.Code)
		}
	}
}

func TestProductionServerRejectsOversizedHeadersBeforeHandler(t *testing.T) {
	called := false
	server := httptest.NewUnstartedServer(nil)
	server.Config = dashboardHTTPServer("", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true; w.WriteHeader(204) }))
	server.Start()
	defer server.Close()
	request, _ := http.NewRequest("GET", server.URL, nil)
	request.Header.Set("X-Large", strings.Repeat("x", 64<<10))
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != 431 || called {
		t.Fatal("oversized headers reached handler", response.StatusCode)
	}
}
