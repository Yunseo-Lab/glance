package dashboard

import (
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func originURL(value string) (*url.URL, bool) {
	u, err := url.Parse(value)
	if err != nil || strings.ContainsAny(value, "?#") || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Opaque != "" || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, false
	}
	return u, true
}
func originHost(u *url.URL) string {
	host := strings.ToLower(u.Hostname())
	if ip := net.ParseIP(host); ip != nil {
		host = ip.String()
	}
	port := u.Port()
	if u.Scheme == "https" && port == "443" || u.Scheme == "http" && port == "80" {
		port = ""
	}
	if port != "" {
		return net.JoinHostPort(host, port)
	}
	if strings.Contains(host, ":") {
		return "[" + host + "]"
	}
	return host
}
func sameOrigin(r *http.Request) bool {
	if len(r.Header.Values("Origin")) != 1 || strings.EqualFold(r.Header.Get("Sec-Fetch-Site"), "cross-site") {
		return false
	}
	actual, ok := originURL(r.Header.Get("Origin"))
	if !ok {
		return false
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	expected, ok := originURL(strings.TrimSuffix(os.Getenv("GLANCE_DASHBOARD_PUBLIC_URL"), "/"))
	if !ok {
		if os.Getenv("GLANCE_DASHBOARD_PUBLIC_URL") != "" {
			return false
		}
		expected, ok = originURL(scheme + "://" + r.Host)
		if !ok {
			return false
		}
	}
	requestHost, ok := originURL(expected.Scheme + "://" + r.Host)
	return ok && actual.Scheme == expected.Scheme && originHost(actual) == originHost(expected) && originHost(requestHost) == originHost(expected)
}
func guardHost(next http.Handler, publicURL string) http.Handler {
	allowed, _ := originURL(strings.TrimSuffix(publicURL, "/"))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, ok := originURL("http://" + r.Host)
		if !ok {
			apiError(w, 400, "허용하지 않는 호스트입니다")
			return
		}
		name := strings.ToLower(host.Hostname())
		ip := net.ParseIP(name)
		loopback := name == "localhost" || ip != nil && ip.IsLoopback()
		if !loopback {
			candidate, valid := originURL("https://" + r.Host)
			if allowed == nil || !valid || originHost(candidate) != originHost(allowed) {
				apiError(w, 400, "허용하지 않는 호스트입니다")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
