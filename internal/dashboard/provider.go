package dashboard

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
)

type Providers struct {
	resolve    func(context.Context, string) ([]net.IP, error)
	proxy      func(*http.Request) (*url.URL, error)
	dial       func(context.Context, string, string) (net.Conn, error)
	store      *Store
	client     *http.Client
	proxyHosts map[string]bool
}

func (p *Providers) checkRequest(req *http.Request) error {
	if _, err := publicURL(req.URL.String()); err != nil {
		return err
	}
	host := strings.ToLower(req.URL.Hostname())
	if ip := net.ParseIP(host); ip != nil && !publicIP(ip) {
		return errors.New("private network destination rejected")
	}
	proxy, err := p.proxyFor(req)
	if err != nil {
		return err
	}
	if proxy != nil {
		// Egress proxies resolve destinations remotely. Never require direct DNS
		// (unavailable in managed environments), or permit arbitrary proxy targets.
		// Only built-in providers and administrator-approved hosts use this route.
		if !p.proxyHosts[host] {
			return errors.New("RSS host must be approved in GLANCE_DASHBOARD_FEED_HOSTS when using an egress proxy")
		}
		return nil
	}
	return p.checkDestination(req.Context(), req.URL)
}

func publicIP(ip net.IP) bool {
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return false
	}
	addr = addr.Unmap()
	if !addr.IsGlobalUnicast() || addr.IsPrivate() || addr.IsLoopback() || addr.IsLinkLocalUnicast() {
		return false
	}
	// Also reject special-use IPv4 and IPv6 translation ranges.
	for _, prefix := range []string{"0.0.0.0/8", "100.64.0.0/10", "192.0.0.0/24", "192.0.2.0/24", "198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24", "240.0.0.0/4", "2001:db8::/32", "64:ff9b::/96"} {
		if netip.MustParsePrefix(prefix).Contains(addr) {
			return false
		}
	}
	return true
}
func (p *Providers) checkDestination(ctx context.Context, u *url.URL) error {
	if _, err := publicURL(u.String()); err != nil {
		return err
	}
	ips, err := p.resolveHost(ctx, u.Hostname())
	if err != nil {
		return err
	}
	if len(ips) == 0 {
		return errors.New("hostname has no addresses")
	}
	for _, ip := range ips {
		if !publicIP(ip) {
			return errors.New("private and special-use network destinations are not supported")
		}
	}
	return nil
}
func NewProviders(stores ...*Store) *Providers {
	p := &Providers{proxyHosts: map[string]bool{"github.com": true, "api.github.com": true, "raw.githubusercontent.com": true, "api.open-meteo.com": true}}
	for _, host := range strings.Split(os.Getenv("GLANCE_DASHBOARD_FEED_HOSTS"), ",") {
		host = strings.ToLower(strings.TrimSpace(host))
		if host != "" && !strings.ContainsAny(host, "/:@*") && host != "localhost" && !strings.HasSuffix(host, ".local") && !strings.HasSuffix(host, ".internal") {
			p.proxyHosts[host] = true
		}
	}
	if len(stores) > 0 {
		p.store = stores[0]
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	// Preserve the environment's HTTP proxy and CA trust. Direct connections
	// resolve again at dial time so DNS changes cannot reach private networks.
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		// A configured proxy is trusted infrastructure, not a user destination.
		proxy, err := p.proxyFor(&http.Request{URL: &url.URL{Scheme: "https", Host: "example.com"}})
		if err != nil {
			return nil, err
		}
		if proxy != nil && host == proxy.Hostname() {
			return p.dialHost(ctx, network, address)
		}
		ips, err := p.resolveHost(ctx, host)
		if err != nil {
			return nil, err
		}
		for _, ip := range ips {
			if !publicIP(ip) {
				return nil, errors.New("private network address rejected")
			}
		}
		var last error
		for _, ip := range ips {
			conn, err := p.dialHost(ctx, network, net.JoinHostPort(ip.String(), port))
			if err == nil {
				return conn, nil
			}
			last = err
		}
		if last == nil {
			last = errors.New("hostname has no addresses")
		}
		return nil, last
	}
	p.client = &http.Client{Transport: transport, Timeout: 8 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) > 0 && req.URL.Host != via[0].URL.Host {
			req.Header.Del("Authorization")
		}
		if len(via) >= 5 {
			return errors.New("too many redirects")
		}
		return p.checkRequest(req)
	}}
	return p
}
func (p *Providers) get(ctx context.Context, address string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}
	if err = p.checkRequest(req); err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Glance-Dashboard/1.0")
	metadata, _ := ctx.Value(conditionalKey{}).(*conditionalRequest)
	if metadata != nil {
		if metadata.previous.ETag != "" {
			req.Header.Set("If-None-Match", metadata.previous.ETag)
		}
		if metadata.previous.LastModified != "" {
			req.Header.Set("If-Modified-Since", metadata.previous.LastModified)
		}
	}

	res, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if metadata != nil {
		metadata.received = HTTPValidators{ETag: res.Header.Get("ETag"), LastModified: res.Header.Get("Last-Modified")}
		if len(metadata.received.ETag) > 1024 {
			metadata.received.ETag = ""
		}
		if len(metadata.received.LastModified) > 128 {
			metadata.received.LastModified = ""
		}
		if res.StatusCode == http.StatusNotModified {
			if metadata.previous.ETag == "" && metadata.previous.LastModified == "" {
				return nil, &ProviderError{Code: "invalid_response"}
			}
			if metadata.received.ETag == "" {
				metadata.received.ETag = metadata.previous.ETag
			}
			if metadata.received.LastModified == "" {
				metadata.received.LastModified = metadata.previous.LastModified
			}
			return nil, errNotModified
		}
	}
	if res.StatusCode != http.StatusOK {
		failure := statusError(res.StatusCode)
		failure.RetryAfter = parseRetryAfter(res.Header.Get("Retry-After"), time.Now())
		return nil, failure
	}
	const maxBody = MaxProviderBody
	data, err := io.ReadAll(io.LimitReader(res.Body, maxBody+1))
	if len(data) > maxBody {
		return nil, errors.New("provider response exceeds 2 MiB")
	}
	return data, err
}

type FeedItem struct {
	Title       string     `json:"title"`
	URL         string     `json:"url"`
	PublishedAt *time.Time `json:"publishedAt,omitempty"`
}
type FeedData struct {
	Title string     `json:"title"`
	Items []FeedItem `json:"items"`
}
type WeatherData struct {
	Location    string  `json:"location"`
	Temperature float64 `json:"temperature"`
	FeelsLike   float64 `json:"feelsLike"`
	Humidity    int     `json:"humidity"`
	Code        int     `json:"code"`
}

func (p *Providers) Fetch(ctx context.Context, s Source) (any, error) {
	definition, ok := providerDefinitions[s.Type]
	if !ok {
		return nil, errors.New("unsupported provider")
	}
	if err := supportedConfigVersion(s.ConfigVersion, definition.ConfigVersion); err != nil {
		return nil, err
	}
	if err := definition.validate(s.Config); err != nil {
		return nil, err
	}
	return definition.fetch(p, ctx, s)
}
func (p *Providers) fetchRSS(ctx context.Context, s Source) (any, error) {
	var c RSSSource
	if err := decodeConfig(s.Config, &c); err != nil {
		return nil, err
	}
	data, err := p.get(ctx, c.URL)
	if err != nil {
		return nil, err
	}
	if err := validateFeedDocument(data); err != nil {
		return nil, err
	}
	feed, err := gofeed.NewParser().Parse(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	out := FeedData{Title: feed.Title, Items: []FeedItem{}}
	for _, item := range feed.Items {
		if len(out.Items) >= 50 {
			break
		}
		if _, err := publicURL(item.Link); err != nil {
			continue
		}
		out.Items = append(out.Items, FeedItem{Title: item.Title, URL: item.Link, PublishedAt: item.PublishedParsed})
	}
	return out, nil
}
func (p *Providers) fetchWeather(ctx context.Context, s Source) (any, error) {
	var c WeatherSource
	if err := decodeConfig(s.Config, &c); err != nil {
		return nil, err
	}
	query := url.Values{"latitude": {strconv.FormatFloat(c.Latitude, 'f', 6, 64)}, "longitude": {strconv.FormatFloat(c.Longitude, 'f', 6, 64)}, "current": {"temperature_2m,apparent_temperature,relative_humidity_2m,weather_code"}}
	data, err := p.get(ctx, "https://api.open-meteo.com/v1/forecast?"+query.Encode())
	if err != nil {
		return nil, err
	}
	return parseWeather(data, c)
}

func parseWeather(data []byte, c WeatherSource) (WeatherData, error) {
	var body struct {
		Current *struct {
			Temperature *float64 `json:"temperature_2m"`
			FeelsLike   float64  `json:"apparent_temperature"`
			Humidity    int      `json:"relative_humidity_2m"`
			Code        int      `json:"weather_code"`
		} `json:"current"`
	}
	if err := json.Unmarshal(data, &body); err != nil {
		return WeatherData{}, err
	}
	if body.Current == nil || body.Current.Temperature == nil {
		return WeatherData{}, errors.New("missing current weather data")
	}
	return WeatherData{Location: c.Location, Temperature: *body.Current.Temperature, FeelsLike: body.Current.FeelsLike, Humidity: body.Current.Humidity, Code: body.Current.Code}, nil
}

func (p *Providers) resolveHost(ctx context.Context, host string) ([]net.IP, error) {
	if p.resolve != nil {
		return p.resolve(ctx, host)
	}
	return net.DefaultResolver.LookupIP(ctx, "ip", host)
}
func (p *Providers) proxyFor(request *http.Request) (*url.URL, error) {
	if p.proxy != nil {
		return p.proxy(request)
	}
	return http.ProxyFromEnvironment(request)
}
func (p *Providers) dialHost(ctx context.Context, network, address string) (net.Conn, error) {
	if p.dial != nil {
		return p.dial(ctx, network, address)
	}
	return (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, network, address)
}
