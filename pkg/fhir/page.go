package fhir

import (
	"encoding/base64"
	"maps"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Count reads the values of _count, the page size a search asks for: one whole number of at least 1, and max for
// one above it (a server may return fewer matches a page than asked). False for any other value.
func Count(values []string, max int) (int, bool) {
	if len(values) != 1 || strings.TrimLeft(values[0], "0123456789") != "" {
		return 0, false
	}
	n, err := strconv.Atoi(values[0])
	if err != nil || n < 1 {
		return 0, false
	}
	return min(n, max), true
}

// NewCursor is the opaque position of a next link: an instant (none when zero) and an id, both of the page's last
// match. It is encoded, not encrypted: put nothing in it that the page does not already show.
func NewCursor(at time.Time, id string) string {
	s := "|" + id
	if !at.IsZero() {
		s = at.UTC().Format(time.RFC3339Nano) + s
	}
	return base64.RawURLEncoding.EncodeToString([]byte(s))
}

// ParseCursor reads a cursor NewCursor wrote, and false for any other value.
func ParseCursor(c string) (time.Time, string, bool) {
	b, err := base64.RawURLEncoding.DecodeString(c)
	if err != nil {
		return time.Time{}, "", false
	}
	instant, id, ok := strings.Cut(string(b), "|")
	if !ok || !IsUUID(id) {
		return time.Time{}, "", false
	}
	if instant == "" {
		return time.Time{}, id, true
	}
	at, err := time.Parse(time.RFC3339Nano, instant)
	if err != nil {
		return time.Time{}, "", false
	}
	return at, id, true
}

// NewPage is one page of a search's searchset: the matches, then the resources included with them, with its
// next link when another page follows. The page that is the whole answer (the first, with no next) is the searchset
// NewSearchset gives, with its total; any other page has no total: this server does not count the matches of the
// pages it has not read.
func NewPage(self, next string, first bool, matches, included []BundleEntry) any {
	b := NewSearchset(self, matches, included)
	if first && next == "" {
		return b
	}
	if next != "" {
		b.Link = append(b.Link, BundleLink{Relation: "next", URL: next})
	}
	return pageBundle{ResourceType: b.ResourceType, Type: b.Type, Link: b.Link, Entry: b.Entry}
}

// pageBundle is a searchset Bundle without its total: a page of a longer answer.
type pageBundle struct {
	ResourceType string        `json:"resourceType"`
	Type         string        `json:"type"`
	Link         []BundleLink  `json:"link"`
	Entry        []BundleEntry `json:"entry,omitempty"`
}

// SelfLink is the URL of the search q on base (the resource type's absolute URL), as it was run.
func SelfLink(base string, q url.Values) string {
	if len(q) == 0 {
		return base
	}
	return base + "?" + q.Encode()
}

// NextLink is the URL of the page after the search q's: the same search from cursor on.
func NextLink(base string, q url.Values, cursor string) string {
	next := maps.Clone(q)
	if next == nil {
		next = url.Values{}
	}
	next.Set("_cursor", cursor)
	return base + "?" + next.Encode()
}
