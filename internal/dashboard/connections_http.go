package dashboard

import (
	"net/http"
	"time"
)

func registerConnectionRoutes(mux *http.ServeMux, service *DashboardService) {
	mux.HandleFunc("POST /api/v1/connections", func(w http.ResponseWriter, r *http.Request) {
		var input ConnectionInput
		if !readMutation(w, r, &input) {
			return
		}
		connection, err := service.store.CreateConnection(input)
		if err != nil {
			serviceError(w, err)
			return
		}
		jsonResponse(w, 201, connection)
	})
	mux.HandleFunc("GET /api/v1/connections/{id}", func(w http.ResponseWriter, r *http.Request) {
		connection, err := service.store.connection(r.PathValue("id"))
		if err != nil {
			serviceError(w, err)
			return
		}
		jsonResponse(w, 200, connection)
	})
	mux.HandleFunc("POST /api/v1/connections/{id}/test", func(w http.ResponseWriter, r *http.Request) {
		var input struct{}
		if !readMutation(w, r, &input) {
			return
		}
		report, err := service.store.TestConnection(r.Context(), r.PathValue("id"), service.providers)
		if err != nil {
			serviceError(w, err)
			return
		}
		jsonResponse(w, 200, report)
	})
	mux.HandleFunc("POST /api/v1/connections/{id}/revoke", func(w http.ResponseWriter, r *http.Request) {
		var input struct{}
		if !readMutation(w, r, &input) {
			return
		}
		id := r.PathValue("id")
		if err := service.store.RevokeConnection(id); err != nil {
			serviceError(w, err)
			return
		}
		if service.runtime != nil {
			statuses, _ := service.store.SourceStatuses(time.Now())
			for _, status := range statuses {
				if status.Paused {
					service.runtime.CancelSource(status.Source.ID)
				}
			}
		}
		jsonResponse(w, 200, map[string]bool{"revoked": true})
	})
	mux.HandleFunc("DELETE /api/v1/connections/{id}", func(w http.ResponseWriter, r *http.Request) {
		var input struct{}
		if !readMutation(w, r, &input) {
			return
		}
		if err := service.store.DeleteConnection(r.PathValue("id")); err != nil {
			serviceError(w, err)
			return
		}
		jsonResponse(w, 200, map[string]bool{"deleted": true})
	})
}
