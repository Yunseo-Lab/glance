package dashboard

import (
	"bytes"
	"compress/gzip"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"
)

// Static assets contain only the public application shell, never owner data.
// Precompression keeps request-time CPU bounded and preserves HEAD/range behavior.
func compressedStatic(files fs.FS) http.Handler {
	compressed := map[string][]byte{}
	_ = fs.WalkDir(files, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		if ext := path.Ext(name); ext != ".js" && ext != ".css" && ext != ".html" {
			return nil
		}
		raw, err := fs.ReadFile(files, name)
		if err != nil {
			return nil
		}
		var buffer bytes.Buffer
		writer := gzip.NewWriter(&buffer)
		_, _ = writer.Write(raw)
		_ = writer.Close()
		compressed[name] = buffer.Bytes()
		return nil
	})
	fallback := http.FileServer(http.FS(files))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Accept-Encoding")
		name := strings.TrimPrefix(r.URL.Path, "/")
		if name == "" {
			name = "index.html"
		}
		if data, ok := compressed[name]; ok && acceptsGzip(r.Header.Get("Accept-Encoding")) {
			w.Header().Set("Content-Encoding", "gzip")
			w.Header().Set("Content-Length", strconv.Itoa(len(data)))
			w.Header().Set("Content-Type", mime.TypeByExtension(path.Ext(name)))
			http.ServeContent(w, r, path.Base(name), time.Time{}, bytes.NewReader(data))
			return
		}
		fallback.ServeHTTP(w, r)
	})
}

func acceptsGzip(header string) bool {
	wildcard := false
	for _, item := range strings.Split(header, ",") {
		parts := strings.Split(item, ";")
		coding := strings.ToLower(strings.TrimSpace(parts[0]))
		allowed := true
		for _, parameter := range parts[1:] {
			key, value, found := strings.Cut(strings.TrimSpace(parameter), "=")
			if found && strings.EqualFold(key, "q") {
				quality, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
				allowed = err == nil && quality > 0 && quality <= 1
			}
		}
		if coding == "gzip" {
			return allowed
		}
		if coding == "*" {
			wildcard = allowed
		}
	}
	return wildcard
}

func publicShellPath(name string) bool {
	switch name {
	case "/", "/sw.js", "/manifest.webmanifest", "/icon.svg", "/icon-192.png", "/icon-512.png":
		return true
	}
	return strings.HasPrefix(name, "/assets/")
}
