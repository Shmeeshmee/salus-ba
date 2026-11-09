package pdfdoc_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/pkg/pdfdoc"
)

func doc() pdfdoc.Document {
	return pdfdoc.Document{
		Created: time.Date(2030, 1, 7, 9, 30, 0, 0, time.UTC),
		Title:   "Placeholder findings report",
		Meta:    []string{"2030-01-07 09:30 UTC"},
		Sections: []pdfdoc.Section{
			{Heading: "Placeholder findings", Paragraphs: []string{"Placeholder text, čćđšž, ћирилица."}},
			{Heading: "Placeholder tests", Items: []string{"Placeholder one", "Placeholder two"}},
			{Heading: "Placeholder control"},
		},
	}
}

// pages counts the page objects of a PDF.
func pages(b []byte) int { return bytes.Count(b, []byte("/Type /Page\n")) }

func TestRender_WritesAPDFWithItsTitle(t *testing.T) {
	t.Parallel()
	b, err := pdfdoc.Render(doc())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(b, []byte("%PDF-")) || !bytes.HasSuffix(bytes.TrimSpace(b), []byte("%%EOF")) {
		t.Errorf("not a PDF: %q … %q", b[:8], b[len(b)-8:])
	}
	if pages(b) != 1 {
		t.Errorf("pages = %d, want 1", pages(b))
	}
	// The document information's title, as UTF-16 hex.
	if !bytes.Contains(b, []byte("/Title <FEFF0050006C")) || !bytes.Contains(b, []byte("/CreationDate(D:20300107093000")) {
		t.Errorf("the information dictionary has no title or creation date")
	}
}

// The same document renders to the same bytes: nothing random or from the wall clock.
func TestRender_IsDeterministic(t *testing.T) {
	t.Parallel()
	a, err := pdfdoc.Render(doc())
	if err != nil {
		t.Fatal(err)
	}
	b, err := pdfdoc.Render(doc())
	if err != nil || !bytes.Equal(a, b) {
		t.Errorf("two renders differ (%v)", err)
	}
}

// A long section wraps and runs on to more pages; a word longer than a line is broken.
func TestRender_WrapsAndBreaksPages(t *testing.T) {
	t.Parallel()
	d := doc()
	d.Sections[0].Paragraphs = []string{strings.Repeat("Placeholder words that wrap across the page. ", 400), strings.Repeat("x", 500)}
	b, err := pdfdoc.Render(d)
	if err != nil {
		t.Fatal(err)
	}
	if pages(b) < 3 {
		t.Errorf("pages = %d, want at least 3", pages(b))
	}
}

// A paragraph's lines each start a line; a blank line keeps its space; list items carry the bullet.
func TestRender_LinesAndItems(t *testing.T) {
	t.Parallel()
	d := doc()
	d.Sections[0].Paragraphs = []string{"one\n\ntwo"}
	d.Sections[1].Items = []string{strings.Repeat("Placeholder item that is long enough to wrap ", 5)}
	if _, err := pdfdoc.Render(d); err != nil {
		t.Errorf("Render() = %v", err)
	}
}

// A letter the Go fonts do not have is an error, not a blank in a signed document.
func TestRender_RefusesALetterTheFontsLack(t *testing.T) {
	t.Parallel()
	for name, change := range map[string]func(*pdfdoc.Document){
		"title":   func(d *pdfdoc.Document) { d.Title = "مرحبا" },
		"meta":    func(d *pdfdoc.Document) { d.Meta = []string{"日本"} },
		"heading": func(d *pdfdoc.Document) { d.Sections[0].Heading = "日本" },
		"text":    func(d *pdfdoc.Document) { d.Sections[0].Paragraphs = []string{"日本"} },
		"item":    func(d *pdfdoc.Document) { d.Sections[1].Items = []string{"日本"} },
	} {
		d := doc()
		change(&d)
		if _, err := pdfdoc.Render(d); !errors.Is(err, pdfdoc.ErrUnsupported) || strings.ContainsAny(err.Error(), "日م") {
			t.Errorf("%s: Render() = %v, want ErrUnsupported naming no text", name, err)
		}
	}
}

// Supported tells a text the fonts set from one they do not, so a caller can refuse it before rendering.
func TestSupported_TheGoFontsLetters(t *testing.T) {
	t.Parallel()
	for s, want := range map[string]bool{
		"Placeholder čćđšž ČĆĐŠŽ ћирилица ğışİ äöüß àéèç 0-9 \n": true, "— •": true, "": true, "mixed 日本": false, "مرحبا": false, "\U0001F600": false,
	} {
		if got := pdfdoc.Supported(s); got != want {
			t.Errorf("Supported(%q) = %v, want %v", s, got, want)
		}
	}
}
