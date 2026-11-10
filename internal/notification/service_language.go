package notification

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"text/template"
	"time"
)

// templateFS holds the e-mails of every language, templates/<language>/ (docs/i18n/SPEC.md §3 "Catalogs"):
// a template per kind (invite, booked, cancelled, reminder, signed) whose first line is the subject and whose body
// follows a blank line; date.txt, the start's template; and names.txt, the weekdays from Sunday and then the
// months, one per line, in the form the date uses.
//
//go:embed templates
var templateFS embed.FS

// fallbackLanguage writes an e-mail to a patient without a language of their own, and an invite from a clinic
// without a default (docs/i18n/SPEC.md §3 "The account's language fields").
const fallbackLanguage = "bs"

// letters is one language's e-mails.
type letters struct {
	templates *template.Template
	weekdays  []string // time.Weekday order
	months    []string // January first
}

// letter is what an e-mail's template is filled with: no id, no address, nothing clinical.
type letter struct {
	Clinic string
	// When is the appointment's start in the clinic's zone, written by the language's date.txt.
	When string
	Link string
	// Questionnaire adds COPY E1 (docs/questionnaire/COPY.md), never which questionnaire.
	Questionnaire bool
}

// loadLetters parses every language's templates. The templates are part of the binary, so a broken one is a
// programming error the tests catch: it panics.
func loadLetters() map[string]letters {
	dirs, err := fs.ReadDir(templateFS, "templates")
	if err != nil {
		panic(err)
	}
	all := make(map[string]letters, len(dirs))
	for _, d := range dirs {
		dir := "templates/" + d.Name()
		t := template.Must(template.New(d.Name()).Option("missingkey=error").ParseFS(templateFS, dir+"/*.txt"))
		names, err := fs.ReadFile(templateFS, dir+"/names.txt")
		if err != nil {
			panic(err)
		}
		lines := strings.Split(strings.TrimSuffix(string(names), "\n"), "\n")
		if len(lines) != 7+12 {
			panic(fmt.Sprintf("notification: %s/names.txt has %d names, want 19", dir, len(lines)))
		}
		all[d.Name()] = letters{templates: t, weekdays: lines[:7], months: lines[7:]}
	}
	return all
}

// in returns the e-mails of language, or of fallbackLanguage when there are none in it.
func (s *Service) in(language string) letters {
	if l, ok := s.letters[language]; ok {
		return l
	}
	return s.letters[fallbackLanguage]
}

// message fills the template kind with data, for to.
func (l letters) message(to, kind string, data letter) (Message, error) {
	var b strings.Builder
	if err := l.templates.ExecuteTemplate(&b, kind+".txt", data); err != nil {
		return Message{}, fmt.Errorf("the %s template: %w", kind, err)
	}
	subject, body, ok := strings.Cut(b.String(), "\n\n")
	if !ok {
		return Message{}, errors.New("the " + kind + " template has no subject line")
	}
	return Message{To: to, Subject: subject, Body: body}, nil
}

// when writes t, already in the clinic's zone, with the zone's name: 24-hour time and Latin digits in every
// language (docs/i18n/SPEC.md §3 "Formatting").
func (l letters) when(t time.Time, zone string) (string, error) {
	var b strings.Builder
	err := l.templates.ExecuteTemplate(&b, "date.txt", struct {
		Weekday, Month, Time, Zone string
		Day, Year                  int
	}{l.weekdays[t.Weekday()], l.months[t.Month()-1], t.Format("15:04"), zone, t.Day(), t.Year()})
	if err != nil {
		return "", fmt.Errorf("the date template: %w", err)
	}
	return strings.TrimSuffix(b.String(), "\n"), nil
}
