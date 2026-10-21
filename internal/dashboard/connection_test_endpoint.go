package dashboard

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

type ConnectionTestReport struct {
	Connected    bool     `json:"connected"`
	Identity     string   `json:"identity,omitempty"`
	Capabilities []string `json:"capabilities"`
	ErrorCode    string   `json:"errorCode,omitempty"`
}

func (s *Store) TestConnection(ctx context.Context, id string, p *Providers) (ConnectionTestReport, error) {
	connection, secret, err := s.resolveConnection(id)
	if err != nil {
		return ConnectionTestReport{}, err
	}
	now := time.Now().UTC()
	if previous, err := time.Parse(time.RFC3339Nano, connection.LastTestedAt); err == nil && previous.Add(30*time.Second).After(now) {
		return ConnectionTestReport{}, &CooldownError{Until: previous.Add(30 * time.Second)}
	}
	result, err := s.db.Exec("UPDATE connections SET last_tested_at=?,status='testing' WHERE id=? AND owner_id='local' AND generation=? AND last_tested_at=?", now.Format(time.RFC3339Nano), id, connection.Generation, connection.LastTestedAt)
	if err != nil {
		return ConnectionTestReport{}, err
	}
	if count, _ := result.RowsAffected(); count != 1 {
		return ConnectionTestReport{}, ErrConflict
	}
	report := ConnectionTestReport{Capabilities: []string{}}
	address := secret.URL
	if connection.Kind == "github" {
		address = "https://api.github.com/user"
	}
	request, err := http.NewRequestWithContext(ctx, "GET", address, nil)
	if err == nil {
		err = p.checkRequest(request)
	}
	if err == nil {
		if secret.Token != "" {
			request.Header.Set("Authorization", "Bearer "+secret.Token)
		}
		request.Header.Set("User-Agent", "Glance-Dashboard/1.0")
		request.Header.Set("Accept", "application/json, text/calendar;q=0.9")
		var response *http.Response
		response, err = p.client.Do(request)
		if err == nil {
			defer response.Body.Close()
			if response.StatusCode != 200 {
				err = statusError(response.StatusCode)
			} else {
				body, readErr := io.ReadAll(io.LimitReader(response.Body, 65537))
				err = readErr
				if len(body) > 65536 {
					err = &ProviderError{Code: "invalid_response"}
				}
				if err == nil {
					switch connection.Kind {
					case "github":
						var identity struct {
							ID    int64  `json:"id"`
							Login string `json:"login"`
						}
						if json.Unmarshal(body, &identity) != nil || identity.ID <= 0 || identity.Login == "" {
							err = &ProviderError{Code: "invalid_response"}
						} else {
							report.Identity = identity.Login
							report.Capabilities = []string{"profile:read"}
						}
					case "calendar":
						if !strings.HasPrefix(strings.TrimSpace(string(body)), "BEGIN:VCALENDAR") || !strings.Contains(string(body), "END:VCALENDAR") {
							err = &ProviderError{Code: "invalid_response"}
						} else {
							report.Capabilities = []string{"calendar:read"}
						}
					case "json":
						if !json.Valid(body) {
							err = &ProviderError{Code: "invalid_response"}
						} else {
							report.Capabilities = []string{"endpoint:read"}
						}
					}
				}
			}
		}
	}
	status := "connected"
	if err != nil {
		status = classifyProviderError(err)
		report.ErrorCode = status
	} else {
		report.Connected = true
	}
	result, dbErr := s.db.Exec("UPDATE connections SET status=? WHERE id=? AND owner_id='local' AND generation=? AND status<>'revoked'", status, id, connection.Generation)
	if dbErr != nil {
		return report, dbErr
	}
	if count, _ := result.RowsAffected(); count != 1 {
		return ConnectionTestReport{}, ErrConflict
	}
	return report, nil
}
