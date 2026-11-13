package history_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/history"
)

// A cursor is opaque to the client and round-trips the position it was made at, to the nanosecond.
func TestCursor_RoundTrips(t *testing.T) {
	at := time.Date(2026, 11, 3, 9, 30, 0, 123456789, time.UTC)
	c := history.Cursor{Date: at, Reference: "Composition/a1000000-0000-4000-8000-0000000000a1"}
	s := c.String()
	if strings.Contains(s, "Composition") || strings.Contains(s, "2026") {
		t.Errorf("cursor %q shows what it holds, want it opaque", s)
	}
	got, err := history.ParseCursor(s)
	if err != nil || !got.Date.Equal(at) || got.Reference != c.Reference {
		t.Errorf("ParseCursor(%q) = %+v, %v; want %+v", s, got, err, c)
	}
}

// The empty cursor is the first page; anything that is not a cursor this package made is ErrCursor.
func TestParseCursor_TheFirstPageAndBadCursors(t *testing.T) {
	if got, err := history.ParseCursor(""); err != nil || !got.Date.IsZero() || got.Reference != "" {
		t.Errorf("ParseCursor(\"\") = %+v, %v; want the zero cursor", got, err)
	}
	for _, bad := range []string{"x", "!!!", history.Cursor{Reference: "Composition/x"}.String() + "x", "bm90IGEgY3Vyc29y"} {
		if _, err := history.ParseCursor(bad); !errors.Is(err, history.ErrCursor) {
			t.Errorf("ParseCursor(%q) = %v, want ErrCursor", bad, err)
		}
	}
}

// Before orders a history newest first, the reference breaking a tie, so a page ends at a position the next page
// starts strictly after.
func TestCursor_BeforeIsNewestFirst(t *testing.T) {
	at := time.Date(2026, 11, 3, 9, 30, 0, 0, time.UTC)
	c := history.Cursor{Date: at, Reference: "Encounter/b"}
	for _, tc := range []struct {
		item history.Item
		want bool
	}{
		{history.Item{Date: at.Add(-time.Second), Reference: "Encounter/z"}, true},
		{history.Item{Date: at, Reference: "Encounter/a"}, true},
		{history.Item{Date: at, Reference: "Encounter/b"}, false},
		{history.Item{Date: at, Reference: "Encounter/c"}, false},
		{history.Item{Date: at.Add(time.Second), Reference: "Encounter/a"}, false},
	} {
		if got := c.Before(tc.item); got != tc.want {
			t.Errorf("Before(%v %s) = %v, want %v", tc.item.Date, tc.item.Reference, got, tc.want)
		}
	}
	if !(history.Cursor{}).Before(history.Item{Date: at, Reference: "Encounter/a"}) {
		t.Error("the zero cursor: Before = false, want every item")
	}
	if got := history.CursorAt(history.Item{Date: at, Reference: "Encounter/b"}); got != c {
		t.Errorf("CursorAt = %+v, want %+v", got, c)
	}
}
