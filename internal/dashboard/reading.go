package dashboard

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"
)

type Entry struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	URL         string   `json:"url"`
	Excerpt     string   `json:"excerpt"`
	PublishedAt string   `json:"publishedAt,omitempty"`
	CollectedAt string   `json:"collectedAt"`
	SourceTitle string   `json:"sourceTitle"`
	SourceIDs   []string `json:"sourceIds"`
	IsRead      bool     `json:"isRead"`
	IsSaved     bool     `json:"isSaved"`
	Tags        []string `json:"tags"`
	sortAt      int64
}
type EntryPage struct {
	Items           []Entry `json:"items"`
	NextCursor      string  `json:"nextCursor,omitempty"`
	Total           int     `json:"total"`
	CapacityReached bool    `json:"capacityReached"`
}
type EntryState struct {
	IsRead    bool     `json:"isRead"`
	IsSaved   bool     `json:"isSaved"`
	Tags      []string `json:"tags"`
	UpdatedAt string   `json:"updatedAt"`
}
type EntryStateInput struct {
	Read  *bool     `json:"read,omitempty"`
	Saved *bool     `json:"saved,omitempty"`
	Tags  *[]string `json:"tags,omitempty"`
}
type EntryFilter struct {
	From, Until                                   *time.Time
	Status, BoardID, SourceID, Tag, Query, Cursor string
	Limit                                         int
}
type entryCursor struct {
	At int64  `json:"at"`
	ID string `json:"id"`
}

func (s *Store) Entries(filter EntryFilter) (EntryPage, error) {
	page := EntryPage{Items: []Entry{}}
	if filter.Limit == 0 {
		filter.Limit = 30
	}
	if filter.Limit < 1 || filter.Limit > 50 {
		return page, ErrInvalidInput
	}
	where := "e.owner_id='local'"
	args := []any{}
	switch filter.Status {
	case "", "all":
	case "unread":
		where += " AND COALESCE(st.is_read,0)=0"
	case "read":
		where += " AND st.is_read=1"
	case "saved":
		where += " AND st.is_saved=1"
	default:
		return page, ErrInvalidInput
	}
	if filter.BoardID != "" {
		if !validID(filter.BoardID) {
			return page, ErrInvalidInput
		}
		if _, err := s.GetBoard(filter.BoardID); err != nil {
			return page, err
		}
		where += " AND EXISTS(SELECT 1 FROM entry_sources es JOIN board_sources bs ON bs.source_id=es.source_id WHERE es.entry_id=e.id AND bs.board_id=?)"
		args = append(args, filter.BoardID)
	}
	if filter.SourceID != "" {
		if !validID(filter.SourceID) {
			return page, ErrInvalidInput
		}
		where += " AND EXISTS(SELECT 1 FROM entry_sources es WHERE es.entry_id=e.id AND es.source_id=?)"
		args = append(args, filter.SourceID)
	}
	if filter.Tag != "" {
		tags, err := normalizeEntryTags([]string{filter.Tag})
		if err != nil || len(tags) != 1 {
			return page, ErrInvalidInput
		}
		where += " AND EXISTS(SELECT 1 FROM json_each(COALESCE(st.tags,'[]')) tag WHERE tag.value=?)"
		args = append(args, tags[0])
	}
	if filter.Query != "" {
		clause, arguments, err := entrySearchClause(filter.Query)
		if err != nil {
			return page, err
		}
		where += " AND (" + clause + ")"
		args = append(args, arguments...)
	}

	if filter.From != nil {
		where += " AND e.sort_at>=?"
		args = append(args, filter.From.UnixNano())
	}
	if filter.Until != nil {
		where += " AND e.sort_at<?"
		args = append(args, filter.Until.UnixNano())
	}
	from := " FROM entries e LEFT JOIN entry_state st ON st.entry_id=e.id AND st.owner_id='local' WHERE "
	if err := s.db.QueryRow("SELECT count(*)"+from+where, args...).Scan(&page.Total); err != nil {
		return page, err
	}
	var total int
	if err := s.db.QueryRow("SELECT count(*) FROM entries WHERE owner_id='local'").Scan(&total); err != nil {
		return page, err
	}
	page.CapacityReached = total >= MaxEntries
	if filter.Cursor != "" {
		if len(filter.Cursor) > 256 {
			return page, ErrInvalidInput
		}
		data, err := base64.RawURLEncoding.DecodeString(filter.Cursor)
		var cursor entryCursor
		if err != nil || json.Unmarshal(data, &cursor) != nil || !validID(cursor.ID) {
			return page, ErrInvalidInput
		}
		where += " AND (e.sort_at<? OR (e.sort_at=? AND e.id<?))"
		args = append(args, cursor.At, cursor.At, cursor.ID)
	}
	args = append(args, filter.Limit+1)
	rows, err := s.db.Query("SELECT e.id,e.title,e.url,e.excerpt,e.published_at,e.collected_at,e.source_title,e.sort_at,COALESCE(st.is_read,0),COALESCE(st.is_saved,0),COALESCE(st.tags,'[]')"+from+where+" ORDER BY e.sort_at DESC,e.id DESC LIMIT ?", args...)
	if err != nil {
		return page, err
	}
	for rows.Next() {
		var item Entry
		var published sql.NullString
		var tags []byte
		item.SourceIDs = []string{}
		if err = rows.Scan(&item.ID, &item.Title, &item.URL, &item.Excerpt, &published, &item.CollectedAt, &item.SourceTitle, &item.sortAt, &item.IsRead, &item.IsSaved, &tags); err != nil {
			rows.Close()
			return page, err
		}
		item.PublishedAt = published.String
		if err = json.Unmarshal(tags, &item.Tags); err != nil {
			rows.Close()
			return page, err
		}
		if item.Tags == nil {
			item.Tags = []string{}
		}
		page.Items = append(page.Items, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return page, err
	}
	if len(page.Items) > filter.Limit {
		page.Items = page.Items[:filter.Limit]
		last := page.Items[len(page.Items)-1]
		data, _ := json.Marshal(entryCursor{At: last.sortAt, ID: last.ID})
		page.NextCursor = base64.RawURLEncoding.EncodeToString(data)
	}
	if len(page.Items) > 0 {
		ids := map[string]int{}
		arguments := []any{}
		for index, item := range page.Items {
			ids[item.ID] = index
			arguments = append(arguments, item.ID)
		}
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(arguments)), ",")
		rows, err = s.db.Query("SELECT entry_id,source_id FROM entry_sources WHERE entry_id IN("+placeholders+") ORDER BY source_id", arguments...)
		if err != nil {
			return page, err
		}
		for rows.Next() {
			var entryID, sourceID string
			if err = rows.Scan(&entryID, &sourceID); err != nil {
				rows.Close()
				return page, err
			}
			if index, ok := ids[entryID]; ok {
				page.Items[index].SourceIDs = append(page.Items[index].SourceIDs, sourceID)
			}
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return page, err
		}
	}
	return page, nil
}
func (s *Store) SetEntryState(id string, input EntryStateInput) (EntryState, error) {
	var out EntryState
	if err := validateEntryStateInput(&input); err != nil {
		return out, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return out, err
	}
	defer tx.Rollback()
	out, err = setEntryStateTx(tx, id, input, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return out, err
	}
	if err = tx.Commit(); err != nil {
		return out, err
	}
	return out, nil
}
func setEntryStateTx(tx *sql.Tx, id string, input EntryStateInput, now string) (EntryState, error) {
	var out EntryState
	if !validID(id) {
		return out, ErrInvalidInput
	}
	var exists int
	if err := tx.QueryRow("SELECT count(*) FROM entries WHERE id=? AND owner_id='local'", id).Scan(&exists); err != nil {
		return out, err
	}
	if exists == 0 {
		return out, ErrNotFound
	}
	var tagsArgument any
	if input.Tags != nil {
		tagsArgument = string(raw(*input.Tags))
	}
	_, err := tx.Exec("INSERT INTO entry_state(owner_id,entry_id,is_read,is_saved,tags,updated_at) VALUES('local',?,COALESCE(?,0),COALESCE(?,0),COALESCE(?,'[]'),?) ON CONFLICT(owner_id,entry_id) DO UPDATE SET is_read=COALESCE(?,entry_state.is_read),is_saved=COALESCE(?,entry_state.is_saved),tags=COALESCE(?,entry_state.tags),updated_at=excluded.updated_at", id, input.Read, input.Saved, tagsArgument, now, input.Read, input.Saved, tagsArgument)
	if err != nil {
		return out, err
	}
	var tags []byte
	if err = tx.QueryRow("SELECT is_read,is_saved,tags,updated_at FROM entry_state WHERE owner_id='local' AND entry_id=?", id).Scan(&out.IsRead, &out.IsSaved, &tags, &out.UpdatedAt); err != nil {
		return out, err
	}
	if err = json.Unmarshal(tags, &out.Tags); err != nil {
		return out, err
	}
	return out, nil
}
