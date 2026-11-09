package main

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

func TestBuildLabels(t *testing.T) {
	long := strings.Repeat("a", 301)
	tests := []struct {
		in   map[string]*names
		want map[string][]string
		name string
	}{
		{
			name: "the label first, then the aliases sorted, trimmed, spaces collapsed, no duplicates, none without a letter",
			in: map[string]*names{
				"la": {label: " hepar ", aliases: []string{"iecur", "Hepar", "  ", "hepar", "a  b", long}},
				"en": {label: "liver", aliases: []string{"👀", "1-2"}},
				"xx": {label: "ignored"},
			},
			want: map[string][]string{"la": {"hepar", "a b", "iecur"}, "en": {"liver"}},
		},
		{
			name: "without a label the first alias is preferred",
			in:   map[string]*names{"bs": {aliases: []string{"jetra", "džigerica"}}},
			want: map[string][]string{"bs": {"džigerica", "jetra"}},
		},
		{
			name: "Cyrillic Serbian fills sr-Cyrl and its transliteration sr-Latn",
			in:   map[string]*names{"sr": {label: "јетра", aliases: []string{"џигерица"}}},
			want: map[string][]string{"sr-Cyrl": {"јетра", "џигерица"}, "sr-Latn": {"jetra", "džigerica"}},
		},
		{
			name: "an sr-el label wins over the transliteration",
			in: map[string]*names{
				"sr":    {label: "јетра"},
				"sr-el": {label: "jetra", aliases: []string{"džigerica"}},
			},
			want: map[string][]string{"sr-Cyrl": {"јетра"}, "sr-Latn": {"jetra", "džigerica"}},
		},
		{
			name: "a Latin sr label goes to sr-Latn only; sr-ec adds to sr-Cyrl",
			in: map[string]*names{
				"sr":    {label: "jetra", aliases: []string{"џигерица"}},
				"sr-ec": {label: "џигерица", aliases: []string{"јетра"}},
			},
			want: map[string][]string{"sr-Cyrl": {"џигерица", "јетра"}, "sr-Latn": {"jetra", "džigerica"}},
		},
		{
			name: "nothing usable leaves no language",
			in:   map[string]*names{"de": {label: " "}},
			want: map[string][]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildLabels(tt.in); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildLabels = %q\nwant %q", got, tt.want)
			}
		})
	}
}

func TestTransliterate(t *testing.T) {
	tests := map[string]string{
		"јетра":         "jetra",
		"Љубљана":       "Ljubljana",
		"ЉУБАВ":         "LJUBAV",
		"Џеп":           "Džep",
		"ХИВ инфекција": "HIV infekcija",
		"ђ ћ ч ш ж њ":   "đ ć č š ž nj",
		"COVID-19":      "COVID-19",
	}
	for in, want := range tests {
		if got := transliterate(in); got != want {
			t.Errorf("transliterate(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestWriteSeedSortsAndOmitsEmptyCodes(t *testing.T) {
	terms := []term{
		{Wikidata: "Q12136", Category: "diagnosis", Rank: 50, ICD10: "J45", Labels: map[string][]string{"en": {"asthma"}}},
		{Wikidata: "Q10", Category: "anatomy", Rank: 7, Labels: map[string][]string{"en": {"b"}}},
		{Wikidata: "Q9", Category: "anatomy", Rank: 7, Labels: map[string][]string{"en": {"a"}}},
		{Wikidata: "Q9368", Category: "anatomy", Rank: 200, TA98: "A05.8.01.001",
			Labels: map[string][]string{"la": {"hepar", "iecur"}, "bs": {"jetra"}}},
		{Wikidata: "Q18216", Category: "drug", Rank: 90, ATC: "N02BA01, B01AC06", Labels: map[string][]string{"en": {"aspirin & co"}}},
		{Wikidata: "Q1", Category: "symptom", Rank: 1, Labels: map[string][]string{"en": {"pain"}}},
		{Wikidata: "Q2", Category: "procedure", Rank: 1, PCS: "0DTJ4ZZ", Labels: map[string][]string{"en": {"appendectomy"}}},
	}
	var buf bytes.Buffer
	if err := writeSeed(&buf, terms); err != nil {
		t.Fatalf("writeSeed: %v", err)
	}
	want := `{"labels":{"bs":["jetra"],"la":["hepar","iecur"]},"wikidata":"Q9368","category":"anatomy","ta98":"A05.8.01.001","rank":200}
{"labels":{"en":["a"]},"wikidata":"Q9","category":"anatomy","rank":7}
{"labels":{"en":["b"]},"wikidata":"Q10","category":"anatomy","rank":7}
{"labels":{"en":["asthma"]},"wikidata":"Q12136","category":"diagnosis","icd10":"J45","rank":50}
{"labels":{"en":["pain"]},"wikidata":"Q1","category":"symptom","rank":1}
{"labels":{"en":["appendectomy"]},"wikidata":"Q2","category":"procedure","pcs":"0DTJ4ZZ","rank":1}
{"labels":{"en":["aspirin & co"]},"wikidata":"Q18216","category":"drug","atc":"N02BA01, B01AC06","rank":90}
`
	if buf.String() != want {
		t.Errorf("seed =\n%s\nwant\n%s", buf.String(), want)
	}
}
