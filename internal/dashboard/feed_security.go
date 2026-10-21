package dashboard

import (
	"bytes"
	"encoding/xml"
	"golang.org/x/net/html/charset"
	"io"
	"strings"
)

// Feed markup is data, not a document with external declarations. Reject DTDs
// and entity definitions before the more tolerant feed parser handles it.
func validateFeedDocument(data []byte) error {
	if !bytes.HasPrefix(bytes.TrimSpace(data), []byte("<")) {
		return nil
	}
	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.CharsetReader = charset.NewReaderLabel
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return &ProviderError{Code: "invalid_response"}
		}
		if directive, ok := token.(xml.Directive); ok {
			value := strings.ToUpper(strings.TrimSpace(string(directive)))
			if strings.HasPrefix(value, "DOCTYPE") || strings.HasPrefix(value, "ENTITY") {
				return &ProviderError{Code: "invalid_response"}
			}
		}
	}
}
