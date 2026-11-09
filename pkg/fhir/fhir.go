// Package fhir holds generic FHIR R4 primitives with no knowledge of this product.
package fhir

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ContentType is the media type of every FHIR response.
const ContentType = "application/fhir+json"

// OperationOutcome reports an error on a FHIR endpoint.
type OperationOutcome struct {
	ResourceType string  `json:"resourceType"`
	Issue        []Issue `json:"issue"`
}

// Issue is one problem in an OperationOutcome. Severity and Code come from the FHIR value sets
// issue-severity and issue-type. Diagnostics never carries personal data or internal error text.
type Issue struct {
	Severity    string `json:"severity"`
	Code        string `json:"code"`
	Diagnostics string `json:"diagnostics,omitempty"`
}

// WriteError sends an OperationOutcome with one error issue.
func WriteError(w http.ResponseWriter, status int, code, diagnostics string) {
	Write(w, status, OperationOutcome{
		ResourceType: "OperationOutcome",
		Issue:        []Issue{{Severity: "error", Code: code, Diagnostics: diagnostics}},
	})
}

// Write sends v as a FHIR JSON response. URLs keep their & (no HTML escaping: this is not HTML).
func Write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", ContentType)
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	// The status is already sent; a failed write means the client went away.
	_ = enc.Encode(v)
}

// InternalError logs err and sends a fixed 500 OperationOutcome: httpx.InternalError for FHIR endpoints. The
// error text never reaches the client.
func InternalError(w http.ResponseWriter, r *http.Request, log *slog.Logger, err error) {
	log.ErrorContext(r.Context(), "internal error", "method", r.Method, "route", r.Pattern, "error", err)
	WriteError(w, http.StatusInternalServerError, "exception", "Internal error.")
}

// Meta is a resource's meta element: the version and when it was last changed.
type Meta struct {
	LastUpdated time.Time `json:"lastUpdated"`
	VersionID   string    `json:"versionId"`
}

// NewMeta is the meta of a resource at version, last updated at (sent in UTC).
func NewMeta(version int32, at time.Time) Meta {
	return Meta{VersionID: strconv.FormatInt(int64(version), 10), LastUpdated: at.UTC()}
}

// Reference points at another resource as ResourceType/id, with a text for it when one is given.
type Reference struct {
	Reference string `json:"reference"`
	Display   string `json:"display,omitempty"`
}

// ReferenceID returns the id of a search parameter's reference to a resource of type typ: "typ/id" or a bare
// id. False for another type, an absolute URL, several values or none.
func ReferenceID(value, typ string) (string, bool) {
	id := strings.TrimPrefix(value, typ+"/")
	if id == "" || strings.ContainsAny(id, "/,:") {
		return "", false
	}
	return id, true
}

// Coding is a code from a code system, with its display when one is given.
type Coding struct {
	System  string `json:"system"`
	Code    string `json:"code"`
	Display string `json:"display,omitempty"`
}

// CodeableConcept is a concept as codings and text.
type CodeableConcept struct {
	Text   string   `json:"text,omitempty"`
	Coding []Coding `json:"coding,omitempty"`
}

// Extension is an extension with a boolean or a code value.
type Extension struct {
	ValueBoolean *bool  `json:"valueBoolean,omitempty"`
	URL          string `json:"url"`
	ValueCode    string `json:"valueCode,omitempty"`
}

// DateRange reads the values of a date search parameter: at most one "ge" and one "lt", each an instant
// with its offset (RFC 3339). It returns the range [from, to), zero for an open end, and false for any other
// value.
func DateRange(values []string) (from, to time.Time, ok bool) {
	for _, v := range values {
		prefix, instant := v[:min(2, len(v))], v[min(2, len(v)):]
		t, err := time.Parse(time.RFC3339, instant)
		switch {
		case err != nil:
			return time.Time{}, time.Time{}, false
		case prefix == "ge" && from.IsZero():
			from = t
		case prefix == "lt" && to.IsZero():
			to = t
		default:
			return time.Time{}, time.Time{}, false
		}
	}
	return from, to, true
}

// ETag is the ETag header of a resource at versionID: a weak tag, as FHIR uses.
func ETag(versionID string) string { return `W/"` + versionID + `"` }

// IfMatch returns the version an If-Match header names (W/"3" or "3"), and false when it names none.
func IfMatch(header string) (string, bool) {
	tag := strings.TrimPrefix(strings.TrimSpace(header), "W/")
	v, ok := strings.CutPrefix(tag, `"`)
	if !ok {
		return "", false
	}
	v, ok = strings.CutSuffix(v, `"`)
	if !ok || v == "" || strings.Contains(v, `"`) {
		return "", false
	}
	return v, true
}

// Bundle is a searchset Bundle: the resources a search matched, then those it included.
type Bundle struct {
	ResourceType string        `json:"resourceType"`
	Type         string        `json:"type"`
	Link         []BundleLink  `json:"link"`
	Entry        []BundleEntry `json:"entry,omitempty"`
	Total        int           `json:"total"`
}

// BundleLink is a link of a Bundle; a searchset has its self link, the search that was run.
type BundleLink struct {
	Relation string `json:"relation"`
	URL      string `json:"url"`
}

// BundleEntry is one resource in a Bundle, at its absolute URL (fullUrl), and why the search returned it.
type BundleEntry struct {
	FullURL  string       `json:"fullUrl"`
	Resource any          `json:"resource"`
	Search   BundleSearch `json:"search"`
}

// BundleSearch says whether an entry matched the search or was included with a match.
type BundleSearch struct {
	Mode string `json:"mode"` // match or include
}

// NewSearchset is the searchset Bundle of the search self (its absolute URL): the matches, then the
// resources included with them. Total counts the matches only.
func NewSearchset(self string, matches, included []BundleEntry) Bundle {
	b := Bundle{
		ResourceType: "Bundle", Type: "searchset", Total: len(matches),
		Link: []BundleLink{{Relation: "self", URL: self}},
	}
	for _, e := range matches {
		e.Search = BundleSearch{Mode: "match"}
		b.Entry = append(b.Entry, e)
	}
	for _, e := range included {
		e.Search = BundleSearch{Mode: "include"}
		b.Entry = append(b.Entry, e)
	}
	return b
}

// IsUUID tells whether s is a UUID in its 8-4-4-4-12 hexadecimal form, as every resource id of this server is
// (CLAUDE.md "FHIR": ids are opaque UUIDs).
func IsUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, r := range s {
		switch {
		case i == 8 || i == 13 || i == 18 || i == 23:
			if r != '-' {
				return false
			}
		case (r < '0' || r > '9') && (r < 'a' || r > 'f') && (r < 'A' || r > 'F'):
			return false
		}
	}
	return true
}
