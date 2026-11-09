package composition

import (
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"
)

// The limits of a template.
const (
	maxTitle       = 200
	maxLabel       = 200
	maxInstruction = 1000
	maxFields      = 50
)

// languages are the tags of the enum language in its order (docs/i18n/SPEC.md §3 "Tags").
var languages = []string{"bs", "hr", "sr-Latn", "sr-Cyrl", "en", "de", "ar", "tr", "it", "fr"}

var (
	keyPattern   = regexp.MustCompile(`^[a-z][a-z0-9_]{0,39}$`)
	loincPattern = regexp.MustCompile(`^[0-9]{1,7}-[0-9]$`)
)

// knownLanguage reports whether a template may be written in the language tag: any of the enum's but Arabic,
// whose joined, right-to-left script the PDF (pkg/pdfdoc) does not set.
func knownLanguage(tag string) bool { return tag != "ar" && slices.Contains(languages, tag) }

func blank(s string) bool { return strings.TrimSpace(s) == "" }

func chars(s string) int { return utf8.RuneCountInString(s) }

// ValidateTemplate checks a template's content: a title, a known original language, status and LOINC type; 1 to
// 50 fields, each a key of lowercase letters, digits and underscores (once), a known type, a label and an
// instruction within their limits; each translation a known language other than the original, once, complete
// (its title, every field's label, and an instruction exactly where the original has one) and naming no other
// field. It returns an *InvalidTemplateError, or nil.
func ValidateTemplate(c TemplateContent) error {
	whole := blank(c.Title) || chars(c.Title) > maxTitle || !knownLanguage(c.Language) ||
		(c.Status != TemplateDraft && c.Status != TemplateActive) || !loincPattern.MatchString(c.TypeCode) ||
		len(c.Fields) == 0 || len(c.Fields) > maxFields
	var bad []string
	seen := make(map[string]bool, len(c.Fields))
	for _, f := range c.Fields {
		if !validField(f) || seen[f.Key] {
			bad = append(bad, f.Key)
		}
		seen[f.Key] = true
	}
	langs := map[string]bool{c.Language: true}
	for _, tr := range c.Translations {
		if !knownLanguage(tr.Language) || langs[tr.Language] || blank(tr.Title) || chars(tr.Title) > maxTitle {
			whole = true
		}
		langs[tr.Language] = true
		for key := range tr.Fields {
			if !seen[key] {
				whole = true
			}
		}
	}
	for _, f := range c.Fields {
		if !slices.Contains(bad, f.Key) && !translated(f, c.Translations) {
			bad = append(bad, f.Key)
		}
	}
	if whole || len(bad) > 0 {
		return &InvalidTemplateError{Keys: bad}
	}
	return nil
}

func validField(f TemplateField) bool {
	switch f.Type {
	case FieldText, FieldList, FieldDate, FieldCoded:
	default:
		return false
	}
	return keyPattern.MatchString(f.Key) && !blank(f.Label) && chars(f.Label) <= maxLabel && chars(f.Instruction) <= maxInstruction
}

// translated reports whether every translation has the field's texts: a label, and an instruction exactly
// where the original has one, within the limits.
func translated(f TemplateField, ts []Translation) bool {
	for _, tr := range ts {
		texts, ok := tr.Fields[f.Key]
		if !ok || blank(texts.Label) || chars(texts.Label) > maxLabel || chars(texts.Instruction) > maxInstruction ||
			blank(texts.Instruction) != blank(f.Instruction) {
			return false
		}
	}
	return true
}
