package history

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/binary"
	"github.com/Shmeeshmee/salus/internal/composition"
	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
	"github.com/Shmeeshmee/salus/internal/questionnaireresponse"
	"github.com/Shmeeshmee/salus/pkg/keyset"
)

// The sources of the patient's timeline (SAL-39). Each is the owning domain's service, which reads the calling
// patient's own items in their scope after the page's position, applies its access rule and audits every item it
// returns. A chat's messages (TypeCommunication) will be one more source, through internal/communication.

// Visits gives the patient's encounters (encounter.Service).
type Visits interface {
	PatientHistory(ctx context.Context, c appointment.Caller, org string, p keyset.Page) ([]encounter.Encounter, error)
}

// Documents gives the patient's signed documents, never a draft (composition.Service).
type Documents interface {
	PatientHistory(ctx context.Context, c appointment.Caller, org string, p keyset.Page) ([]composition.Document, error)
}

// Responses gives the patient's completed questionnaire responses, and the files of them that are no longer
// pending (questionnaireresponse.Service).
type Responses interface {
	PatientHistory(ctx context.Context, c appointment.Caller, org string, p keyset.Page) ([]questionnaireresponse.Response, error)
	PatientAttachments(ctx context.Context, c appointment.Caller, org string, p keyset.Page) ([]questionnaireresponse.Attached, error)
}

// Files gives, of the files a page lists, those that are the patient's own and clean (binary.Service).
type Files interface {
	Clean(ctx context.Context, c binary.Caller, ids []string) ([]string, error)
}

// Clinics gives the names of the clinics of the patient's items (organization.Service), unaudited: a clinic holds
// no personal data.
type Clinics interface {
	Names(ctx context.Context, ids []string) (map[string]string, error)
}

// Roles gives a clinic's practitioner role, for its specialty (practitionerrole.Service), unaudited.
type Roles interface {
	Lookup(ctx context.Context, org, id string) (practitionerrole.PractitionerRole, error)
}

// MaxLimit is the largest page of a history.
const MaxLimit = 100

// entry is an item of the timeline with the practitioner role it was with, for its specialty.
type entry struct {
	role string
	Item
}

// Timeline returns one page of the calling patient's own history from every clinic, newest first (the reference
// breaking a tie), strictly after the cursor, as the filter narrows it: their visits, signed documents, completed
// questionnaire answers and the clean files of those (ADR 0014: the patient sees every visit of theirs, never a
// draft). Each source is asked for one more item than the page, so Next is set exactly when there is a later
// page. Every item is read and audited by its owning domain. Staff get ErrForbidden; a limit outside 1 to
// MaxLimit is ErrLimit; neither reads anything.
func (s *Service) Timeline(ctx context.Context, c appointment.Caller, f Filter, cur Cursor, limit int) (Page, error) {
	switch {
	case c.OrganizationID != "":
		return Page{}, ErrForbidden
	case limit < 1 || limit > MaxLimit:
		return Page{}, ErrLimit
	}
	p := keyset.Page{At: cur.Date, Ref: cur.Reference, From: f.From, To: f.To, Limit: limit + 1}
	all, err := s.sources(ctx, c, f, p)
	if err != nil {
		return Page{}, fmt.Errorf("history: timeline: %w", err)
	}
	all = slices.DeleteFunc(all, func(e entry) bool { return !cur.Before(e.Item) }) // not on each source alone
	slices.SortFunc(all, func(a, b entry) int {
		if c := b.Date.Compare(a.Date); c != 0 {
			return c
		}
		return strings.Compare(b.Reference, a.Reference)
	})
	var out Page
	if len(all) > limit {
		all = all[:limit]
		out.Next = CursorAt(all[limit-1].Item).String()
	}
	if all, err = s.clean(ctx, c, all); err != nil {
		return Page{}, fmt.Errorf("history: timeline: %w", err)
	}
	if out.Items, err = s.items(ctx, all); err != nil {
		return Page{}, fmt.Errorf("history: timeline: %w", err)
	}
	return out, nil
}

// source reads one source's part of a page.
type source func(ctx context.Context, c appointment.Caller, org string, p keyset.Page) ([]entry, error)

// sources asks every source the filter's types name for the page, all at once: each is a transaction of its own
// in its domain, so the page waits for the slowest of them, not for their sum (PROGRESS.md, SAL-39).
func (s *Service) sources(ctx context.Context, c appointment.Caller, f Filter, p keyset.Page) ([]entry, error) {
	var asked []source
	for _, src := range []struct {
		read source
		typ  ItemType
	}{
		{s.visits, TypeVisit}, {s.documents, TypeDocument}, {s.responses, TypeQuestionnaireResponse}, {s.attachments, TypeAttachment},
	} {
		if len(f.Types) == 0 || slices.Contains(f.Types, src.typ) {
			asked = append(asked, src.read)
		}
	}
	parts, errs := make([][]entry, len(asked)), make([]error, len(asked))
	var wg sync.WaitGroup
	for i, read := range asked {
		wg.Go(func() { parts[i], errs[i] = read(ctx, c, f.OrganizationID, p) })
	}
	wg.Wait()
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	return slices.Concat(parts...), nil
}

func (s *Service) visits(ctx context.Context, c appointment.Caller, org string, p keyset.Page) ([]entry, error) {
	visits, err := s.Visits.PatientHistory(ctx, c, org, p)
	out := make([]entry, len(visits))
	for i, e := range visits {
		out[i] = entry{Item: Item{Date: e.PeriodStart, Clinic: Clinic{ID: e.OrganizationID}, Type: TypeVisit,
			Reference: "Encounter/" + e.ID, EncounterID: e.ID}, role: e.PractitionerRoleID}
	}
	return out, err
}

func (s *Service) documents(ctx context.Context, c appointment.Caller, org string, p keyset.Page) ([]entry, error) {
	docs, err := s.Documents.PatientHistory(ctx, c, org, p)
	out := make([]entry, len(docs))
	for i, d := range docs {
		out[i] = entry{Item: Item{Date: d.Draft.SignedAt, Clinic: Clinic{ID: d.Encounter.OrganizationID}, Type: TypeDocument,
			Reference: "Composition/" + d.Draft.ID, EncounterID: d.Encounter.ID, Title: d.Draft.Title}, role: d.Encounter.PractitionerRoleID}
	}
	return out, err
}

func (s *Service) responses(ctx context.Context, c appointment.Caller, org string, p keyset.Page) ([]entry, error) {
	rs, err := s.Responses.PatientHistory(ctx, c, org, p)
	out := make([]entry, len(rs))
	for i, r := range rs {
		out[i] = entry{Item: Item{Date: r.AuthoredAt, Clinic: Clinic{ID: r.OrganizationID}, Type: TypeQuestionnaireResponse,
			Reference: "QuestionnaireResponse/" + r.ID}, role: r.PractitionerRoleID}
	}
	return out, err
}

func (s *Service) attachments(ctx context.Context, c appointment.Caller, org string, p keyset.Page) ([]entry, error) {
	files, err := s.Responses.PatientAttachments(ctx, c, org, p)
	out := make([]entry, len(files))
	for i, a := range files {
		r := a.Response
		out[i] = entry{Item: Item{Date: r.AuthoredAt, Clinic: Clinic{ID: r.OrganizationID}, Type: TypeAttachment,
			Reference: "Binary/" + a.File.BinaryID, Title: a.File.Title}, role: r.PractitionerRoleID}
	}
	return out, err
}

// clean leaves out the files of the page that binary does not find the patient's own and clean.
func (s *Service) clean(ctx context.Context, c appointment.Caller, page []entry) ([]entry, error) {
	var ids []string
	for _, e := range page {
		if e.Type == TypeAttachment {
			ids = append(ids, strings.TrimPrefix(e.Reference, "Binary/"))
		}
	}
	if len(ids) == 0 {
		return page, nil
	}
	clean, err := s.Files.Clean(ctx, binary.Caller{AccountID: c.AccountID, PatientID: c.PatientID}, ids)
	if err != nil {
		return nil, err
	}
	return slices.DeleteFunc(page, func(e entry) bool {
		return e.Type == TypeAttachment && !slices.Contains(clean, strings.TrimPrefix(e.Reference, "Binary/"))
	}), nil
}

// items gives the page's entries their clinic's name and their role's specialty, each clinic and role read once.
func (s *Service) items(ctx context.Context, page []entry) ([]Item, error) {
	out := make([]Item, len(page))
	var orgs []string
	for _, e := range page {
		if !slices.Contains(orgs, e.Clinic.ID) {
			orgs = append(orgs, e.Clinic.ID)
		}
	}
	if len(orgs) == 0 {
		return out, nil
	}
	names, err := s.Clinics.Names(ctx, orgs)
	if err != nil {
		return nil, err
	}
	specialties := map[string]string{}
	for i, e := range page {
		key := e.Clinic.ID + "/" + e.role
		specialty, known := specialties[key]
		if !known && e.role != "" {
			r, err := s.Roles.Lookup(ctx, e.Clinic.ID, e.role)
			if err != nil && !errors.Is(err, practitionerrole.ErrNotFound) {
				return nil, err
			}
			specialty, specialties[key] = r.SpecialtyCode, r.SpecialtyCode
		}
		out[i] = e.Item
		out[i].Clinic.Name, out[i].Specialty = names[e.Clinic.ID], specialty
	}
	return out, nil
}
