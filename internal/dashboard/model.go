package dashboard

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Board is a versioned aggregate: layout and source edits commit together.
type Board struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Revision int64    `json:"revision"`
	Theme    string   `json:"theme"`
	Widgets  []Widget `json:"widgets"`
	Sources  []Source `json:"sources"`
}

type Widget struct {
	ID       string          `json:"id"`
	Type     string          `json:"type"`
	Title    string          `json:"title"`
	Size     string          `json:"size"`
	SourceID string          `json:"sourceId,omitempty"`
	Config   json.RawMessage `json:"config"`
}

type Source struct {
	ID     string          `json:"id"`
	Type   string          `json:"type"`
	Config json.RawMessage `json:"config"`
}

type ClockConfig struct {
	Timezone string `json:"timezone"`
}
type NoteConfig struct {
	Body string `json:"body"`
}
type Link struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}
type BookmarkConfig struct {
	Links []Link `json:"links"`
}
type RSSConfig struct {
	Limit int `json:"limit"`
}
type RSSSource struct {
	URL string `json:"url"`
}
type WeatherSource struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Location  string  `json:"location"`
}

func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}

func raw(v any) json.RawMessage { b, _ := json.Marshal(v); return b }

func defaultBoard() Board {
	source := Source{ID: newID(), Type: "rss", Config: raw(RSSSource{URL: "https://github.com/glanceapp/glance/releases.atom"})}
	return Board{ID: "home", Title: "나의 대시보드", Revision: 1, Theme: "light", Sources: []Source{source}, Widgets: []Widget{
		{ID: newID(), Type: "clock", Title: "지금, 서울", Size: "small", Config: raw(ClockConfig{Timezone: "Asia/Seoul"})},
		{ID: newID(), Type: "rss", Title: "Glance 업데이트", Size: "wide", SourceID: source.ID, Config: raw(RSSConfig{Limit: 5})},
		{ID: newID(), Type: "calendar", Title: "이번 달", Size: "small", Config: raw(struct{}{})},
		{ID: newID(), Type: "bookmarks", Title: "자주 가는 곳", Size: "small", Config: raw(BookmarkConfig{Links: []Link{{Title: "GitHub", URL: "https://github.com"}, {Title: "Glance 문서", URL: "https://github.com/glanceapp/glance/blob/main/docs/configuration.md"}}})},
		{ID: newID(), Type: "note", Title: "오늘의 메모", Size: "small", Config: raw(NoteConfig{Body: "나에게 필요한 정보만, 한곳에.\n\n편집을 눌러 위젯을 바꾸고 나만의 대시보드를 만들어 보세요."})},
	}}
}

func validID(id string) bool {
	if len(id) < 1 || len(id) > 64 {
		return false
	}
	for _, c := range id {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

func decodeConfig(data json.RawMessage, dst any) error {
	if len(data) == 0 || string(data) == "null" {
		return errors.New("설정이 필요합니다")
	}
	d := json.NewDecoder(strings.NewReader(string(data)))
	d.DisallowUnknownFields()
	return d.Decode(dst)
}

func publicURL(value string) (*url.URL, error) {
	u, err := url.Parse(value)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || (u.Port() != "" && u.Port() != "443") || len(value) > 2048 {
		return nil, errors.New("인증 정보가 없는 HTTPS URL을 입력하세요 (포트 443)")
	}
	return u, nil
}

func (b Board) Validate() error {
	if b.ID != "home" || b.Revision < 1 || len(strings.TrimSpace(b.Title)) == 0 || len(b.Title) > 120 {
		return errors.New("대시보드 이름 또는 버전이 올바르지 않습니다")
	}
	if b.Theme != "light" && b.Theme != "dark" {
		return errors.New("지원하지 않는 테마입니다")
	}
	if b.Widgets == nil || b.Sources == nil || len(b.Widgets) > 32 || len(b.Sources) > 32 {
		return errors.New("위젯과 데이터 소스는 최대 32개입니다")
	}
	sources := map[string]Source{}
	for _, s := range b.Sources {
		if !validID(s.ID) {
			return errors.New("잘못된 데이터 소스 ID입니다")
		}
		if _, ok := sources[s.ID]; ok {
			return errors.New("데이터 소스 ID가 중복됩니다")
		}
		sources[s.ID] = s
		switch s.Type {
		case "rss":
			var c RSSSource
			if err := decodeConfig(s.Config, &c); err != nil {
				return err
			}
			if _, err := publicURL(c.URL); err != nil {
				return err
			}
		case "weather":
			var c WeatherSource
			if err := decodeConfig(s.Config, &c); err != nil {
				return err
			}
			if c.Latitude < -90 || c.Latitude > 90 || c.Longitude < -180 || c.Longitude > 180 || strings.TrimSpace(c.Location) == "" || len(c.Location) > 100 {
				return errors.New("날씨 위치와 좌표를 확인하세요")
			}
		default:
			return errors.New("지원하지 않는 데이터 소스입니다")
		}
	}
	ids := map[string]bool{}
	for _, w := range b.Widgets {
		if !validID(w.ID) || ids[w.ID] {
			return errors.New("위젯 ID가 잘못되었거나 중복됩니다")
		}
		ids[w.ID] = true
		if strings.TrimSpace(w.Title) == "" || len(w.Title) > 100 {
			return errors.New("위젯 제목은 1~100바이트로 입력하세요")
		}
		if w.Size != "small" && w.Size != "wide" {
			return errors.New("잘못된 위젯 크기입니다")
		}
		switch w.Type {
		case "clock":
			var c ClockConfig
			if err := decodeConfig(w.Config, &c); err != nil {
				return err
			}
			if c.Timezone == "" || c.Timezone == "Local" {
				return errors.New("명시적인 IANA 시간대를 입력하세요")
			}
			if _, err := time.LoadLocation(c.Timezone); err != nil {
				return errors.New("유효한 시간대를 입력하세요")
			}
		case "note":
			var c NoteConfig
			if err := decodeConfig(w.Config, &c); err != nil {
				return err
			}
			if len(c.Body) > 12000 {
				return errors.New("메모가 너무 깁니다")
			}
		case "bookmarks":
			var c BookmarkConfig
			if err := decodeConfig(w.Config, &c); err != nil {
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
		case "rss":
			var c RSSConfig
			if err := decodeConfig(w.Config, &c); err != nil {
				return err
			}
			if c.Limit < 1 || c.Limit > 20 {
				return errors.New("RSS 표시 개수는 1~20입니다")
			}
		case "calendar", "weather":
			var c struct{}
			if err := decodeConfig(w.Config, &c); err != nil {
				return err
			}
		default:
			return fmt.Errorf("지원하지 않는 위젯: %s", w.Type)
		}
		if w.Type == "rss" || w.Type == "weather" {
			s, ok := sources[w.SourceID]
			if !ok || s.Type != w.Type {
				return errors.New("위젯과 데이터 소스가 일치하지 않습니다")
			}
		} else if w.SourceID != "" {
			return errors.New("이 위젯은 데이터 소스를 사용하지 않습니다")
		}
	}
	return nil
}
