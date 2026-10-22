package dashboard

import (
	"database/sql"
	"errors"
	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
	"strings"
	"unicode"
	"unicode/utf8"
)

var ErrFTS5Unavailable = errors.New("dashboard requires SQLite FTS5; build with CGO_ENABLED=1 and -tags sqlite_fts5, or use scripts/build-dashboard.sh")

func requireFTS5(db *sql.DB) error {
	var enabled int
	if err := db.QueryRow("SELECT sqlite_compileoption_used('ENABLE_FTS5')").Scan(&enabled); err != nil {
		return err
	}
	if enabled != 1 {
		return ErrFTS5Unavailable
	}
	return nil
}
func ftsExpression(query string) (string, error) {
	query = norm.NFC.String(strings.TrimSpace(query))
	if utf8.RuneCountInString(query) > 200 {
		return "", ErrInvalidInput
	}
	terms := strings.Fields(query)
	if len(terms) == 0 || len(terms) > 8 {
		return "", ErrInvalidInput
	}
	for index, term := range terms {
		if utf8.RuneCountInString(term) < 3 {
			return "", ErrInvalidInput
		}
		terms[index] = `"` + strings.ReplaceAll(term, `"`, `""`) + `"`
	}
	return strings.Join(terms, " AND "), nil
}

func foldSearch(value string) string { return cases.Fold().String(norm.NFC.String(value)) }
func entrySearchClause(query string) (string, []any, error) {
	if !utf8.ValidString(query) || strings.IndexFunc(query, func(r rune) bool { return unicode.IsControl(r) && !unicode.IsSpace(r) }) >= 0 {
		return "", nil, ErrInvalidInput
	}
	query = norm.NFC.String(strings.TrimSpace(query))
	terms := strings.Fields(query)
	if len(terms) == 0 || len(terms) > 8 || utf8.RuneCountInString(query) > 200 {
		return "", nil, ErrInvalidInput
	}
	long := []string{}
	parts := []string{}
	arguments := []any{}
	for _, term := range terms {
		if utf8.RuneCountInString(term) >= 3 {
			long = append(long, term)
		} else {
			parts = append(parts, "instr(e.search_fold,?)>0")
			arguments = append(arguments, foldSearch(term))
		}
	}
	if len(long) > 0 {
		expression, err := ftsExpression(strings.Join(long, " "))
		if err != nil {
			return "", nil, err
		}
		parts = append(parts, "e.rowid IN(SELECT rowid FROM entry_fts WHERE entry_fts MATCH ?)")
		arguments = append(arguments, expression)
	}
	return strings.Join(parts, " AND "), arguments, nil
}
func normalizeEntryTextTx(tx *sql.Tx) error {
	rows, err := tx.Query("SELECT id,title,excerpt,source_title FROM entries")
	if err != nil {
		return err
	}
	type textRow struct{ id, title, excerpt, source string }
	var changed []textRow
	for rows.Next() {
		var row textRow
		if err = rows.Scan(&row.id, &row.title, &row.excerpt, &row.source); err != nil {
			rows.Close()
			return err
		}
		normalized := textRow{row.id, norm.NFC.String(row.title), norm.NFC.String(row.excerpt), norm.NFC.String(row.source)}
		if normalized != row {
			changed = append(changed, normalized)
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, row := range changed {
		if _, err = tx.Exec("UPDATE entries SET title=?,excerpt=?,source_title=? WHERE id=?", row.title, row.excerpt, row.source, row.id); err != nil {
			return err
		}
	}
	return nil
}

func populateSearchFoldTx(tx *sql.Tx) error {
	rows, err := tx.Query("SELECT id,title||' '||excerpt||' '||source_title FROM entries")
	if err != nil {
		return err
	}
	type row struct{ id, text string }
	var values []row
	for rows.Next() {
		var value row
		if err = rows.Scan(&value.id, &value.text); err != nil {
			rows.Close()
			return err
		}
		value.text = foldSearch(value.text)
		values = append(values, value)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, value := range values {
		if _, err = tx.Exec("UPDATE entries SET search_fold=? WHERE id=?", value.text, value.id); err != nil {
			return err
		}
	}
	return nil
}
