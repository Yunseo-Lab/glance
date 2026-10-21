package dashboard

import "net/http"

type TemplateDefinition struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	Description     string `json:"description"`
	RequiresNetwork bool   `json:"requiresNetwork"`
}

var templateDefinitions = []TemplateDefinition{
	{"daily", "일상으로 시작", "시계·달력·메모·즐겨찾기로 나의 하루를 정리해요.", false},
	{"developer", "개발 공간", "개발 메모와 Go 릴리스 소식을 함께 모아요.", true},
	{"news", "소식 모음", "관심 프로젝트의 업데이트를 놓치지 마세요.", true},
}

func templateBoard(id string) (Board, error) {
	board := defaultBoard()
	board.Sources = []Source{}
	board.Widgets = []Widget{}
	add := func(kind, title string, config any) {
		board.Widgets = append(board.Widgets, Widget{ID: newID(), Type: kind, Title: title, Size: "small", ConfigVersion: 1, Config: raw(config)})
	}
	feed := func(title, address string) {
		source := Source{ID: newID(), Type: "rss", ConfigVersion: 1, Config: raw(RSSSource{URL: address})}
		board.Sources = append(board.Sources, source)
		board.Widgets = append(board.Widgets, Widget{ID: newID(), Type: "rss", Title: title, Size: "wide", SourceID: source.ID, ConfigVersion: 1, Config: raw(RSSConfig{Limit: 5})})
	}
	switch id {
	case "daily":
		add("clock", "지금, 서울", ClockConfig{Timezone: "Asia/Seoul"})
		add("calendar", "이번 달", struct{}{})
		add("note", "오늘의 메모", NoteConfig{Body: "오늘 집중할 한 가지를 적어보세요."})
		add("bookmarks", "자주 가는 곳", BookmarkConfig{Links: []Link{{Title: "GitHub", URL: "https://github.com"}}})
	case "developer":
		add("clock", "지금, UTC", ClockConfig{Timezone: "UTC"})
		add("note", "개발 메모", NoteConfig{Body: "이번에 해결할 문제와 배운 내용을 기록하세요."})
		feed("Go 릴리스", "https://github.com/golang/go/releases.atom")
		add("bookmarks", "개발 도구", BookmarkConfig{Links: []Link{{Title: "Go 문서", URL: "https://go.dev/doc"}, {Title: "GitHub", URL: "https://github.com"}}})
	case "news":
		feed("Glance 업데이트", "https://github.com/glanceapp/glance/releases.atom")
		feed("Go 릴리스", "https://github.com/golang/go/releases.atom")
		add("note", "읽고 남길 생각", NoteConfig{Body: "다시 보고 싶은 소식과 내 생각을 남겨보세요."})
	default:
		return Board{}, ErrNotFound
	}
	return board, board.Validate()
}
func registerTemplateRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/templates", func(w http.ResponseWriter, r *http.Request) { jsonResponse(w, 200, templateDefinitions) })
	mux.HandleFunc("GET /api/v1/templates/{id}", func(w http.ResponseWriter, r *http.Request) {
		board, err := templateBoard(r.PathValue("id"))
		if err != nil {
			serviceError(w, err)
			return
		}
		jsonResponse(w, 200, board)
	})
}
