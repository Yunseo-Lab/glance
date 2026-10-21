package dashboard

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"
)

// Definitions are registered at build time; the catalog never executes uploaded code.
type WidgetDefinition struct {
	Type          string         `json:"type"`
	Title         string         `json:"title"`
	Description   string         `json:"description"`
	ConfigVersion int            `json:"configVersion"`
	SourceType    string         `json:"sourceType,omitempty"`
	Capabilities  []string       `json:"capabilities"`
	ConfigSchema  map[string]any `json:"configSchema"`
	validate      func(json.RawMessage) error
}
type ProviderDefinition struct {
	Type            string         `json:"type"`
	Title           string         `json:"title"`
	ConfigVersion   int            `json:"configVersion"`
	FreshForSeconds int            `json:"freshForSeconds"`
	Capabilities    []string       `json:"capabilities"`
	ConfigSchema    map[string]any `json:"configSchema"`
	validate        func(json.RawMessage) error
	fetch           func(*Providers, context.Context, Source) (any, error)
}
type CatalogDocument struct {
	Version   int                  `json:"version"`
	Widgets   []WidgetDefinition   `json:"widgets"`
	Providers []ProviderDefinition `json:"providers"`
}

func objectSchema(properties map[string]any, required ...string) map[string]any {
	if required == nil {
		required = []string{}
	}
	return map[string]any{"type": "object", "additionalProperties": false, "properties": properties, "required": required}
}
func textSchema(max int) map[string]any { return map[string]any{"type": "string", "maxLength": max} }

var widgetDefinitions = map[string]WidgetDefinition{
	"clock":     {Type: "clock", Title: "세계 시계", Description: "원하는 시간대의 현재 시각", ConfigVersion: 1, Capabilities: []string{"local", "time"}, ConfigSchema: objectSchema(map[string]any{"timezone": textSchema(100)}, "timezone"), validate: validateClock},
	"calendar":  {Type: "calendar", Title: "캘린더", Description: "이번 달의 날짜", ConfigVersion: 1, Capabilities: []string{"local", "time"}, ConfigSchema: objectSchema(map[string]any{}), validate: validateEmpty},
	"note":      {Type: "note", Title: "메모", Description: "오늘 기억할 내용을 내 서버에 보관", ConfigVersion: 1, Capabilities: []string{"local", "editable"}, ConfigSchema: objectSchema(map[string]any{"body": textSchema(12000)}, "body"), validate: validateNote},
	"bookmarks": {Type: "bookmarks", Title: "즐겨찾기", Description: "자주 방문하는 HTTPS 링크", ConfigVersion: 1, Capabilities: []string{"local", "editable"}, ConfigSchema: objectSchema(map[string]any{"links": map[string]any{"type": "array", "maxItems": 20, "items": objectSchema(map[string]any{"title": textSchema(100), "url": textSchema(2048)}, "title", "url")}}, "links"), validate: validateBookmarks},
	"rss":       {Type: "rss", Title: "RSS 소식", Description: "RSS·Atom 피드를 한눈에", ConfigVersion: 1, SourceType: "rss", Capabilities: []string{"remote", "reading"}, ConfigSchema: objectSchema(map[string]any{"limit": map[string]any{"type": "integer", "minimum": 1, "maximum": 20}}, "limit"), validate: validateRSSWidget},
	"weather":   {Type: "weather", Title: "날씨", Description: "Open-Meteo의 현재 날씨", ConfigVersion: 1, SourceType: "weather", Capabilities: []string{"remote"}, ConfigSchema: objectSchema(map[string]any{}), validate: validateEmpty},
}
var providerDefinitions = map[string]ProviderDefinition{
	"rss":     {Type: "rss", Title: "공개 RSS·Atom", ConfigVersion: 1, FreshForSeconds: 600, Capabilities: []string{"public-https"}, ConfigSchema: objectSchema(map[string]any{"url": textSchema(2048)}, "url"), validate: validateRSSSource, fetch: (*Providers).fetchRSS},
	"weather": {Type: "weather", Title: "Open-Meteo", ConfigVersion: 1, FreshForSeconds: 900, Capabilities: []string{"public-https"}, ConfigSchema: objectSchema(map[string]any{"latitude": map[string]any{"type": "number", "minimum": -90, "maximum": 90}, "longitude": map[string]any{"type": "number", "minimum": -180, "maximum": 180}, "location": textSchema(100)}, "latitude", "longitude", "location"), validate: validateWeatherSource, fetch: (*Providers).fetchWeather},
}

func Catalog() CatalogDocument {
	out := CatalogDocument{Version: 1, Widgets: []WidgetDefinition{}, Providers: []ProviderDefinition{}}
	for _, d := range widgetDefinitions {
		out.Widgets = append(out.Widgets, d)
	}
	for _, d := range providerDefinitions {
		out.Providers = append(out.Providers, d)
	}
	sort.Slice(out.Widgets, func(i, j int) bool { return out.Widgets[i].Type < out.Widgets[j].Type })
	sort.Slice(out.Providers, func(i, j int) bool { return out.Providers[i].Type < out.Providers[j].Type })
	return out
}
func validateClock(data json.RawMessage) error {
	var c ClockConfig
	if err := decodeConfig(data, &c); err != nil {
		return err
	}
	if c.Timezone == "" || c.Timezone == "Local" {
		return errors.New("명시적인 IANA 시간대를 입력하세요")
	}
	if _, err := time.LoadLocation(c.Timezone); err != nil {
		return errors.New("유효한 시간대를 입력하세요")
	}
	return nil
}
func validateEmpty(data json.RawMessage) error { var c struct{}; return decodeConfig(data, &c) }
func validateNote(data json.RawMessage) error {
	var c NoteConfig
	if err := decodeConfig(data, &c); err != nil {
		return err
	}
	if len(c.Body) > 12000 {
		return errors.New("메모가 너무 깁니다")
	}
	return nil
}
func validateBookmarks(data json.RawMessage) error {
	var c BookmarkConfig
	if err := decodeConfig(data, &c); err != nil {
		return err
	}
	if len(c.Links) > 20 {
		return errors.New("북마크는 최대 20개입니다")
	}
	for _, l := range c.Links {
		if strings.TrimSpace(l.Title) == "" || len(l.Title) > 100 {
			return errors.New("북마크 이름을 확인하세요")
		}
		if _, err := publicURL(l.URL); err != nil {
			return err
		}
	}
	return nil
}
func validateRSSWidget(data json.RawMessage) error {
	var c RSSConfig
	if err := decodeConfig(data, &c); err != nil {
		return err
	}
	if c.Limit < 1 || c.Limit > 20 {
		return errors.New("RSS 표시 개수는 1~20입니다")
	}
	return nil
}
func validateRSSSource(data json.RawMessage) error {
	var c RSSSource
	if err := decodeConfig(data, &c); err != nil {
		return err
	}
	_, err := publicURL(c.URL)
	return err
}
func validateWeatherSource(data json.RawMessage) error {
	var c WeatherSource
	if err := decodeConfig(data, &c); err != nil {
		return err
	}
	if c.Latitude < -90 || c.Latitude > 90 || c.Longitude < -180 || c.Longitude > 180 || strings.TrimSpace(c.Location) == "" || len(c.Location) > 100 {
		return errors.New("날씨 위치와 좌표를 확인하세요")
	}
	return nil
}
