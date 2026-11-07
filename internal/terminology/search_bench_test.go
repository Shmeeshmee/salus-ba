package terminology_test

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/terminology"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// The seed's size (docs/terminology/SPEC.md §3 "Data"): about 5,300 terms by category.
var benchCategories = []struct {
	name  string
	terms int
}{{"anatomy", 1500}, {"diagnosis", 2000}, {"symptom", 500}, {"procedure", 500}, {"drug", 800}}

// alphabet is a script's sounds for synthetic words: a syllable is an onset, a vowel and, one time in three, a
// coda, so the labels have about as many distinct trigrams as real words do.
type alphabet struct{ onsets, vowels, codas []string }

// The scripts: bs, hr and sr-Latn share the South Slavic Latin one with its accents.
var (
	latinScript = alphabet{strings.Fields("b c d f g h l m n p r s t v ph th tr pl cr st gr br sp"),
		strings.Fields("a e i o u y ae"), strings.Fields("s n r l m x t c")}
	southScript = alphabet{strings.Fields("b c č ć d dž đ f g h j k l lj m n nj p r s š t v z ž"),
		strings.Fields("a e i o u"), strings.Fields("n r l j m s k t")}
	cyrillicScript = alphabet{strings.Fields("б ц ч ћ д џ ђ ф г х ј к л љ м н њ п р с ш т в з ж"),
		strings.Fields("а е и о у"), strings.Fields("н р л ј м с к т")}
	arabicScript = alphabet{strings.Fields("ب ت ث ج ح خ د ذ ر ز س ش ص ض ط ظ ع غ ف ق ك ل م ن ه"),
		strings.Fields("ا و ي"), strings.Fields("ن ر ل م س ك ت")}
)

// word is a synthetic word of 2 to 4 syllables in a.
func word(r *rand.Rand, a alphabet) string {
	var b strings.Builder
	for range 2 + r.IntN(3) {
		b.WriteString(a.onsets[r.IntN(len(a.onsets))] + a.vowels[r.IntN(len(a.vowels))])
		if r.IntN(3) == 0 {
			b.WriteString(a.codas[r.IntN(len(a.codas))])
		}
	}
	return b.String()
}

// label is a synthetic label of one to three words, as most medical terms are.
func label(r *rand.Rand, a alphabet) string {
	words := make([]string, 1+r.IntN(3))
	for i := range words {
		words[i] = word(r, a)
	}
	return strings.Join(words, " ")
}

func scriptOf(language string) alphabet {
	switch language {
	case "bs", "hr", "sr-Latn":
		return southScript
	case "sr-Cyrl":
		return cyrillicScript
	case "ar":
		return arabicScript
	}
	return latinScript
}

// benchLabels are the synthetic labels the benchmark loads, flattened into the columns of term_label.
type benchLabels struct {
	termIDs, languages, texts []string
	preferred                 []bool
}

// loadTerms writes the synthetic terms as owner: per term, in Latin (70%) and each app language (85%), a
// preferred label and up to two synonyms. It returns every label, for the searches.
func loadTerms(b *testing.B, owner *pgxpool.Pool) []string {
	b.Helper()
	r := rand.New(rand.NewPCG(1, 2))
	var ids, cats []string
	var ranks []int32
	var l benchLabels
	for _, c := range benchCategories {
		for range c.terms {
			id := fmt.Sprintf("00000000-0000-4000-8000-%012d", len(ids)+1)
			ids, cats, ranks = append(ids, id), append(cats, c.name), append(ranks, int32(r.IntN(300)))
			for _, lang := range []string{"la", "bs", "hr", "sr-Latn", "sr-Cyrl", "en", "de", "ar", "tr", "it", "fr"} {
				if (lang == "la" && r.IntN(10) >= 7) || r.IntN(100) >= 85 {
					continue
				}
				seen := []string{}
				for i := range 1 + r.IntN(3)/2 + r.IntN(2) {
					text := label(r, scriptOf(lang))
					if slices.Contains(seen, text) {
						continue
					}
					seen = append(seen, text)
					l.termIDs, l.languages, l.texts, l.preferred = append(l.termIDs, id), append(l.languages, lang), append(l.texts, text), append(l.preferred, i == 0)
				}
			}
		}
	}
	_, err := owner.Exec(b.Context(), `INSERT INTO term (id, category, rank) SELECT * FROM unnest($1::uuid[], $2::term_category[], $3::integer[])`,
		ids, cats, ranks)
	if err == nil {
		_, err = owner.Exec(b.Context(), `INSERT INTO term_label (term_id, language, label, preferred)
			SELECT * FROM unnest($1::uuid[], $2::term_language[], $3::text[], $4::boolean[])`, l.termIDs, l.languages, l.texts, l.preferred)
	}
	if err == nil {
		_, err = owner.Exec(b.Context(), "VACUUM ANALYZE term, term_label") // merges the GIN pending list, as autovacuum does after an import
	}
	if err != nil {
		b.Fatal(err)
	}
	b.Logf("%d terms, %d labels", len(ids), len(l.texts))
	return l.texts
}

// queries are what a doctor types while the palette searches: the first three to six letters of a label,
// a whole label, or a word with one letter missing.
func queries(labels []string, n int) []string {
	r := rand.New(rand.NewPCG(3, 4))
	out := make([]string, n)
	for i := range out {
		l := []rune(labels[r.IntN(len(labels))])
		switch i % 3 {
		case 0:
			out[i] = string(l[:min(len(l), 3+r.IntN(4))])
		case 1:
			out[i] = string(l)
		default:
			cut := r.IntN(len(l))
			out[i] = string(append(l[:cut:cut], l[cut+1:]...))
		}
		if strings.TrimSpace(out[i]) == "" {
			out[i] = string(l)
		}
	}
	return out
}

// BenchmarkSearch runs b.N searches through the service on the app's role, alone and 50 at a time, on synthetic
// terms of the seed's size (5,300 terms, labels in Latin and the ten app languages), with the app's default pool.
// It reports the p50 and p95 of one search (docs/terminology/SPEC.md §3 "Search": p95 under 15 ms at 50).
func BenchmarkSearch(b *testing.B) {
	for _, concurrent := range []int{1, 50} {
		b.Run(fmt.Sprintf("concurrent=%d", concurrent), func(b *testing.B) { benchmarkSearch(b, concurrent) })
	}
}

func benchmarkSearch(b *testing.B, concurrent int) {
	owner, app := pgtest.NewAppDatabase(b)
	texts := queries(loadTerms(b, owner), b.N)
	svc := terminology.NewService(terminology.NewPostgresRepository(app), nil)
	staff := terminology.Caller{AccountID: "bench", Staff: true}
	latencies := make([]time.Duration, b.N)
	next := make(chan int, b.N)
	for i := range b.N {
		next <- i
	}
	close(next)

	b.ResetTimer()
	var wg sync.WaitGroup
	var failed sync.Once
	for range concurrent {
		wg.Go(func() {
			for i := range next {
				start := clock.System{}.Now()
				_, err := svc.Search(b.Context(), staff, terminology.Search{Text: texts[i], Languages: []string{"la", "hr", "en"}, Limit: 10})
				latencies[i] = time.Since(start)
				if err != nil {
					failed.Do(func() { b.Error(err) })
				}
			}
		})
	}
	wg.Wait()
	b.StopTimer()

	slices.Sort(latencies)
	ms := func(q float64) float64 {
		return float64(latencies[int(q*float64(len(latencies)-1))].Microseconds()) / 1000
	}
	b.ReportMetric(ms(0.5), "p50-ms")
	b.ReportMetric(ms(0.95), "p95-ms")
}
