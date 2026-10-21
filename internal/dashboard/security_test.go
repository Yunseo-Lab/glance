package dashboard

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestDNSRebindingCannotReachTheDialer(t *testing.T) {
	provider := NewProviders()
	lookups := 0
	dialed := false
	provider.proxy = func(*http.Request) (*url.URL, error) { return nil, nil }
	provider.resolve = func(context.Context, string) ([]net.IP, error) {
		lookups++
		if lookups == 1 {
			return []net.IP{net.ParseIP("8.8.8.8")}, nil
		}
		return []net.IP{net.ParseIP("127.0.0.1")}, nil
	}
	provider.dial = func(context.Context, string, string) (net.Conn, error) {
		dialed = true
		return nil, errors.New("fixture dial must never run")
	}
	request, _ := http.NewRequest("GET", "https://example.com/feed", nil)
	if err := provider.checkRequest(request); err != nil {
		t.Fatal(err)
	}
	transport := provider.client.Transport.(*http.Transport)
	if _, err := transport.DialContext(context.Background(), "tcp", "example.com:443"); err == nil || dialed {
		t.Fatal("rebound private IP reached dialer")
	}
}
func TestRedirectStripsAuthAndRejectsPrivateTarget(t *testing.T) {
	provider := NewProviders()
	from, _ := http.NewRequest("GET", "https://github.com/feed", nil)
	next, _ := http.NewRequest("GET", "https://raw.githubusercontent.com/feed", nil)
	next.Header.Set("Authorization", "Bearer fixture")
	if err := provider.client.CheckRedirect(next, []*http.Request{from}); err != nil {
		t.Fatal(err)
	}
	if next.Header.Get("Authorization") != "" {
		t.Fatal("credential forwarded across origins")
	}
	private, _ := http.NewRequest("GET", "https://127.0.0.1/private", nil)
	if provider.client.CheckRedirect(private, []*http.Request{from}) == nil {
		t.Fatal("private redirect accepted")
	}
}
func TestSpecialAddressFamiliesAndMaliciousFeeds(t *testing.T) {
	for _, address := range []string{"127.0.0.1", "10.0.0.1", "169.254.169.254", "100.64.0.1", "192.0.2.1", "0.0.0.0", "::1", "::ffff:127.0.0.1", "fc00::1", "fe80::1", "64:ff9b::7f00:1", "2001:db8::1"} {
		if publicIP(net.ParseIP(address)) {
			t.Error("unsafe address accepted", address)
		}
	}
	provider := NewProviders()
	payload := `<!DOCTYPE rss [<!ENTITY xxe SYSTEM "file:///etc/passwd">]><rss><channel><title>&xxe;</title></channel></rss>`
	provider.client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(payload))}, nil
	})}
	if _, err := provider.Fetch(context.Background(), Source{ID: "malicious", Type: "rss", Config: raw(RSSSource{URL: "https://github.com/feed"})}); err == nil {
		t.Fatal("external XML entity accepted")
	}
}
func FuzzOriginPolicy(f *testing.F) {
	for _, input := range []string{"http://example.com", "null", "http://user@example.com", "http://example.com/path", "http://example.com?x=1", "https://example.com"} {
		f.Add(input)
	}
	f.Fuzz(func(t *testing.T, input string) {
		request := &http.Request{Host: "example.com", Header: http.Header{}}
		request.Header.Set("Origin", input)
		if sameOrigin(request) {
			u, err := url.Parse(input)
			if err != nil || u.User != nil || u.RawQuery != "" || u.Path != "" || u.Fragment != "" || u.Scheme != "http" || originHost(u) != "example.com" {
				t.Fatal("invalid same-origin accepted")
			}
		}
	})
}
func FuzzEncryptedConnectionEnvelope(f *testing.F) {
	for _, input := range []string{"", `{}`, `{"v":999}`, `{"v":1,"keyVersion":1,"nonce":"AA==","data":"AA=="}`} {
		f.Add([]byte(input))
	}
	vault, _ := newVault(keyMaterial{Version: 1, Key: make([]byte, 32)})
	f.Fuzz(func(t *testing.T, input []byte) {
		secret, err := vault.decrypt("fixture", "github", input)
		if err == nil {
			encoded, err := vault.encrypt("fixture", "github", secret)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := vault.decrypt("fixture", "github", encoded)
			if err != nil || decoded != secret {
				t.Fatal("envelope round trip failed")
			}
		}
	})
}
