// Package dashboard implements the editable, single-owner dashboard mode.
// It is independent of the legacy YAML renderer and shares its deployment binary.
package dashboard

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	_ "time/tzdata"
)

//go:embed ui
var webFiles embed.FS

func jsonResponse(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

func apiError(w http.ResponseWriter, status int, message string) {
	jsonResponse(w, status, ErrorResponse{Error: message, Code: strings.ToLower(strings.ReplaceAll(http.StatusText(status), " ", "_"))})
}

// Validation previews share commit validation without touching storage or providers.
func readBoardRequest(w http.ResponseWriter, r *http.Request) (Board, bool) {
	var b Board
	origin, err := url.Parse(r.Header.Get("Origin"))
	if err != nil || origin.Host != r.Host || (origin.Scheme != "https" && origin.Scheme != "http") || r.Header.Get("Sec-Fetch-Site") == "cross-site" {
		apiError(w, 403, "같은 사이트에서만 설정을 처리할 수 있습니다")
		return b, false
	}
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		apiError(w, 415, "JSON 요청이 필요합니다")
		return b, false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&b); err != nil {
		apiError(w, 400, "올바른 대시보드 JSON을 입력하세요")
		return b, false
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		apiError(w, 400, "요청 본문은 JSON 하나여야 합니다")
		return b, false
	}
	if err = b.Validate(); err != nil {
		apiError(w, 400, err.Error())
		return b, false
	}
	return b, true
}

func NewHandler(store *Store, runtime *Runtime, token string) http.Handler {
	mux := http.NewServeMux()
	service := &DashboardService{store: store, runtime: runtime}
	auth := newOwnerAuth(token)
	auth.register(mux)
	mux.HandleFunc("GET /api/v1/catalog", func(w http.ResponseWriter, r *http.Request) { jsonResponse(w, 200, Catalog()) })
	mux.HandleFunc("GET /api/readyz", func(w http.ResponseWriter, r *http.Request) {
		if err := store.db.PingContext(r.Context()); err != nil {
			apiError(w, 503, "저장소를 준비하지 못했습니다")
			return
		}
		if _, err := fs.Stat(webFiles, "ui/dist/index.html"); err != nil {
			apiError(w, 503, "화면을 준비하지 못했습니다")
			return
		}
		jsonResponse(w, 200, map[string]string{"status": "ready"})
	})
	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := store.db.PingContext(r.Context()); err != nil {
			apiError(w, 503, "저장소에 연결할 수 없습니다")
			return
		}
		jsonResponse(w, 200, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /api/v1/dashboard", func(w http.ResponseWriter, r *http.Request) {
		b, err := service.Board("home")
		if err != nil {
			apiError(w, 500, "대시보드를 읽을 수 없습니다")
			return
		}
		jsonResponse(w, 200, b)
	})
	mux.HandleFunc("PUT /api/v1/dashboard", func(w http.ResponseWriter, r *http.Request) {
		b, ok := readBoardRequest(w, r)
		if !ok {
			return
		}
		b, err := service.Commit(b)
		if errors.Is(err, ErrConflict) {
			apiError(w, 409, err.Error())
			return
		}
		if err != nil {
			log.Printf("dashboard save: %v", err)
			apiError(w, 500, "대시보드를 저장하지 못했습니다")
			return
		}
		jsonResponse(w, 200, b)
	})
	mux.HandleFunc("GET /api/v1/snapshots", func(w http.ResponseWriter, r *http.Request) {
		out, err := service.Snapshots("home")
		if err != nil {
			apiError(w, 500, "캐시를 읽을 수 없습니다")
			return
		}
		jsonResponse(w, 200, out)
	})
	mux.HandleFunc("POST /api/v1/dashboard/validate", func(w http.ResponseWriter, r *http.Request) {
		b, ok := readBoardRequest(w, r)
		if ok {
			jsonResponse(w, 200, b)
		}
	})
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) { apiError(w, 404, "API를 찾을 수 없습니다") })
	assets, assetErr := fs.Sub(webFiles, "ui/dist")
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if assetErr != nil {
			http.Error(w, "Dashboard UI is not built. Run: cd web && npm ci && npm run build", 503)
			return
		}
		if r.URL.Path != "/" && !strings.HasPrefix(r.URL.Path, "/assets/") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-cache")
		if strings.HasPrefix(r.URL.Path, "/assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		http.FileServer(http.FS(assets)).ServeHTTP(w, r)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("dashboard request panic: %T", recovered)
				apiError(w, 500, "요청을 처리하지 못했습니다")
			}
		}()
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
		public := r.URL.Path == "/" || strings.HasPrefix(r.URL.Path, "/assets/") || r.URL.Path == "/api/v1/auth/session" || r.URL.Path == "/api/v1/auth/login" || r.URL.Path == "/api/v1/auth/logout"
		if !public && !auth.authenticated(r) {
			apiError(w, 401, "로그인이 필요합니다.")
			return
		}
		mux.ServeHTTP(w, r)
	})
}

func Run(args []string) int {
	if len(args) > 0 {
		switch args[0] {
		case "backup", "restore", "verify-backup", "healthcheck":
			return maintenance(args[0], args[1:])
		}
	}
	flags := flag.NewFlagSet("glance dashboard", flag.ContinueOnError)
	listen := flags.String("listen", "127.0.0.1:8090", "HTTP listen address (non-loopback requires GLANCE_DASHBOARD_TOKEN)")
	path := flags.String("db", "data/dashboard.db", "SQLite database path")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 1
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "unexpected dashboard arguments")
		return 1
	}
	host, _, err := net.SplitHostPort(*listen)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	token, err := loadOwnerToken()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if publicURL := os.Getenv("GLANCE_DASHBOARD_PUBLIC_URL"); publicURL != "" {
		u, parseErr := url.Parse(publicURL)
		if parseErr != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "" && u.Path != "/" {
			fmt.Fprintln(os.Stderr, "GLANCE_DASHBOARD_PUBLIC_URL must be an HTTPS origin")
			return 1
		}
	}
	ip := net.ParseIP(host)
	loopback := host == "localhost" || (ip != nil && ip.IsLoopback())
	if (!loopback || token != "") && len(token) < 16 {
		fmt.Fprintln(os.Stderr, "Set GLANCE_DASHBOARD_TOKEN to at least 16 characters before listening outside loopback.")
		return 1
	}
	store, err := OpenStore(*path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer store.Close()
	runtime := NewRuntime(store, NewProviders().Fetch)
	defer runtime.Close()
	server := &http.Server{Addr: *listen, Handler: NewHandler(store, runtime, token), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	shutdownDone := make(chan struct{})
	shutdownFinished := make(chan struct{})
	go func() {
		defer close(shutdownFinished)
		select {
		case <-ctx.Done():
			shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := server.Shutdown(shutdown); err != nil {
				_ = server.Close()
			}
		case <-shutdownDone:
		}
	}()
	log.Printf("Glance dashboard listening on %s", *listen)
	err = server.ListenAndServe()
	close(shutdownDone)
	<-shutdownFinished
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}
