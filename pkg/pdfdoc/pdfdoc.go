// Package pdfdoc renders a plain text document to PDF: a title, lines under it, and sections of a heading with
// paragraphs or a list, on A4 pages. The text is set in the Go fonts (golang.org/x/image/font/gofont: Latin with
// its Bosnian, Croatian and Serbian letters, Cyrillic, Greek), embedded as subsets, so the PDF needs no font on
// the reader's side. Rendering is deterministic: the same document gives the same bytes. Arabic script (joined,
// right to left) is not shaped. It knows nothing of the product.
package pdfdoc

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/signintech/gopdf"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/sfnt"
)

// Document is what Render writes.
type Document struct {
	Created  time.Time // the information dictionary's CreationDate
	Title    string
	Meta     []string // lines under the title
	Sections []Section
}

// Section is a heading and its paragraphs, or its list items; one with neither shows a dash.
type Section struct {
	Heading    string
	Paragraphs []string
	Items      []string
}

// The page, in points: A4 with 2 cm margins.
const (
	margin     = 56.7
	titleSize  = 16
	headSize   = 11
	textSize   = 10.5
	lineHeight = 1.35
	bullet     = "• "
)

// page is a document being written.
type page struct {
	pdf    *gopdf.GoPdf
	width  float64 // of the text
	bottom float64
}

// ErrUnsupported is returned for a text with a letter the fonts lack, which the PDF would otherwise leave out
// silently. It names no text.
var ErrUnsupported = errors.New("pdfdoc: a letter the fonts do not have")

// Supported reports whether the fonts set every letter of s (line breaks and other control characters aside),
// in both the regular and the bold face.
func Supported(s string) bool {
	var buf sfnt.Buffer
	for _, data := range [][]byte{goregular.TTF, gobold.TTF} {
		f, err := sfnt.Parse(data)
		if err != nil {
			return false
		}
		for _, r := range s {
			if unicode.IsControl(r) {
				continue
			}
			if i, err := f.GlyphIndex(&buf, r); err != nil || i == 0 {
				return false
			}
		}
	}
	return true
}

// Render returns the document as PDF, or ErrUnsupported when a text has a letter the fonts lack.
func Render(d Document) ([]byte, error) {
	texts := append([]string{d.Title, bullet, "—"}, d.Meta...) // and what write adds: the bullet, the dash
	for _, s := range d.Sections {
		texts = append(append(append(texts, s.Heading), s.Paragraphs...), s.Items...)
	}
	if !Supported(strings.Join(texts, "\n")) {
		return nil, ErrUnsupported
	}
	pdf := &gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
	pdf.SetInfo(gopdf.PdfInfo{Title: d.Title, Producer: "Salus", CreationDate: d.Created})
	for _, f := range []struct {
		family string
		data   []byte
	}{{"regular", goregular.TTF}, {"bold", gobold.TTF}} { // in order: the PDF's objects follow it
		if err := pdf.AddTTFFontData(f.family, f.data); err != nil {
			return nil, fmt.Errorf("pdfdoc: font: %w", err)
		}
	}
	p := page{pdf: pdf, width: gopdf.PageSizeA4.W - 2*margin, bottom: gopdf.PageSizeA4.H - margin}
	pdf.AddPage()
	pdf.SetY(margin)
	if err := p.write(d); err != nil {
		return nil, fmt.Errorf("pdfdoc: %w", err)
	}
	return pdf.GetBytesPdf(), nil
}

func (p page) write(d Document) error {
	if err := p.text("bold", titleSize, d.Title, ""); err != nil {
		return err
	}
	for _, line := range d.Meta {
		if err := p.text("regular", textSize, line, ""); err != nil {
			return err
		}
	}
	for _, s := range d.Sections {
		p.pdf.Br(textSize)
		if err := p.text("bold", headSize, s.Heading, ""); err != nil {
			return err
		}
		if len(s.Paragraphs) == 0 && len(s.Items) == 0 {
			s.Paragraphs = []string{"—"}
		}
		for _, para := range s.Paragraphs {
			if err := p.text("regular", textSize, para, ""); err != nil {
				return err
			}
		}
		for _, item := range s.Items {
			if err := p.text("regular", textSize, item, bullet); err != nil {
				return err
			}
		}
	}
	return nil
}

// text writes s in the font family at size, wrapped to the text's width (a word longer than a line is broken),
// each line on the current page or the next one; prefix (a list's bullet) starts the first line, and the lines
// after it are indented by as much.
func (p page) text(family string, size float64, s, prefix string) error {
	if err := p.pdf.SetFont(family, "", size); err != nil {
		return err
	}
	indent := 0.0
	if prefix != "" {
		w, err := p.pdf.MeasureTextWidth(prefix)
		if err != nil {
			return err
		}
		indent = w
	}
	for i, para := range strings.Split(s, "\n") {
		lines, err := p.pdf.SplitTextWithWordWrap(strings.TrimSpace(para)+" ", p.width-indent)
		if err != nil && para != "" {
			return err
		}
		if len(lines) == 0 {
			lines = []string{""}
		}
		for j, line := range lines {
			if p.pdf.GetY()+size*lineHeight > p.bottom {
				p.pdf.AddPage()
				p.pdf.SetY(margin)
			}
			p.pdf.SetX(margin + indent)
			if i == 0 && j == 0 && prefix != "" {
				p.pdf.SetX(margin)
				line = prefix + line
			}
			if err := p.pdf.Cell(nil, strings.TrimRight(line, " ")); err != nil {
				return err
			}
			p.pdf.Br(size * lineHeight)
		}
	}
	return nil
}
