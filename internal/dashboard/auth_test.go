package dashboard

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func authRequest(handler http.Handler, path, body, origin string, cookie *http.Cookie) *httptest.ResponseRecorder {
	r := httptest.NewRequest("POST", "http://localhost"+path, strings.NewReader(body))
	r.Header.Set("Origin", origin)
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
	w := authRequest(mux, "/api/v1/auth/login", `{"password":"test-owner-password-long-enough"}`, "http://localhost", nil)
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
	if w = authRequest(mux, "/api/v1/auth/logout", "", "http://localhost", cookie); w.Code != 200 {
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
