package dashboard

import (
	"golang.org/x/text/unicode/norm"
	"sort"
	"strings"
	"time"
	"unicode"
)

const MaxEntryTags = 10
const MaxBulkEntries = 100

type EntryTag struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}
type EntryBulkInput struct {
	IDs   []string  `json:"ids"`
	Read  *bool     `json:"read,omitempty"`
	Saved *bool     `json:"saved,omitempty"`
	Tags  *[]string `json:"tags,omitempty"`
}
type EntryBulkResult struct {
	Updated int `json:"updated"`
}

func normalizeEntryTags(input []string) ([]string, error) {
	if len(input) > MaxEntryTags {
		return nil, ErrInvalidInput
	}
	out := []string{}
	seen := map[string]bool{}
	for _, tag := range input {
		tag = norm.NFC.String(strings.TrimSpace(tag))
		if tag == "" || len(tag) > 40 || strings.ContainsRune(tag, ',') || strings.IndexFunc(tag, unicode.IsControl) >= 0 {
			return nil, ErrInvalidInput
		}
		key := strings.ToLower(tag)
		if !seen[key] {
			seen[key] = true
			out = append(out, tag)
		}
	}
	sort.Strings(out)
	return out, nil
}
func validateEntryStateInput(input *EntryStateInput) error {
	if input.Read == nil && input.Saved == nil && input.Tags == nil {
		return ErrInvalidInput
	}
	if input.Tags != nil {
		tags, err := normalizeEntryTags(*input.Tags)
		if err != nil {
			return err
		}
		input.Tags = &tags
	}
	return nil
}
func (s *Store) BulkEntryState(input EntryBulkInput) (EntryBulkResult, error) {
	out := EntryBulkResult{}
	if len(input.IDs) == 0 || len(input.IDs) > MaxBulkEntries {
		return out, ErrInvalidInput
	}
	changes := EntryStateInput{Read: input.Read, Saved: input.Saved, Tags: input.Tags}
	if err := validateEntryStateInput(&changes); err != nil {
		return out, err
	}
	ids := []string{}
	seen := map[string]bool{}
	for _, id := range input.IDs {
		if !validID(id) {
			return out, ErrInvalidInput
		}
		if !seen[id] {
			ids = append(ids, id)
			seen[id] = true
		}
	}
	tx, err := s.db.Begin()
	if err != nil {
		return out, err
	}
	defer tx.Rollback()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, id := range ids {
		if _, err = setEntryStateTx(tx, id, changes, now); err != nil {
			return out, err
		}
	}
	if err = tx.Commit(); err != nil {
		return out, err
	}
	out.Updated = len(ids)
	return out, nil
}
func (s *Store) EntryTags() ([]EntryTag, error) {
	rows, err := s.db.Query("SELECT tag.value,count(DISTINCT e.id) FROM entries e JOIN entry_state st ON st.entry_id=e.id AND st.owner_id='local' JOIN json_each(st.tags) tag WHERE e.owner_id='local' GROUP BY tag.value ORDER BY count(DISTINCT e.id) DESC,tag.value LIMIT 100")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []EntryTag{}
	for rows.Next() {
		var tag EntryTag
		if err = rows.Scan(&tag.Tag, &tag.Count); err != nil {
			return nil, err
		}
		out = append(out, tag)
	}
	return out, rows.Err()
}
