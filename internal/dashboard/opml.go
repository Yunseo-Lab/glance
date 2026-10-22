package dashboard

import (
	"bytes"
	"encoding/xml"
	"golang.org/x/net/html/charset"
	"io"
	"net/url"
	"strings"
)

type FeedImportItem struct {
	Title  string `json:"title"`
	URL    string `json:"url"`
	Folder string `json:"folder,omitempty"`
}
type FeedImportPreview struct {
	Feeds          []FeedImportItem `json:"feeds"`
	DuplicateCount int              `json:"duplicateCount"`
}

func parseOPML(document string) (FeedImportPreview, error) {
	out := FeedImportPreview{Feeds: []FeedImportItem{}}
	invalid := func() (FeedImportPreview, error) { return out, ErrInvalidInput }
	if len(document) > 524288 || validateFeedDocument([]byte(document)) != nil {
		return invalid()
	}
	decoder := xml.NewDecoder(bytes.NewBufferString(document))
	decoder.CharsetReader = charset.NewReaderLabel
	seen := map[string]bool{}
	var folders []string
	depth, outlines := 0, 0
	root := false
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return invalid()
		}
		switch value := token.(type) {
		case xml.StartElement:
			depth++
			if depth == 1 && root {
				return invalid()
			}
			if depth > 32 {
				return invalid()
			}
			if !root {
				if !strings.EqualFold(value.Name.Local, "opml") {
					return invalid()
				}
				root = true
			}
			if strings.EqualFold(value.Name.Local, "outline") {
				outlines++
				if outlines > 2000 {
					return invalid()
				}
				title, address := "", ""
				for _, attribute := range value.Attr {
					switch strings.ToLower(attribute.Name.Local) {
					case "text":
						if title == "" {
							title = attribute.Value
						}
					case "title":
						title = attribute.Value
					case "xmlurl":
						address = strings.TrimSpace(attribute.Value)
					}
				}
				title = plainText(title, 100)
				folder := strings.Join(folders, " / ")
				folders = append(folders, title)
				if address == "" {
					continue
				}
				canonical, err := canonicalArticleURL(address, "")
				if err != nil {
					return invalid()
				}
				original, _ := url.Parse(address)
				normalized, _ := url.Parse(canonical)
				normalized.RawQuery = original.Query().Encode()
				address = normalized.String()
				if seen[address] {
					out.DuplicateCount++
					continue
				}
				seen[address] = true
				if len(out.Feeds) >= 128 {
					return invalid()
				}
				if title == "" {
					title = normalized.Hostname()
				}
				out.Feeds = append(out.Feeds, FeedImportItem{Title: title, URL: address, Folder: plainText(folder, 200)})
			}
		case xml.EndElement:
			depth--
			if strings.EqualFold(value.Name.Local, "outline") && len(folders) > 0 {
				folders = folders[:len(folders)-1]
			}
		}
	}
	if !root || depth != 0 {
		return invalid()
	}
	return out, nil
}
