package main

import (
	"bufio"
	"cmp"
	"encoding/json"
	"io"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// maxLabel is term_label.label's bound (migrations/00030_terminology.sql); a longer label is left out.
const maxLabel = 300

// names is what Wikidata has for one item in one of its language codes: the label and the aliases.
type names struct {
	label   string
	aliases []string
}

// term is one line of assets/terminology/seed.jsonl (docs/terminology/PROGRESS.md, 0.1 "Contracts for the next
// packages"): codes absent when none, several of one kind joined with ", "; per language the preferred label
// first.
type term struct {
	Labels   map[string][]string `json:"labels"`
	Wikidata string              `json:"wikidata"`
	Category string              `json:"category"`
	ICD10    string              `json:"icd10,omitempty"`
	TA98     string              `json:"ta98,omitempty"`
	ATC      string              `json:"atc,omitempty"`
	PCS      string              `json:"pcs,omitempty"`
	Rank     int                 `json:"rank"`
}

// wikidataLanguages are the Wikidata language codes the fetch asks for: Latin, the ten app languages, and
// Serbian's three codes (sr, mostly Cyrillic; sr-ec Cyrillic; sr-el Latin).
func wikidataLanguages() []string {
	return []string{"la", "bs", "hr", "sr", "sr-ec", "sr-el", "en", "de", "ar", "tr", "it", "fr"}
}

// buildLabels turns an item's Wikidata names into the seed's labels per term_language. Serbian (SPEC.md §3
// "Data"): Cyrillic sr and sr-ec fill sr-Cyrl; sr-Latn is sr-el when it has a label, else sr transliterated
// (a Latin sr name is taken as it is). A language with nothing usable is absent; nothing is translated.
func buildLabels(in map[string]*names) map[string][]string {
	out := map[string][]string{}
	for _, lang := range []string{"la", "bs", "hr", "en", "de", "ar", "tr", "it", "fr"} {
		if n := in[lang]; n != nil {
			put(out, lang, *n)
		}
	}
	var cyr, lat names
	if sr := in["sr"]; sr != nil {
		for i, s := range append([]string{sr.label}, sr.aliases...) {
			if containsCyrillic(s) {
				add(&cyr, s, i == 0)
				s = transliterate(s)
			}
			add(&lat, s, i == 0)
		}
	}
	if ec := in["sr-ec"]; ec != nil {
		add(&cyr, ec.label, cyr.label == "")
		cyr.aliases = append(cyr.aliases, ec.aliases...)
	}
	if el := in["sr-el"]; el != nil {
		if normalise(el.label) != "" {
			lat = *el
		} else {
			lat.aliases = append(lat.aliases, el.aliases...)
		}
	}
	put(out, "sr-Cyrl", cyr)
	put(out, "sr-Latn", lat)
	return out
}

func add(n *names, s string, asLabel bool) {
	if asLabel && n.label == "" {
		n.label = s
		return
	}
	n.aliases = append(n.aliases, s)
}

// put stores the label first and the aliases in byte order, each normalised, dropping labels without a letter
// (blanks, emoji, bare numbers), labels over maxLabel and duplicates that differ only in case. Without a label the first alias is preferred.
func put(out map[string][]string, lang string, n names) {
	aliases := make([]string, 0, len(n.aliases))
	for _, a := range n.aliases {
		aliases = append(aliases, normalise(a))
	}
	slices.Sort(aliases)
	var labels []string
	seen := map[string]bool{}
	for _, s := range append([]string{normalise(n.label)}, aliases...) {
		key := strings.ToLower(s)
		if !strings.ContainsFunc(s, unicode.IsLetter) || utf8.RuneCountInString(s) > maxLabel || seen[key] {
			continue
		}
		seen[key] = true
		labels = append(labels, s)
	}
	if len(labels) > 0 {
		out[lang] = labels
	}
}

// normalise trims and collapses every run of white space (no-break spaces included) to one space.
func normalise(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func containsCyrillic(s string) bool {
	return strings.IndexFunc(s, func(r rune) bool { return unicode.Is(unicode.Cyrillic, r) }) >= 0
}

// transliterate writes Serbian Cyrillic in Serbian Latin, letter for letter; Љ, Њ and Џ become LJ, NJ and DŽ
// before another capital, else Lj, Nj and Dž. Anything else is kept.
func transliterate(s string) string {
	cyr := []rune("абвгдђежзијклљмнњопрстћуфхцчџшАБВГДЂЕЖЗИЈКЛЉМНЊОПРСТЋУФХЦЧЏШ")
	lat := strings.Fields("a b v g d đ e ž z i j k l lj m n nj o p r s t ć u f h c č dž š " +
		"A B V G D Đ E Ž Z I J K L Lj M N Nj O P R S T Ć U F H C Č Dž Š")
	runes := []rune(s)
	var b strings.Builder
	for i, r := range runes {
		k := slices.Index(cyr, r)
		switch {
		case k < 0:
			b.WriteRune(r)
		case utf8.RuneCountInString(lat[k]) == 2 && i+1 < len(runes) && unicode.IsUpper(runes[i+1]):
			b.WriteString(strings.ToUpper(lat[k]))
		default:
			b.WriteString(lat[k])
		}
	}
	return b.String()
}

// categoryOrder is term_category's enum order, the seed's first sort key.
func categoryOrder(c string) int {
	return slices.Index([]string{"anatomy", "diagnosis", "symptom", "procedure", "drug"}, c)
}

// writeSeed writes one term per line, sorted by category, then rank (higher first), then Wikidata id
// (numerically), so a fetch of the same data writes the same file.
func writeSeed(w io.Writer, terms []term) error {
	sorted := slices.Clone(terms)
	slices.SortFunc(sorted, func(a, b term) int {
		return cmp.Or(
			cmp.Compare(categoryOrder(a.Category), categoryOrder(b.Category)),
			cmp.Compare(b.Rank, a.Rank),
			cmp.Compare(qNumber(a.Wikidata), qNumber(b.Wikidata)),
		)
	})
	bw := bufio.NewWriter(w)
	enc := json.NewEncoder(bw)
	enc.SetEscapeHTML(false)
	for _, t := range sorted {
		if err := enc.Encode(t); err != nil {
			return err
		}
	}
	return bw.Flush()
}

// qNumber is the number of a Wikidata id such as Q9368.
func qNumber(id string) int {
	n, _ := strconv.Atoi(strings.TrimPrefix(id, "Q"))
	return n
}
