package fhir_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/pkg/fhir"
)

func TestCount_ReadsOnePositiveNumberAndClampsItToTheMaximum(t *testing.T) {
	for _, tt := range []struct {
		values []string
		want   int
		ok     bool
	}{
		{[]string{"1"}, 1, true},
		{[]string{"50"}, 50, true},
		{[]string{"200"}, 200, true},
		{[]string{"201"}, 200, true},
		{[]string{"100000000000000000000"}, 0, false},
		{[]string{"0"}, 0, false},
		{[]string{"-3"}, 0, false},
		{[]string{"ten"}, 0, false},
		{[]string{""}, 0, false},
		{[]string{"5", "5"}, 0, false},
		{[]string{" 5"}, 0, false},
	} {
		got, ok := fhir.Count(tt.values, 200)
		if got != tt.want || ok != tt.ok {
			t.Errorf("Count(%q, 200) = %d, %v; want %d, %v", tt.values, got, ok, tt.want, tt.ok)
		}
	}
}

func TestCursor_RoundTripsAnInstantAndAnIDAndShowsNeitherAsText(t *testing.T) {
	at := time.Date(2030, 1, 7, 9, 0, 0, 123456000, time.FixedZone("CET", 3600))
	id := "0a6f4c1e-58a5-4d7e-9c1b-2f3e4d5c6b7a"

	c := fhir.NewCursor(at, id)
	if strings.Contains(c, id) || strings.Contains(c, "2030") || strings.ContainsAny(c, "+/=&") {
		t.Errorf("cursor %q, want it opaque and safe in a query", c)
	}
	gotAt, gotID, ok := fhir.ParseCursor(c)
	if !ok || !gotAt.Equal(at) || gotID != id {
		t.Errorf("ParseCursor(%q) = %v, %q, %v; want %v, %q", c, gotAt, gotID, ok, at, id)
	}
	gotAt, gotID, ok = fhir.ParseCursor(fhir.NewCursor(time.Time{}, id))
	if !ok || !gotAt.IsZero() || gotID != id {
		t.Errorf("an id-only cursor = %v, %q, %v; want no instant and %q", gotAt, gotID, ok, id)
	}
}

func TestParseCursor_RejectsAnythingItDidNotWrite(t *testing.T) {
	for _, c := range []string{
		"", "not base64!", fhir.NewCursor(time.Date(2030, 1, 7, 9, 0, 0, 0, time.UTC), "not-a-uuid"), "MjAzMC0wMS0wN1QwOTowMDowMFo", // no id
		"eWVzdGVyZGF5fDBhNmY0YzFlLTU4YTUtNGQ3ZS05YzFiLTJmM2U0ZDVjNmI3YQ", // "yesterday|<uuid>"
	} {
		if _, _, ok := fhir.ParseCursor(c); ok {
			t.Errorf("ParseCursor(%q) = ok, want false", c)
		}
	}
}

func TestNewPage_LinksTheNextPageAndGivesTheTotalOnlyForTheWholeAnswer(t *testing.T) {
	m := []fhir.BundleEntry{{FullURL: "https://x.test/fhir/R4/Appointment/1", Resource: map[string]string{"id": "1"}}}
	inc := []fhir.BundleEntry{{FullURL: "https://x.test/fhir/R4/Questionnaire/9", Resource: map[string]string{"id": "9"}}}
	for _, tt := range []struct {
		name, next, want string
		first            bool
	}{
		{"the first page with another after it", "https://x.test/next",
			`{"resourceType":"Bundle","type":"searchset","link":[{"relation":"self","url":"https://x.test/self"},{"relation":"next","url":"https://x.test/next"}],` +
				`"entry":[{"fullUrl":"https://x.test/fhir/R4/Appointment/1","resource":{"id":"1"},"search":{"mode":"match"}},` +
				`{"fullUrl":"https://x.test/fhir/R4/Questionnaire/9","resource":{"id":"9"},"search":{"mode":"include"}}]}`, true},
		{"a later page, the last", "",
			`{"resourceType":"Bundle","type":"searchset","link":[{"relation":"self","url":"https://x.test/self"}],` +
				`"entry":[{"fullUrl":"https://x.test/fhir/R4/Appointment/1","resource":{"id":"1"},"search":{"mode":"match"}},` +
				`{"fullUrl":"https://x.test/fhir/R4/Questionnaire/9","resource":{"id":"9"},"search":{"mode":"include"}}]}`, false},
		{"the first page and the last: the whole answer", "",
			`{"resourceType":"Bundle","type":"searchset","link":[{"relation":"self","url":"https://x.test/self"}],` +
				`"entry":[{"fullUrl":"https://x.test/fhir/R4/Appointment/1","resource":{"id":"1"},"search":{"mode":"match"}},` +
				`{"fullUrl":"https://x.test/fhir/R4/Questionnaire/9","resource":{"id":"9"},"search":{"mode":"include"}}],"total":1}`, true},
	} {
		got, err := json.Marshal(fhir.NewPage("https://x.test/self", tt.next, tt.first, m, inc))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != tt.want {
			t.Errorf("%s:\n got %s\nwant %s", tt.name, got, tt.want)
		}
	}
	whole, _ := json.Marshal(fhir.NewPage("https://x.test/self", "", true, m, inc))
	old, _ := json.Marshal(fhir.NewSearchset("https://x.test/self", m, inc))
	if string(whole) != string(old) {
		t.Errorf("the whole answer as a page = %s, want the searchset as before: %s", whole, old)
	}
}
