package dashboard

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"strings"
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
	ConfigVersion int             `json:"configVersion,omitempty"`
	ID            string          `json:"id"`
	Type          string          `json:"type"`
	Title         string          `json:"title"`
	Size          string          `json:"size"`
	SourceID      string          `json:"sourceId,omitempty"`
	Config        json.RawMessage `json:"config"`
}

type Source struct {
	ConnectionID  string          `json:"connectionId,omitempty"`
	ConfigVersion int             `json:"configVersion,omitempty"`
	ID            string          `json:"id"`
	Type          string          `json:"type"`
	Config        json.RawMessage `json:"config"`
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
	Units     string  `json:"units,omitempty"`
	Timezone  string  `json:"timezone,omitempty"`
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
	if err := d.Decode(dst); err != nil {
		return err
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return errors.New("설정은 JSON 하나여야 합니다")
	}
	return nil
}

func publicURL(value string) (*url.URL, error) {
	u, err := url.Parse(value)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || (u.Port() != "" && u.Port() != "443") || len(value) > 2048 {
		return nil, errors.New("인증 정보가 없는 HTTPS URL을 입력하세요 (포트 443)")
	}
	return u, nil
}

func (b Board) Validate() error {
	if !validID(b.ID) || b.Revision < 1 || len(strings.TrimSpace(b.Title)) == 0 || len(b.Title) > 120 {
		return errors.New("대시보드 이름 또는 버전이 올바르지 않습니다")
	}
	if b.Theme != "light" && b.Theme != "dark" {
		return errors.New("지원하지 않는 테마입니다")
	}
	if b.Widgets == nil || b.Sources == nil || len(b.Widgets) > MaxWidgets || len(b.Sources) > MaxBoardSources {
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
		if s.ConnectionID != "" && (!validID(s.ConnectionID) || s.Type != "github" && s.Type != "ical" && s.Type != "json") {
			return errors.New("지원하지 않는 개인 연결입니다")
		}
		if s.Type == "json" {
			var config JSONSource
			if err := decodeConfig(s.Config, &config); err != nil {
				return err
			}
			if (s.ConnectionID == "") == (config.URL == "") {
				return errors.New("공개 JSON 주소 또는 개인 연결 중 하나를 선택하세요")
			}
		}
		if s.Type == "ical" {
			var config ICalSource
			if err := decodeConfig(s.Config, &config); err != nil {
				return err
			}
			if (s.ConnectionID == "") == (config.URL == "") {
				return errors.New("공개 일정 주소 또는 개인 연결 중 하나를 선택하세요")
			}
		}
		sources[s.ID] = s
		definition, ok := providerDefinitions[s.Type]
		if !ok {
			return errors.New("지원하지 않는 데이터 소스입니다")
		}
		if err := supportedConfigVersion(s.ConfigVersion, definition.ConfigVersion); err != nil {
			return err
		}
		if err := definition.validate(s.Config); err != nil {
			return err
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
		definition, ok := widgetDefinitions[w.Type]
		if !ok {
			return errors.New("지원하지 않는 위젯입니다")
		}
		if err := supportedConfigVersion(w.ConfigVersion, definition.ConfigVersion); err != nil {
			return err
		}
		if err := definition.validate(w.Config); err != nil {
			return err
		}
		if definition.SourceType != "" {
			s, ok := sources[w.SourceID]
			if !ok || s.Type != definition.SourceType {
				return errors.New("위젯과 데이터 소스가 일치하지 않습니다")
			}
		} else if w.SourceID != "" {
			return errors.New("이 위젯은 데이터 소스를 사용하지 않습니다")
		}
	}
	return nil
}

// Version zero is the original export/API format. Its meaning is explicitly v1;
// unknown future versions must never be interpreted using today's validator.
func supportedConfigVersion(version, current int) error {
	if version == 0 {
		version = 1
	}
	if version != current {
		return errors.New("지원하지 않는 설정 버전입니다")
	}
	return nil
}
