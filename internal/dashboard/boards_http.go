package dashboard

import (
	"errors"
	"net/http"
)

func serviceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		apiError(w, 404, err.Error())
	case errors.Is(err, ErrConflict), errors.Is(err, ErrSourceConflict):
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
		}
		if !readMutation(w, r, &input) {
			return
		}
		var template *Board
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
