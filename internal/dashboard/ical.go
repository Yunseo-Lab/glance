package dashboard

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	ics "github.com/arran4/golang-ical"
	"github.com/teambition/rrule-go"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type ICalSource struct {
	URL      string `json:"url,omitempty"`
	Timezone string `json:"timezone,omitempty"`
}
type CalendarEvent struct {
	ID       string    `json:"id"`
	Title    string    `json:"title"`
	Start    time.Time `json:"start"`
	End      time.Time `json:"end"`
	AllDay   bool      `json:"allDay"`
	Timezone string    `json:"timezone"`
	Location string    `json:"location,omitempty"`
	URL      string    `json:"url,omitempty"`
}
type CalendarData struct {
	Title    string          `json:"title"`
	Events   []CalendarEvent `json:"events"`
	Warnings []string        `json:"warnings"`
}

func validateICalSource(data json.RawMessage) error {
	var config ICalSource
	if err := decodeConfig(data, &config); err != nil {
		return err
	}
	if config.URL != "" {
		if _, err := publicURL(config.URL); err != nil {
			return err
		}
	}
	if config.Timezone != "" {
		if _, err := time.LoadLocation(config.Timezone); err != nil {
			return ErrInvalidInput
		}
	}
	return nil
}
func (p *Providers) fetchICal(ctx context.Context, source Source) (any, error) {
	var config ICalSource
	if err := decodeConfig(source.Config, &config); err != nil {
		return nil, err
	}
	headers := http.Header{"Accept": []string{"text/calendar"}}
	address := config.URL
	if source.ConnectionID != "" {
		if address != "" || p.store == nil {
			return nil, ErrInvalidInput
		}
		connection, secret, err := p.store.resolveConnection(source.ConnectionID)
		if err != nil || connection.Kind != "calendar" {
			return nil, &ProviderError{Code: "authentication"}
		}
		address = secret.URL
		if secret.Token != "" {
			headers.Set("Authorization", "Bearer "+secret.Token)
		}
		ctx = context.WithValue(ctx, sensitiveRequestKey{}, true)
	}
	if address == "" {
		return nil, ErrInvalidInput
	}
	data, err := p.getWithHeaders(ctx, address, headers)
	if err != nil {
		return nil, err
	}
	return parseICalendar(ctx, data, config.Timezone, time.Now())
}
func calendarProperty(event *ics.VEvent, name string) *ics.IANAProperty {
	return event.GetProperty(ics.ComponentProperty(name))
}
func calendarText(event *ics.VEvent, name string) string {
	if prop := calendarProperty(event, name); prop != nil {
		return prop.Value
	}
	return ""
}
func calendarTime(prop *ics.IANAProperty, fallback *time.Location) (time.Time, bool, error) {
	if prop == nil {
		return time.Time{}, false, errors.New("missing date")
	}
	location := fallback
	if zone := prop.ICalParameters["TZID"]; len(zone) > 0 {
		var err error
		location, err = time.LoadLocation(strings.Trim(zone[0], `"`))
		if err != nil {
			return time.Time{}, false, err
		}
	}
	value := prop.Value
	allDay := len(value) == 8
	layout := "20060102T150405"
	if allDay {
		layout = "20060102"
	}
	if strings.HasSuffix(value, "Z") {
		layout = "20060102T150405Z"
		location = time.UTC
	}
	parsed, err := time.ParseInLocation(layout, value, location)
	if err == nil && parsed.Format(layout) != value {
		err = errors.New("nonexistent local date")
	}
	return parsed, allDay, err
}
func calendarDateList(event *ics.VEvent, name string, location *time.Location) []time.Time {
	var result []time.Time
	for _, prop := range event.GetProperties(ics.ComponentProperty(name)) {
		for _, value := range strings.Split(prop.Value, ",") {
			copy := *prop
			copy.Value = value
			if parsed, _, err := calendarTime(&copy, location); err == nil {
				result = append(result, parsed)
			}
		}
	}
	return result
}
func parseICalendar(ctx context.Context, data []byte, zone string, now time.Time) (CalendarData, error) {
	out := CalendarData{Title: "일정", Events: []CalendarEvent{}, Warnings: []string{}}
	invalid := func() (CalendarData, error) { return out, &ProviderError{Code: "invalid_response"} }
	if len(data) > MaxProviderBody || !bytes.Contains(bytes.ToUpper(data), []byte("BEGIN:VCALENDAR")) {
		return invalid()
	}
	depth := 0
	for _, line := range strings.Split(string(data), "\n") {
		upper := strings.ToUpper(strings.TrimSpace(line))
		if strings.HasPrefix(upper, "BEGIN:") {
			depth++
			if depth > 16 {
				return invalid()
			}
		}
		if strings.HasPrefix(upper, "END:") {
			depth--
			if depth < 0 {
				return invalid()
			}
		}
	}
	if depth != 0 {
		return invalid()
	}
	calendar, err := ics.ParseCalendar(bytes.NewReader(data))
	if err != nil {
		return invalid()
	}
	events := calendar.Events()
	if len(events) > 1000 {
		return invalid()
	}
	location := time.UTC
	if zone != "" {
		location, err = time.LoadLocation(zone)
		if err != nil {
			return invalid()
		}
	}
	warn := func() {
		if len(out.Warnings) == 0 {
			out.Warnings = append(out.Warnings, "일부 일정의 날짜·반복 규칙을 지원하지 않거나 표시 한도를 넘었습니다.")
		}
	}
	lower := now.AddDate(0, 0, -1)
	upper := now.AddDate(0, 0, 31)
	overrides := map[string]bool{}
	key := func(uid string, at time.Time) string { return uid + "@" + at.UTC().Format(time.RFC3339) }
	for _, event := range events {
		if prop := calendarProperty(event, "RECURRENCE-ID"); prop != nil {
			at, _, err := calendarTime(prop, location)
			if err != nil {
				warn()
				continue
			}
			if len(prop.ICalParameters["RANGE"]) > 0 {
				warn()
			}
			overrides[key(calendarText(event, "UID"), at)] = true
		}
	}
	seen := map[string]bool{}
	work := 0
	for _, event := range events {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		if len(out.Events) > 2000 {
			warn()
			break
		}
		uid := calendarText(event, "UID")
		if uid == "" || strings.EqualFold(calendarText(event, "STATUS"), "CANCELLED") {
			continue
		}
		start, allDay, err := calendarTime(calendarProperty(event, "DTSTART"), location)
		if err != nil {
			warn()
			continue
		}
		end, _, endErr := calendarTime(calendarProperty(event, "DTEND"), start.Location())
		if endErr != nil {
			if value := calendarText(event, "DURATION"); value != "" {
				days, clock, err := calendarDuration(value)
				if err != nil {
					warn()
					continue
				}
				end = start.AddDate(0, 0, days).Add(clock)
			} else if allDay {
				end = start.AddDate(0, 0, 1)
			} else {
				end = start
			}
		}
		if end.Before(start) || end.Sub(start) > 366*24*time.Hour {
			warn()
			continue
		}
		duration := end.Sub(start)
		days := int(time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, time.UTC).Sub(time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)).Hours() / 24)
		recurrenceID, _, _ := calendarTime(calendarProperty(event, "RECURRENCE-ID"), location)
		occurrences := []time.Time{start}
		if rule := calendarProperty(event, "RRULE"); rule != nil && !recurrenceID.IsZero() {
			warn()
			continue
		} else if rule != nil {
			options, err := rrule.StrToROptionInLocation(rule.Value, start.Location())
			if err != nil || options.Freq > rrule.DAILY || options.Interval > 1000 || len(options.Byhour) > 1 || len(options.Byminute) > 1 || len(options.Bysecond) > 1 || start.Year() < now.Year()-120 {
				warn()
				continue
			}
			options.Dtstart = start
			if options.Until.IsZero() || options.Until.After(upper) {
				options.Until = upper
			}
			if cost := int(now.Sub(start).Hours() / 24); cost > 0 {
				work += cost
			}
			if work > 500000 {
				warn()
				continue
			}
			rule, err := rrule.NewRRule(*options)
			if err != nil {
				warn()
				continue
			}
			occurrences = nil
			next := rule.Iterator()
			for count := 0; count < 50000; count++ {
				at, ok := next()
				if !ok || at.After(upper) {
					break
				}
				if at.Add(duration).Before(lower) {
					continue
				}
				occurrences = append(occurrences, at)
				if len(occurrences) > 500 {
					warn()
					break
				}
			}
		}
		occurrences = append(occurrences, calendarDateList(event, "RDATE", start.Location())...)
		excluded := map[int64]bool{}
		for _, at := range calendarDateList(event, "EXDATE", start.Location()) {
			excluded[at.Unix()] = true
		}
		for _, at := range occurrences {
			finish := at.Add(duration)
			if allDay {
				finish = at.AddDate(0, 0, days)
			}
			if at.After(upper) || finish.Before(lower) || excluded[at.Unix()] {
				continue
			}
			identityAt := at
			if !recurrenceID.IsZero() {
				identityAt = recurrenceID
			} else if overrides[key(uid, at)] {
				continue
			}
			identity := key(uid, identityAt)
			if seen[identity] {
				continue
			}
			seen[identity] = true
			digest := sha256.Sum256([]byte(identity))
			entry := CalendarEvent{ID: hex.EncodeToString(digest[:]), Title: plainText(calendarText(event, "SUMMARY"), 300), Start: at, End: finish, AllDay: allDay, Timezone: start.Location().String(), Location: plainText(calendarText(event, "LOCATION"), 300)}
			if entry.Title == "" {
				entry.Title = "제목 없는 일정"
			}
			if address := calendarText(event, "URL"); address != "" {
				if safe, err := canonicalArticleURL(address, ""); err == nil {
					entry.URL = safe
				}
			}
			out.Events = append(out.Events, entry)
		}
	}
	sort.SliceStable(out.Events, func(i, j int) bool { return out.Events[i].Start.Before(out.Events[j].Start) })
	if len(out.Events) > 200 {
		out.Events = out.Events[:200]
		warn()
	}
	return out, nil
}

var calendarDurationPattern = regexp.MustCompile(`^P(?:(\d+)W)?(?:(\d+)D)?(?:T(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?)?$`)

func calendarDuration(value string) (int, time.Duration, error) {
	parts := calendarDurationPattern.FindStringSubmatch(value)
	if len(parts) != 6 || value == "P" || value == "PT" {
		return 0, 0, ErrInvalidInput
	}
	numbers := make([]int, 5)
	for i, text := range parts[1:] {
		if text == "" {
			continue
		}
		number, err := strconv.Atoi(text)
		if err != nil || number < 0 || number > 31622400 {
			return 0, 0, ErrInvalidInput
		}
		numbers[i] = number
	}
	days := numbers[0]*7 + numbers[1]
	clock := time.Duration(numbers[2])*time.Hour + time.Duration(numbers[3])*time.Minute + time.Duration(numbers[4])*time.Second
	if days > 366 || clock > 366*24*time.Hour {
		return 0, 0, ErrInvalidInput
	}
	return days, clock, nil
}
