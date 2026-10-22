package dashboard

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"time"
)

type DailyBrief struct {
	Date        string  `json:"date"`
	Timezone    string  `json:"timezone"`
	Total       int     `json:"total"`
	Unread      int     `json:"unread"`
	Saved       int     `json:"saved"`
	Items       []Entry `json:"items"`
	Truncated   bool    `json:"truncated"`
	AsOf        string  `json:"asOf,omitempty"`
	Fingerprint string  `json:"fingerprint"`
}

// Find the first instant whose local civil date reaches target. This includes
// 23/25-hour days and midnight transitions without assuming a 24-hour day.
func civilDateBoundary(date time.Time, location *time.Location) time.Time {
	target := date.Format("2006-01-02")
	low, high := date.Add(-36*time.Hour).Unix(), date.Add(36*time.Hour).Unix()
	for low < high {
		middle := low + (high-low)/2
		if time.Unix(middle, 0).In(location).Format("2006-01-02") < target {
			low = middle + 1
		} else {
			high = middle
		}
	}
	return time.Unix(low, 0).In(location)
}
func (s *Store) DailyBrief(date, zone string) (DailyBrief, error) {
	out := DailyBrief{Date: date, Timezone: zone, Items: []Entry{}}
	if zone == "" {
		zone = "UTC"
		out.Timezone = zone
	}
	location, err := time.LoadLocation(zone)
	if err != nil {
		return out, ErrInvalidInput
	}
	if date == "" {
		date = time.Now().In(location).Format("2006-01-02")
		out.Date = date
	}
	civil, err := time.Parse("2006-01-02", date)
	if err != nil || civil.Year() < 1900 || civil.Year() > 2100 {
		return out, ErrInvalidInput
	}
	start, end := civilDateBoundary(civil, location), civilDateBoundary(civil.AddDate(0, 0, 1), location)
	if start.Format("2006-01-02") != date {
		return out, ErrInvalidInput
	}
	rows, err := s.db.Query(`SELECT e.id,e.title,e.url,e.excerpt,e.published_at,e.collected_at,e.source_title,COALESCE(st.is_read,0),COALESCE(st.is_saved,0),COALESCE(st.tags,'[]'),count(*) OVER(),sum(CASE WHEN COALESCE(st.is_read,0)=0 THEN 1 ELSE 0 END) OVER(),sum(CASE WHEN st.is_saved=1 THEN 1 ELSE 0 END) OVER(),max(e.collected_at) OVER() FROM entries e LEFT JOIN entry_state st ON st.entry_id=e.id AND st.owner_id='local' WHERE e.owner_id='local' AND e.sort_at>=? AND e.sort_at<? ORDER BY e.sort_at DESC,e.id DESC LIMIT 20`, start.UnixNano(), end.UnixNano())
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		item := Entry{SourceIDs: []string{}}
		var published sql.NullString
		var tags []byte
		if err = rows.Scan(&item.ID, &item.Title, &item.URL, &item.Excerpt, &published, &item.CollectedAt, &item.SourceTitle, &item.IsRead, &item.IsSaved, &tags, &out.Total, &out.Unread, &out.Saved, &out.AsOf); err != nil {
			return out, err
		}
		item.PublishedAt = published.String
		if err = json.Unmarshal(tags, &item.Tags); err != nil {
			return out, err
		}
		if item.Tags == nil {
			item.Tags = []string{}
		}
		out.Items = append(out.Items, item)
	}
	if err = rows.Err(); err != nil {
		return out, err
	}
	out.Truncated = out.Total > len(out.Items)
	digest := sha256.Sum256(raw(out))
	out.Fingerprint = hex.EncodeToString(digest[:])
	return out, nil
}
