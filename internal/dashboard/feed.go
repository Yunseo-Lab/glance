package dashboard

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net"
	"net/url"
	"sort"
	"strings"

	"github.com/mmcdole/gofeed"
	"golang.org/x/net/html"
)

func plainText(markup string, limit int) string {
	tokenizer := html.NewTokenizer(strings.NewReader(markup))
	var output strings.Builder
	hidden := 0
	for {
		kind := tokenizer.Next()
		if kind == html.ErrorToken {
			break
		}
		token := tokenizer.Token()
		switch kind {
		case html.StartTagToken:
			if token.Data == "script" || token.Data == "style" {
				hidden++
			}
			if token.Data == "p" || token.Data == "br" || token.Data == "div" {
				output.WriteByte(' ')
			}
		case html.EndTagToken:
			if (token.Data == "script" || token.Data == "style") && hidden > 0 {
				hidden--
			}
			output.WriteByte(' ')
		case html.TextToken:
			if hidden == 0 {
				output.WriteString(token.Data)
			}
		}
		if output.Len() > limit*8 {
			break
		}
	}
	text := strings.Join(strings.Fields(output.String()), " ")
	runes := []rune(text)
	if len(runes) > limit {
		text = string(runes[:limit]) + "…"
	}
	return text
}
func canonicalArticleURL(address, base string) (string, error) {
	link, err := url.Parse(strings.TrimSpace(address))
	if err != nil {
		return "", err
	}
	if !link.IsAbs() {
		root, err := url.Parse(base)
		if err != nil {
			return "", err
		}
		link = root.ResolveReference(link)
	}
	checked, err := publicURL(link.String())
	if err != nil {
		return "", err
	}
	if ip := net.ParseIP(checked.Hostname()); ip != nil && !publicIP(ip) {
		return "", errors.New("private article destination")
	}
	if host := strings.ToLower(checked.Hostname()); host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") {
		return "", errors.New("private article destination")
	}
	checked.Scheme = strings.ToLower(checked.Scheme)
	checked.Host = strings.ToLower(checked.Host)
	checked.Fragment = ""
	checked.RawFragment = ""
	query := checked.Query()
	for key := range query {
		lower := strings.ToLower(key)
		if strings.HasPrefix(lower, "utm_") || lower == "fbclid" || lower == "gclid" {
			query.Del(key)
		}
	}
	checked.RawQuery = query.Encode()
	if checked.Path == "" {
		checked.Path = "/"
	}
	return checked.String(), nil
}
func parseFeed(data []byte, sourceURL string) (FeedData, error) {
	if err := validateFeedDocument(data); err != nil {
		return FeedData{}, err
	}
	feed, err := gofeed.NewParser().Parse(bytes.NewReader(data))
	if err != nil {
		return FeedData{}, &ProviderError{Code: "invalid_response"}
	}
	out := FeedData{Title: plainText(feed.Title, 240), Items: []FeedItem{}}
	seenURL, seenID := map[string]bool{}, map[string]bool{}
	for _, item := range feed.Items {
		address, err := canonicalArticleURL(item.Link, sourceURL)
		if err != nil || seenURL[address] {
			continue
		}
		identity := strings.TrimSpace(item.GUID)
		if identity == "" {
			identity = address
		}
		sum := sha256.Sum256([]byte(sourceURL + "\x00" + identity))
		id := hex.EncodeToString(sum[:])
		if seenID[id] {
			continue
		}
		seenURL[address] = true
		seenID[id] = true
		published := item.PublishedParsed
		if published == nil {
			published = item.UpdatedParsed
		}
		if published != nil {
			utc := published.UTC()
			published = &utc
		}
		title := plainText(item.Title, 300)
		if title == "" {
			title = address
		}
		excerpt := item.Description
		if excerpt == "" {
			excerpt = item.Content
		}
		out.Items = append(out.Items, FeedItem{ID: id, Title: title, URL: address, PublishedAt: published, Excerpt: plainText(excerpt, 600)})
	}
	sort.SliceStable(out.Items, func(i, j int) bool {
		a, b := out.Items[i].PublishedAt, out.Items[j].PublishedAt
		if a == nil {
			return false
		}
		if b == nil {
			return true
		}
		return a.After(*b)
	})
	if len(out.Items) > 50 {
		out.Items = out.Items[:50]
	}
	return out, nil
}
