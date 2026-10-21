package dashboard

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const sessionLifetime = 12 * time.Hour
const sessionCookie = "glance_session"

type loginAttempts struct {
	count int
	until time.Time
}
type ownerAuth struct {
	passwordHash  [32]byte
	key           [32]byte
	enabled       bool
	secure        bool
	mu            sync.Mutex
	attempts      map[string]loginAttempts
	basicAttempts map[string]loginAttempts
	sessions      map[string]time.Time
	now           func() time.Time
}

func newOwnerAuth(token string) *ownerAuth {
	a := &ownerAuth{passwordHash: sha256.Sum256([]byte(token)), enabled: token != "", secure: strings.HasPrefix(os.Getenv("GLANCE_DASHBOARD_PUBLIC_URL"), "https://"), attempts: map[string]loginAttempts{}, basicAttempts: map[string]loginAttempts{}, sessions: map[string]time.Time{}, now: time.Now}
	if _, err := rand.Read(a.key[:]); err != nil {
		panic(err)
	}
	return a
}
func (a *ownerAuth) validPassword(password string) bool {
	hash := sha256.Sum256([]byte(password))
	return subtle.ConstantTimeCompare(hash[:], a.passwordHash[:]) == 1
}
func (a *ownerAuth) signedCookie() string {
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		panic(err)
	}
	payload := strconv.FormatInt(a.now().Unix(), 10) + "." + base64.RawURLEncoding.EncodeToString(nonce[:])
	a.mu.Lock()
	for key, until := range a.sessions {
		if !until.After(a.now()) {
			delete(a.sessions, key)
		}
	}
	if len(a.sessions) >= 32 {
		var oldest string
		var until time.Time
		for key, value := range a.sessions {
			if oldest == "" || value.Before(until) {
				oldest = key
				until = value
			}
		}
		delete(a.sessions, oldest)
	}
	a.sessions[payload] = a.now().Add(sessionLifetime)
	a.mu.Unlock()
	mac := hmac.New(sha256.New, a.key[:])
	_, _ = mac.Write([]byte(payload))
	return payload + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
func (a *ownerAuth) authenticated(r *http.Request) bool {
	if !a.enabled {
		return true
	}
	if _, password, ok := r.BasicAuth(); ok {
		if a.basicBlocked(r, false) {
			return false
		}
		if len(password) <= 4096 && a.validPassword(password) {
			return true
		}
		a.basicBlocked(r, true)
	}
	cookie, err := r.Cookie(sessionCookie)
	if err != nil || len(cookie.Value) > 256 {
		return false
	}
	parts := strings.Split(cookie.Value, ".")
	if len(parts) != 3 {
		return false
	}
	stamp, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return false
	}
	issued := time.Unix(stamp, 0)
	if issued.After(a.now().Add(time.Minute)) || a.now().Sub(issued) >= sessionLifetime {
		return false
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return false
	}
	payload := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, a.key[:])
	_, _ = mac.Write([]byte(payload))
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return false
	}
	a.mu.Lock()
	until, ok := a.sessions[payload]
	a.mu.Unlock()
	return ok && until.After(a.now())
}
func (a *ownerAuth) limited(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	now := a.now()
	for key, state := range a.attempts {
		if !state.until.After(now) {
			delete(a.attempts, key)
		}
	}
	state := a.attempts[host]
	if state.count >= 5 || len(a.attempts) >= 1024 && state.count == 0 {
		return true
	}
	if state.count == 0 {
		state.until = now.Add(time.Minute)
	}
	state.count++
	a.attempts[host] = state
	return false
}
func (a *ownerAuth) register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/auth/session", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 200, map[string]bool{"enabled": a.enabled, "authenticated": a.authenticated(r)})
	})
	mux.HandleFunc("POST /api/v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if !sameOrigin(r) {
			apiError(w, 403, "같은 사이트에서 로그인해 주세요")
			return
		}
		if a.limited(r) {
			w.Header().Set("Retry-After", "60")
			apiError(w, 429, "시도가 너무 많습니다. 1분 후 다시 시도하세요.")
			return
		}
		mediaType, _, mediaErr := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if mediaErr != nil || mediaType != "application/json" {
			apiError(w, 415, "JSON 요청이 필요합니다")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 32768)
		defer r.Body.Close()
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		var body struct {
			Password string `json:"password"`
		}
		var extra any
		if err := decoder.Decode(&body); err != nil || decoder.Decode(&extra) != io.EOF {
			apiError(w, 400, "로그인 요청을 확인하세요")
			return
		}
		if len(body.Password) > 4096 {
			apiError(w, 400, "비밀번호 길이를 확인하세요")
			return
		}
		if !a.enabled || !a.validPassword(body.Password) {
			apiError(w, 401, "비밀번호를 확인하세요.")
			return
		}
		http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: a.signedCookie(), Path: "/", HttpOnly: true, Secure: a.secure || r.TLS != nil, SameSite: http.SameSiteStrictMode, MaxAge: int(sessionLifetime.Seconds()), Expires: a.now().Add(sessionLifetime)})
		jsonResponse(w, 200, map[string]bool{"authenticated": true})
	})
	mux.HandleFunc("POST /api/v1/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		if !sameOrigin(r) {
			apiError(w, 403, "같은 사이트에서 로그아웃해 주세요")
			return
		}
		if cookie, err := r.Cookie(sessionCookie); err == nil {
			parts := strings.Split(cookie.Value, ".")
			if len(parts) == 3 {
				a.mu.Lock()
				delete(a.sessions, parts[0]+"."+parts[1])
				a.mu.Unlock()
			}
		}
		http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", HttpOnly: true, Secure: a.secure || r.TLS != nil, SameSite: http.SameSiteStrictMode, MaxAge: -1})
		jsonResponse(w, 200, map[string]bool{"authenticated": false})
	})
}

func loadOwnerToken() (string, error) {
	value := os.Getenv("GLANCE_DASHBOARD_TOKEN")
	path := os.Getenv("GLANCE_DASHBOARD_TOKEN_FILE")
	if value != "" && path != "" {
		return "", fmt.Errorf("configure either GLANCE_DASHBOARD_TOKEN or GLANCE_DASHBOARD_TOKEN_FILE")
	}
	if path != "" {
		file, err := os.Open(path)
		if err != nil {
			return "", fmt.Errorf("cannot open owner credential file: %w", err)
		}
		defer file.Close()
		data, err := io.ReadAll(io.LimitReader(file, 4097))
		if err != nil || len(data) > 4096 {
			return "", fmt.Errorf("owner credential file must be at most 4096 bytes")
		}
		value = strings.TrimSpace(string(data))
	}
	if len(value) > 4096 {
		return "", fmt.Errorf("owner credential must be at most 4096 bytes")
	}
	return value, nil
}

func peerAddress(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// Count failed Basic attempts separately from browser login and never trust
// caller-supplied forwarding headers. Valid health/read traffic consumes no quota.
func (a *ownerAuth) basicBlocked(r *http.Request, failure bool) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	now := a.now()
	for key, state := range a.basicAttempts {
		if !state.until.After(now) {
			delete(a.basicAttempts, key)
		}
	}
	host := peerAddress(r)
	state := a.basicAttempts[host]
	if state.count >= 10 || (state.count == 0 && len(a.basicAttempts) >= 1024) {
		return true
	}
	if failure {
		if state.count == 0 {
			state.until = now.Add(time.Minute)
		}
		state.count++
		a.basicAttempts[host] = state
	}
	return false
}
