package main

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

const entityPrefix = "http://www.wikidata.org/entity/"

// category is one term_category: the SPARQL pattern that binds its ?item, and how many of the most common
// items it keeps (SPEC.md §3 "Data").
type category struct {
	name  string
	where string
	limit int
}

// categories in the order they claim items: an item that fits several is kept in the first. Symptoms first (a
// symptom with an ICD-10 R code stays a symptom); diagnosis last, as anything else with an ICD-10 code.
// Symptoms are a direct instance or subclass: the transitive closure (wdt:P279*) times out on the service.
func categories() []category {
	return []category{
		{name: "symptom", limit: 500, where: `VALUES ?class { wd:Q169872 wd:Q1441305 }
  ?item wdt:P31|wdt:P279 ?class .`},
		{name: "anatomy", limit: 1500, where: `?item wdt:P1323 [] .`},
		{name: "procedure", limit: 500, where: `{ ?item wdt:P1690 [] . }
  UNION { ?item wdt:P1692 ?icd9 . FILTER(REGEX(?icd9, "^[0-9]{2}(\\.[0-9]{1,2})?$")) }`},
		{name: "drug", limit: 800, where: `?item wdt:P267 ?atc .
  FILTER(REGEX(?atc, "^[A-Z][0-9]{2}[A-Z]{2}[0-9]{2}$"))`},
		{name: "diagnosis", limit: 2000, where: `?item wdt:P494 [] .`},
	}
}

// rankQuery lists a category's items with their sitelinks, the most written about first. The first line tags
// the query for the service's logs and the recorded tests.
func rankQuery(c category, limit int) string {
	return fmt.Sprintf(`#salus-terminology rank %s
SELECT DISTINCT ?item ?links WHERE {
  %s
  ?item wikibase:sitelinks ?links .
} ORDER BY DESC(?links) ?item
LIMIT %d`, c.name, c.where, limit)
}

// detailsQuery asks for the labels, aliases and codes of a batch of items.
func detailsQuery(ids []string) string {
	langs := `"` + strings.Join(wikidataLanguages(), `", "`) + `"`
	return fmt.Sprintf(`#salus-terminology details
SELECT ?item ?field ?value WHERE {
  VALUES ?item { wd:%s }
  {
    ?item rdfs:label ?value . BIND("label" AS ?field)
    FILTER(LANG(?value) IN (%s))
  } UNION {
    ?item skos:altLabel ?value . BIND("alias" AS ?field)
    FILTER(LANG(?value) IN (%s))
  } UNION {
    VALUES (?p ?field) { (wdt:P494 "icd10") (wdt:P1323 "ta98") (wdt:P267 "atc") (wdt:P1690 "pcs") (wdt:P1692 "icd9cm") }
    ?item ?p ?value .
  }
}`, strings.Join(ids, " wd:"), langs, langs)
}

// item is what the details query found for one Wikidata item: its names per Wikidata language and its codes.
type item struct {
	names                 map[string]*names
	icd10, ta98, atc, pcs string
}

var (
	qidRe       = regexp.MustCompile(`^Q[1-9][0-9]*$`)
	icd9ProcRe  = regexp.MustCompile(`^[0-9]{2}(\.[0-9]{1,2})?$`)
	codeFieldOf = map[string]string{"icd10": "icd10", "ta98": "ta98", "atc": "atc", "pcs": "pcs", "icd9cm": "pcs"}
)

// details gathers the details query's rows per item. ICD-9-CM codes count as procedure codes (pcs) only in
// the procedure format (two digits, an optional decimal part); diagnosis codes are left out.
func details(rows []row) map[string]item {
	codes := map[string]map[string][]string{}
	nms := map[string]map[string]*names{}
	for _, r := range rows {
		id := strings.TrimPrefix(r["item"].Value, entityPrefix)
		field, v := r["field"].Value, r["value"]
		if nms[id] == nil {
			nms[id], codes[id] = map[string]*names{}, map[string][]string{}
		}
		switch field {
		case "label", "alias":
			n := nms[id][v.Lang]
			if n == nil {
				n = &names{}
				nms[id][v.Lang] = n
			}
			if field == "label" {
				n.label = v.Value
			} else {
				n.aliases = append(n.aliases, v.Value)
			}
		case "icd9cm":
			if icd9ProcRe.MatchString(v.Value) {
				codes[id]["pcs"] = append(codes[id]["pcs"], v.Value)
			}
		default:
			codes[id][codeFieldOf[field]] = append(codes[id][codeFieldOf[field]], strings.TrimSpace(v.Value))
		}
	}
	out := make(map[string]item, len(nms))
	for id, n := range nms {
		c := codes[id]
		out[id] = item{names: n, icd10: joinCodes(c["icd10"]), ta98: joinCodes(c["ta98"]), atc: joinCodes(c["atc"]), pcs: joinCodes(c["pcs"])}
	}
	return out
}

func joinCodes(cs []string) string {
	cs = slices.DeleteFunc(slices.Clone(cs), func(c string) bool { return c == "" })
	slices.Sort(cs)
	return strings.Join(slices.Compact(cs), ", ")
}

// pick is an item a category keeps, with its rank.
type pick struct {
	id, category string
	rank         int
}

// fetch ranks every category, then asks for the details of the kept items batch by batch, one request at a
// time. An item without any label in our languages is left out.
func fetch(ctx context.Context, s *sparql, cats []category, batch int) ([]term, error) {
	picks, err := rank(ctx, s, cats)
	if err != nil {
		return nil, err
	}
	found := map[string]item{}
	for chunk := range slices.Chunk(picks, batch) {
		ids := make([]string, len(chunk))
		for i, p := range chunk {
			ids[i] = p.id
		}
		rows, err := s.query(ctx, detailsQuery(ids))
		if err != nil {
			return nil, fmt.Errorf("details: %w", err)
		}
		for id, it := range details(rows) {
			found[id] = it
		}
	}
	terms := make([]term, 0, len(picks))
	for _, p := range picks {
		it := found[p.id]
		labels := buildLabels(it.names)
		if len(labels) == 0 {
			continue
		}
		terms = append(terms, term{Wikidata: p.id, Category: p.category, Rank: p.rank, Labels: labels,
			ICD10: it.icd10, TA98: it.ta98, ATC: it.atc, PCS: it.pcs})
	}
	return terms, nil
}

// rank keeps each category's most common items that no earlier category took. It asks for the limit plus
// every item already taken, so the category fills even when all of those overlap.
func rank(ctx context.Context, s *sparql, cats []category) ([]pick, error) {
	var picks []pick
	taken := map[string]bool{}
	for _, c := range cats {
		rows, err := s.query(ctx, rankQuery(c, c.limit+len(taken)))
		if err != nil {
			return nil, fmt.Errorf("rank %s: %w", c.name, err)
		}
		kept := 0
		for _, r := range rows {
			id := strings.TrimPrefix(r["item"].Value, entityPrefix)
			if kept == c.limit {
				break
			}
			if !qidRe.MatchString(id) || taken[id] {
				continue
			}
			var links int
			if _, err := fmt.Sscan(r["links"].Value, &links); err != nil {
				return nil, fmt.Errorf("rank %s: sitelinks of %s: %w", c.name, id, err)
			}
			taken[id] = true
			picks = append(picks, pick{id: id, category: c.name, rank: links})
			kept++
		}
	}
	return picks, nil
}
