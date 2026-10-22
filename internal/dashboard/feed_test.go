package dashboard

import (
	"fmt"
	"strings"
	"testing"
)

func TestFeedFormatsStableIdentitySortingDedupAndPlainText(t *testing.T) {
	fixtures := map[string]string{
		"rss":  `<rss version="2.0"><channel><title>News</title><item><guid>old</guid><title>Old</title><link>https://example.com/old?utm_source=feed</link><pubDate>Mon, 20 Oct 2025 12:00:00 GMT</pubDate></item><item><guid>new</guid><title>New</title><link>https://example.com/new#fragment</link><description>&lt;p&gt;안전한 &lt;b&gt;본문&lt;/b&gt;&lt;/p&gt;&lt;script&gt;hidden()&lt;/script&gt;</description><pubDate>Tue, 21 Oct 2025 12:00:00 GMT</pubDate></item><item><guid>dup</guid><title>Duplicate</title><link>https://example.com/old?utm_source=other</link></item><item><title>Unsafe</title><link>javascript:alert(1)</link></item></channel></rss>`,
		"atom": `<feed xmlns="http://www.w3.org/2005/Atom"><title>News</title><entry><id>old</id><title>Old</title><link href="https://example.com/old"/><updated>2025-10-20T12:00:00Z</updated></entry><entry><id>new</id><title>New</title><link href="https://example.com/new"/><updated>2025-10-21T12:00:00Z</updated><summary type="html">&lt;b&gt;본문&lt;/b&gt;</summary></entry></feed>`,
		"json": `{"version":"https://jsonfeed.org/version/1.1","title":"News","items":[{"id":"old","url":"https://example.com/old","title":"Old","date_published":"2025-10-20T12:00:00Z","content_text":"Old"},{"id":"new","url":"https://example.com/new","title":"New","date_published":"2025-10-21T12:00:00Z","content_html":"<p>본문</p>"}]}`,
	}
	for name, fixture := range fixtures {
		t.Run(name, func(t *testing.T) {
			feed, err := parseFeed([]byte(fixture), "https://example.com/feed")
			if err != nil {
				t.Fatal(err)
			}
			if len(feed.Items) != 2 || feed.Items[0].Title != "New" || feed.Items[1].Title != "Old" {
				t.Fatalf("unexpected normalization: %+v", feed)
			}
			again, err := parseFeed([]byte(fixture), "https://example.com/feed")
			if err != nil || feed.Items[0].ID != again.Items[0].ID || len(feed.Items[0].ID) != 64 {
				t.Fatal("unstable identity")
			}
			if strings.ContainsAny(feed.Items[0].Excerpt, "<>") || strings.Contains(feed.Items[0].Excerpt, "hidden()") {
				t.Fatal("unsafe excerpt")
			}
			if feed.Items[0].PublishedAt == nil || feed.Items[0].PublishedAt.Location().String() != "UTC" {
				t.Fatal("date not normalized")
			}
		})
	}
}
func TestFeedBoundariesAndURLResolution(t *testing.T) {
	for _, fixture := range []string{`{broken`, `<rss><broken>`, `<!DOCTYPE rss SYSTEM "https://example.com/entity"><rss/>`, `not a feed`} {
		if _, err := parseFeed([]byte(fixture), "https://example.com/feed"); err == nil {
			t.Fatalf("accepted malformed feed: %s", fixture)
		}
	}
	var fixture strings.Builder
	fixture.WriteString(`<rss version="2.0"><channel><title>Bounded</title>`)
	for i := range 70 {
		fmt.Fprintf(&fixture, `<item><guid>%d</guid><title>Item %d</title><link>/article/%d</link></item>`, i, i, i)
	}
	fixture.WriteString(`</channel></rss>`)
	feed, err := parseFeed([]byte(fixture.String()), "https://example.com/feed")
	if err != nil || len(feed.Items) != 50 || feed.Items[0].URL != "https://example.com/article/0" {
		t.Fatalf("unexpected limit/relative URL: %+v %v", feed, err)
	}
	for _, address := range []string{"https://127.0.0.1/private", "https://user:secret@example.com/x", "javascript:alert(1)"} {
		if _, err := canonicalArticleURL(address, "https://example.com/feed"); err == nil {
			t.Fatalf("accepted %s", address)
		}
	}
	address, err := canonicalArticleURL("https://EXAMPLE.com/a?z=2&utm_source=x&fbclid=y&a=1#part", "https://example.com/feed")
	if err != nil || address != "https://example.com/a?a=1&z=2" {
		t.Fatal(address, err)
	}
}
