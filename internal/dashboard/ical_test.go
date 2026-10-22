package dashboard

import (
	"context"
	ics "github.com/arran4/golang-ical"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func calendarFixture(events string) []byte {
	return []byte("BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//Glance Test//EN\r\n" + strings.ReplaceAll(events, "\n", "\r\n") + "END:VCALENDAR\r\n")
}
func TestICalendarDSTRecurrenceExceptionsAndAllDay(t *testing.T) {
	fixture := calendarFixture(`BEGIN:VEVENT
UID:weekly
DTSTART;TZID=America/New_York:20261025T090000
DTEND;TZID=America/New_York:20261025T100000
RRULE:FREQ=WEEKLY;COUNT=4
EXDATE;TZID=America/New_York:20261108T090000
SUMMARY:Weekly
END:VEVENT
BEGIN:VEVENT
UID:weekly
RECURRENCE-ID;TZID=America/New_York:20261115T090000
DTSTART;TZID=America/New_York:20261115T110000
DTEND;TZID=America/New_York:20261115T120000
SUMMARY:Moved
END:VEVENT
BEGIN:VEVENT
UID:all-day
DTSTART;VALUE=DATE:20261031
DTEND;VALUE=DATE:20261102
RRULE:FREQ=WEEKLY;COUNT=2
SUMMARY:Holiday
END:VEVENT
BEGIN:VEVENT
UID:cancel
DTSTART;TZID=America/New_York:20261101T130000
DTEND;TZID=America/New_York:20261101T140000
RRULE:FREQ=DAILY;COUNT=2
SUMMARY:Cancel one
END:VEVENT
BEGIN:VEVENT
UID:cancel
RECURRENCE-ID;TZID=America/New_York:20261102T130000
STATUS:CANCELLED
END:VEVENT
`)
	now := time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC)
	data, err := parseICalendar(context.Background(), fixture, "America/New_York", now)
	if err != nil {
		t.Fatal(err)
	}
	if len(data.Events) != 5 || len(data.Warnings) != 0 {
		t.Fatalf("unexpected occurrences: %+v", data)
	}
	var holiday []CalendarEvent
	for _, event := range data.Events {
		switch event.Title {
		case "Weekly":
			if event.Start.UTC().Hour() != 14 || event.Start.Day() != 1 {
				t.Fatal("DST wall clock shifted", event)
			}
		case "Moved":
			if event.Start.Hour() != 11 || event.Start.Day() != 15 {
				t.Fatal("exception lost", event)
			}
		case "Holiday":
			holiday = append(holiday, event)
			if !event.AllDay || event.End.Day()-event.Start.Day() != 2 && event.End.Month() == event.Start.Month() {
				t.Fatal("all-day boundary", event)
			}
		}
	}
	if len(holiday) != 2 || holiday[0].End.Sub(holiday[0].Start) != 49*time.Hour || holiday[1].End.Sub(holiday[1].Start) != 48*time.Hour {
		t.Fatal("DST all-day duration", holiday)
	}
	again, err := parseICalendar(context.Background(), fixture, "America/New_York", now)
	if err != nil || data.Events[0].ID != again.Events[0].ID {
		t.Fatal("unstable occurrence ID")
	}
}
func TestICalendarBoundariesAndNoUnsafeMarkup(t *testing.T) {
	now := time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC)
	for _, fixture := range [][]byte{[]byte("not calendar"), calendarFixture(strings.Repeat("BEGIN:VALARM\n", 20)), calendarFixture(strings.Repeat("BEGIN:VEVENT\nUID:x\nEND:VEVENT\n", 1001))} {
		if _, err := parseICalendar(context.Background(), fixture, "UTC", now); err == nil {
			t.Fatal("invalid calendar accepted")
		}
	}
	data, err := parseICalendar(context.Background(), calendarFixture("BEGIN:VEVENT\nUID:bad\nDTSTART:20261022T090000Z\nRRULE:FREQ=SECONDLY\nSUMMARY:Unsupported\nEND:VEVENT\n"), "UTC", now)
	if err != nil || len(data.Events) != 0 || len(data.Warnings) == 0 {
		t.Fatal("unbounded recurrence accepted", data, err)
	}
	data, err = parseICalendar(context.Background(), calendarFixture("BEGIN:VEVENT\nUID:safe\nDTSTART:20261022T090000Z\nSUMMARY:<b>Safe</b><script>hidden()</script>\nURL:javascript:alert(1)\nATTACH:https://127.0.0.1/private\nEND:VEVENT\n"), "UTC", now)
	if err != nil || len(data.Events) != 1 || data.Events[0].Title != "Safe" || data.Events[0].URL != "" {
		t.Fatal("unsafe calendar content", data, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = parseICalendar(ctx, calendarFixture("BEGIN:VEVENT\nUID:a\nDTSTART:20261022T090000Z\nEND:VEVENT\n"), "UTC", now); err == nil {
		t.Fatal("cancellation ignored")
	}
}
func TestPrivateCalendarAddressIsResolvedOnlyOnServer(t *testing.T) {
	store := testStore(t)
	connection, err := store.CreateConnection(ConnectionInput{Name: "Calendar", Kind: "calendar", Endpoint: "https://calendar.example.com/private/fixture-token.ics"})
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	provider := &Providers{store: store, resolve: func(context.Context, string) ([]net.IP, error) { return []net.IP{net.ParseIP("8.8.8.8")}, nil }, proxy: func(*http.Request) (*url.URL, error) { return nil, nil }, client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Host != "calendar.example.com" || r.URL.Path != "/private/fixture-token.ics" || r.Header.Get("Authorization") != "" {
			t.Fatal("private address not resolved safely")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("BEGIN:VCALENDAR\r\nVERSION:2.0\r\nEND:VCALENDAR\r\n"))}, nil
	})}}
	source := Source{ID: "calendar", Type: "ical", ConnectionID: connection.ID, Config: raw(ICalSource{Timezone: "UTC"})}
	if _, err = provider.Fetch(context.Background(), source); err != nil || calls != 1 {
		t.Fatal(err)
	}
	if strings.Contains(string(raw(source)), "fixture-token") {
		t.Fatal("private address in source")
	}
}

func TestCalendarSpringDSTAndDuration(t *testing.T) {
	fixture := calendarFixture("BEGIN:VEVENT\nUID:spring\nDTSTART;TZID=America/New_York:20260301T090000\nDTEND;TZID=America/New_York:20260301T100000\nRRULE:FREQ=WEEKLY;COUNT=2\nSUMMARY:Spring\nEND:VEVENT\nBEGIN:VEVENT\nUID:duration\nDTSTART:20260302T090000Z\nDURATION:PT1H30M\nSUMMARY:Duration\nEND:VEVENT\n")
	data, err := parseICalendar(context.Background(), fixture, "UTC", time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC))
	if err != nil || len(data.Events) != 3 {
		t.Fatal(data, err)
	}
	for _, event := range data.Events {
		if event.Title == "Spring" {
			if event.Start.Hour() != 9 || event.Start.Day() == 8 && event.Start.UTC().Hour() != 13 {
				t.Fatal("spring DST shifted", event)
			}
		} else if event.End.Sub(event.Start) != 90*time.Minute {
			t.Fatal("duration lost", event)
		}
	}
	location, _ := time.LoadLocation("America/New_York")
	prop := &ics.IANAProperty{BaseProperty: ics.BaseProperty{Value: "20260308T023000"}}
	if _, _, err := calendarTime(prop, location); err == nil {
		t.Fatal("nonexistent wall time silently shifted")
	}
}
