package dashboard

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func authRequest(handler http.Handler, path, body, origin string, cookie *http.Cookie) *httptest.ResponseRecorder {
	base := os.Getenv("GLANCE_DASHBOARD_PUBLIC_URL")
	if base == "" {
		base = "http://localhost"
	}
	r := httptest.NewRequest("POST", base+path, strings.NewReader(body))
	r.Header.Set("Origin", origin)
	r.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		r.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	return w
}
func TestOwnerSessionLoginExpiryTamperAndLogout(t *testing.T) {
	t.Setenv("GLANCE_DASHBOARD_PUBLIC_URL", "https://dashboard.example")
	a := newOwnerAuth("test-owner-password-long-enough")
	now := time.Now()
	a.now = func() time.Time { return now }
	mux := http.NewServeMux()
	a.register(mux)
	if w := authRequest(mux, "/api/v1/auth/login", `{"password":"test-owner-password-long-enough"}`, "http://evil.example", nil); w.Code != 403 {
		t.Fatal(w.Code)
	}
	w := authRequest(mux, "/api/v1/auth/login", `{"password":"test-owner-password-long-enough"}`, "https://dashboard.example", nil)
	if w.Code != 200 {
		t.Fatalf("login: %d %s", w.Code, w.Body.String())
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatal("missing cookie")
	}
	cookie := cookies[0]
	if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteStrictMode || cookie.MaxAge != 43200 {
		t.Fatalf("unsafe cookie flags: %+v", cookie)
	}
	r := httptest.NewRequest("GET", "/api/v1/dashboard", nil)
	r.AddCookie(cookie)
	if !a.authenticated(r) {
		t.Fatal("valid cookie rejected")
	}
	tampered := *cookie
	tampered.Value += "x"
	bad := httptest.NewRequest("GET", "/", nil)
	bad.AddCookie(&tampered)
	if a.authenticated(bad) {
		t.Fatal("tampered cookie accepted")
	}
	now = now.Add(sessionLifetime + time.Second)
	if a.authenticated(r) {
		t.Fatal("expired cookie accepted")
	}
	now = now.Add(-sessionLifetime - time.Second)
	if w = authRequest(mux, "/api/v1/auth/logout", "", "https://dashboard.example", cookie); w.Code != 200 {
		t.Fatal(w.Code)
	}
	if a.authenticated(r) {
		t.Fatal("logged-out cookie can be replayed")
	}
}
func TestLoginAttemptsAreThrottledAndExpire(t *testing.T) {
	a := newOwnerAuth("test-owner-password-long-enough")
	now := time.Now()
	a.now = func() time.Time { return now }
	mux := http.NewServeMux()
	a.register(mux)
	for i := 0; i < 5; i++ {
		if w := authRequest(mux, "/api/v1/auth/login", `{"password":"incorrect"}`, "http://localhost", nil); w.Code != 401 {
			t.Fatal(w.Code)
		}
	}
	if w := authRequest(mux, "/api/v1/auth/login", `{"password":"incorrect"}`, "http://localhost", nil); w.Code != 429 || w.Header().Get("Retry-After") == "" {
		t.Fatal("missing throttling")
	}
	now = now.Add(time.Minute + time.Second)
	if w := authRequest(mux, "/api/v1/auth/login", `{"password":"test-owner-password-long-enough"}`, "http://localhost", nil); w.Code != 200 {
		t.Fatal(w.Code)
	}
}
func TestOwnerTokenFileAndConflictingConfiguration(t *testing.T) {
	t.Setenv("GLANCE_DASHBOARD_TOKEN", "set")
	t.Setenv("GLANCE_DASHBOARD_TOKEN_FILE", "/not-used")
	if _, err := loadOwnerToken(); err == nil {
		t.Fatal("conflicting credential sources accepted")
	}
}

func TestBasicFailuresAreBoundedAndForwardedHeadersCannotBypassThem(t *testing.T) {
	auth := newOwnerAuth("a-long-owner-fixture-password")
	now := time.Now()
	auth.now = func() time.Time { return now }
	request := httptest.NewRequest("GET", "/api/v1/dashboard", nil)
	request.RemoteAddr = "192.0.2.1:1234"
	for i := 0; i < 200; i++ {
		request.SetBasicAuth("owner", "a-long-owner-fixture-password")
		if !auth.authenticated(request) {
			t.Fatal("valid reads consumed failure quota")
		}
	}
	for i := 0; i < 10; i++ {
		request.Header.Set("X-Forwarded-For", fmt.Sprintf("203.0.113.%d", i))
		request.SetBasicAuth("owner", "wrong")
		if auth.authenticated(request) {
			t.Fatal("wrong Basic credential accepted")
		}
	}
	request.SetBasicAuth("owner", "a-long-owner-fixture-password")
	if auth.authenticated(request) {
		t.Fatal("failure limit bypassed")
	}
	now = now.Add(time.Minute + time.Second)
	if !auth.authenticated(request) {
		t.Fatal("expired Basic limit not recovered")
	}
	if len(auth.basicAttempts) > 1 {
		t.Fatal("forged forwarding addresses created rate-limit identities")
	}
}
