package dashboard

import (
	"errors"
	"math"
	"net/http"
	"strconv"
	"time"
)

func serviceError(w http.ResponseWriter, err error) {
	var cooldown *CooldownError
	switch {
	case errors.As(err, &cooldown):
		seconds := int(math.Ceil(time.Until(cooldown.Until).Seconds()))
		if seconds < 1 {
			seconds = 1
		}
		w.Header().Set("Retry-After", strconv.Itoa(seconds))
		apiError(w, 429, cooldown.Error())
	case errors.Is(err, ErrConnectionLocked):
		apiError(w, 503, ErrConnectionLocked.Error())
	case errors.Is(err, ErrNotFound):
		apiError(w, 404, err.Error())
	case errors.Is(err, ErrConflict), errors.Is(err, ErrSourceConflict), errors.Is(err, ErrInUse):
		apiError(w, 409, err.Error())
	case errors.Is(err, ErrLimit), errors.Is(err, ErrSourceLimit):
		apiError(w, 422, err.Error())
	case errors.Is(err, ErrProtectedBoard), errors.Is(err, ErrInvalidInput):
		apiError(w, 400, err.Error())
	default:
		apiError(w, 500, "요청을 처리하지 못했습니다")
	}
}
func registerBoardRoutes(mux *http.ServeMux, service *DashboardService) {
	mux.HandleFunc("GET /api/v1/locations", func(w http.ResponseWriter, r *http.Request) {
		result, err := service.SearchLocations(r.Context(), r.URL.Query().Get("q"))
		if err != nil {
			var upstream *ProviderError
			if errors.As(err, &upstream) {
				apiError(w, 502, "지역을 검색하지 못했습니다. 잠시 뒤 다시 시도하세요.")
			} else {
				serviceError(w, err)
			}
			return
		}
		jsonResponse(w, 200, result)
	})

	mux.HandleFunc("DELETE /api/v1/sources/{id}", func(w http.ResponseWriter, r *http.Request) {
		var input struct{}
		if !readMutation(w, r, &input) {
			return
		}
		id := r.PathValue("id")
		if err := service.store.DeleteSource(id); err != nil {
			serviceError(w, err)
			return
		}
		if service.runtime != nil {
			service.runtime.CancelSource(id)
		}
		_ = service.store.pruneSnapshots(nil)
		jsonResponse(w, 200, map[string]bool{"deleted": true})
	})

	mux.HandleFunc("GET /api/v1/connections", func(w http.ResponseWriter, r *http.Request) {
		connections, err := service.store.Connections()
		if err != nil {
			serviceError(w, err)
			return
		}
		jsonResponse(w, 200, connections)
	})

	mux.HandleFunc("GET /api/v1/sources", func(w http.ResponseWriter, r *http.Request) {
		statuses, err := service.store.SourceStatuses(time.Now())
		if err != nil {
			serviceError(w, err)
			return
		}
		jsonResponse(w, 200, statuses)
	})
	mux.HandleFunc("GET /api/v1/diagnostics", func(w http.ResponseWriter, r *http.Request) {
		statuses, err := service.store.SourceStatuses(time.Now())
		if err != nil {
			serviceError(w, err)
			return
		}
		running, failed, paused := 0, 0, 0
		for _, status := range statuses {
			if status.Running {
				running++
			}
			if status.Snapshot.ErrorCode != "" {
				failed++
			}
			if status.Paused {
				paused++
			}
		}
		jsonResponse(w, 200, map[string]any{"activeSources": len(statuses), "runningSources": running, "failedSources": failed, "pausedSources": paused, "limits": Limits()})
	})
	mux.HandleFunc("POST /api/v1/sources/{id}/pause", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Paused *bool `json:"paused"`
		}
		if !readMutation(w, r, &input) {
			return
		}
		if input.Paused == nil {
			apiError(w, 400, "일시정지 여부를 입력하세요")
			return
		}
		id := r.PathValue("id")
		if err := service.store.PauseSource(id, *input.Paused); err != nil {
			serviceError(w, err)
			return
		}
		if service.runtime != nil {
			service.runtime.CancelSource(id)
		}
		jsonResponse(w, 200, map[string]bool{"paused": *input.Paused})
	})

	mux.HandleFunc("POST /api/v1/sources/{id}/refresh", func(w http.ResponseWriter, r *http.Request) {
		var input struct{}
		if !readMutation(w, r, &input) {
			return
		}
		source, receipt, err := service.store.RequestRefresh(r.PathValue("id"), time.Now())
		if err != nil {
			serviceError(w, err)
			return
		}
		if service.runtime != nil {
			_, _ = service.runtime.Read(source)
		}
		jsonResponse(w, 202, receipt)
	})

	mux.HandleFunc("GET /api/v1/boards", func(w http.ResponseWriter, r *http.Request) {
		boards, err := service.store.Boards()
		if err != nil {
			serviceError(w, err)
			return
		}
		jsonResponse(w, 200, boards)
	})
	mux.HandleFunc("POST /api/v1/boards", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Title     string `json:"title"`
			CloneFrom string `json:"cloneFrom,omitempty"`
			Template  string `json:"template,omitempty"`
		}
		if !readMutation(w, r, &input) {
			return
		}
		var template *Board
		if input.Template != "" {
			if input.CloneFrom != "" {
				serviceError(w, ErrInvalidInput)
				return
			}
			board, err := templateBoard(input.Template)
			if err != nil {
				serviceError(w, err)
				return
			}
			template = &board
		}
		if input.CloneFrom != "" {
			board, err := service.Board(input.CloneFrom)
			if err != nil {
				serviceError(w, err)
				return
			}
			template = &board
		}
		board, err := service.store.CreateBoard(input.Title, template)
		if err != nil {
			serviceError(w, err)
			return
		}
		jsonResponse(w, 201, board)
	})
	mux.HandleFunc("GET /api/v1/boards/{id}", func(w http.ResponseWriter, r *http.Request) {
		board, err := service.Board(r.PathValue("id"))
		if err != nil {
			serviceError(w, err)
			return
		}
		jsonResponse(w, 200, board)
	})
	mux.HandleFunc("PUT /api/v1/boards/{id}", func(w http.ResponseWriter, r *http.Request) {
		var board Board
		if !readMutation(w, r, &board) {
			return
		}
		if board.ID != r.PathValue("id") {
			apiError(w, 400, "보드 ID가 일치하지 않습니다")
			return
		}
		if err := board.Validate(); err != nil {
			apiError(w, 400, err.Error())
			return
		}
		saved, err := service.Commit(board)
		if err != nil {
			serviceError(w, err)
			return
		}
		jsonResponse(w, 200, saved)
	})
	mux.HandleFunc("DELETE /api/v1/boards/{id}", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Revision int64 `json:"revision"`
		}
		if !readMutation(w, r, &input) {
			return
		}
		if input.Revision < 1 {
			apiError(w, 400, "현재 revision이 필요합니다")
			return
		}
		if err := service.store.DeleteBoard(r.PathValue("id"), input.Revision); err != nil {
			serviceError(w, err)
			return
		}
		if service.runtime != nil {
			service.runtime.CancelUnreferenced()
		}
		_ = service.store.pruneSnapshots(nil)
		jsonResponse(w, 200, map[string]bool{"deleted": true})
	})
	mux.HandleFunc("POST /api/v1/boards/{id}/validate", func(w http.ResponseWriter, r *http.Request) {
		var board Board
		if !readMutation(w, r, &board) {
			return
		}
		if board.ID != r.PathValue("id") {
			apiError(w, 400, "보드 ID가 일치하지 않습니다")
			return
		}
		if err := board.Validate(); err != nil {
			apiError(w, 400, err.Error())
			return
		}
		jsonResponse(w, 200, board)
	})
	mux.HandleFunc("GET /api/v1/boards/{id}/snapshots", func(w http.ResponseWriter, r *http.Request) {
		out, err := service.Snapshots(r.PathValue("id"))
		if err != nil {
			serviceError(w, err)
			return
		}
		jsonResponse(w, 200, out)
	})
}
