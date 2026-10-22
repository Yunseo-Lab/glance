package dashboard

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type JSONSource struct {
	URL     string `json:"url,omitempty"`
	Pointer string `json:"pointer"`
}
type JSONWidgetConfig struct {
	Unit string `json:"unit,omitempty"`
}
type JSONValueData struct {
	Value string `json:"value"`
	Kind  string `json:"kind"`
}

func pointerSegments(pointer string) ([]string, error) {
	if len(pointer) > 512 {
		return nil, ErrInvalidInput
	}
	if pointer == "" {
		return nil, nil
	}
	if !strings.HasPrefix(pointer, "/") {
		return nil, ErrInvalidInput
	}
	parts := strings.Split(pointer[1:], "/")
	if len(parts) > 16 {
		return nil, ErrInvalidInput
	}
	for index, part := range parts {
		for i := 0; i < len(part); i++ {
			if part[i] == '~' {
				if i+1 >= len(part) || part[i+1] != '0' && part[i+1] != '1' {
					return nil, ErrInvalidInput
				}
				i++
			}
		}
		parts[index] = strings.ReplaceAll(strings.ReplaceAll(part, "~1", "/"), "~0", "~")
	}
	return parts, nil
}
func validateJSONSource(data json.RawMessage) error {
	var config JSONSource
	if err := decodeConfig(data, &config); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(data, &fields) != nil {
		return ErrInvalidInput
	}
	if _, ok := fields["pointer"]; !ok {
		return ErrInvalidInput
	}
	if config.URL != "" {
		if _, err := publicURL(config.URL); err != nil {
			return err
		}
	}
	_, err := pointerSegments(config.Pointer)
	return err
}
func validateJSONWidget(data json.RawMessage) error {
	var config JSONWidgetConfig
	if err := decodeConfig(data, &config); err != nil {
		return err
	}
	if len(config.Unit) > 40 {
		return ErrInvalidInput
	}
	return nil
}
func (p *Providers) fetchJSONValue(ctx context.Context, source Source) (any, error) {
	var config JSONSource
	if err := decodeConfig(source.Config, &config); err != nil {
		return nil, err
	}
	address := config.URL
	headers := http.Header{"Accept": []string{"application/json"}}
	if source.ConnectionID != "" {
		if address != "" || p.store == nil {
			return nil, ErrInvalidInput
		}
		connection, secret, err := p.store.resolveConnection(source.ConnectionID)
		if err != nil || connection.Kind != "json" {
			return nil, &ProviderError{Code: "authentication"}
		}
		address = secret.URL
		ctx = context.WithValue(ctx, sensitiveRequestKey{}, true)
		if secret.Token != "" {
			headers.Set("Authorization", "Bearer "+secret.Token)
		}
	}
	if address == "" {
		return nil, ErrInvalidInput
	}
	data, err := p.getWithHeaders(ctx, address, headers)
	if err != nil {
		return nil, err
	}
	return extractJSONValue(data, config.Pointer)
}
func extractJSONValue(data []byte, pointer string) (JSONValueData, error) {
	invalid := func() (JSONValueData, error) { return JSONValueData{}, &ProviderError{Code: "invalid_response"} }
	if len(data) > MaxProviderBody {
		return invalid()
	}
	parts, err := pointerSegments(pointer)
	if err != nil {
		return invalid()
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if err = decoder.Decode(&value); err != nil {
		return invalid()
	}
	var extra any
	if err = decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return invalid()
	}
	for _, part := range parts {
		switch node := value.(type) {
		case map[string]any:
			var ok bool
			value, ok = node[part]
			if !ok {
				return invalid()
			}
		case []any:
			index, err := strconv.Atoi(part)
			if err != nil || index < 0 || index >= len(node) || strconv.Itoa(index) != part {
				return invalid()
			}
			value = node[index]
		default:
			return invalid()
		}
	}
	result := JSONValueData{}
	switch scalar := value.(type) {
	case nil:
		result.Kind = "null"
		result.Value = "정보 없음"
	case string:
		result.Kind = "string"
		result.Value = scalar
	case json.Number:
		result.Kind = "number"
		result.Value = scalar.String()
	case bool:
		result.Kind = "boolean"
		result.Value = strconv.FormatBool(scalar)
	default:
		return invalid()
	}
	if len(result.Value) > 4096 {
		return invalid()
	}
	return result, nil
}
