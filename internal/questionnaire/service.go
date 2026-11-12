package questionnaire

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// Transactor runs fn in a transaction carried by the context, joining one already there
// (postgres.Transactor).
type Transactor interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Auditor records an AuditEvent inside the caller's transaction (auditevent.Service).
type Auditor interface {
	Record(ctx context.Context, e auditevent.AuditEvent) error
}

// Roles reads a practitioner role of a clinic, unaudited inside the caller's own audited access
// (practitionerrole.Service).
type Roles interface {
	Lookup(ctx context.Context, org, id string) (practitionerrole.PractitionerRole, error)
}

// Clinics reads a clinic's default language, the original language of its new questionnaires
// (organization.Service).
type Clinics interface {
	DefaultLanguage(ctx context.Context, organizationID string) (string, error)
}

// Ports are the Service's dependencies.
type Ports struct {
	Repo    Repository
	Audit   Auditor
	Tx      Transactor
	Clock   clock.Clock
	Roles   Roles
	Clinics Clinics
}

// Service holds the questionnaire rules (docs/questionnaire/SPEC.md §2): staff of a clinic, practitioners
// and admins alike, build its questionnaires; only a draft is edited or deleted; publishing a draft retires
// its series' active version; an active version is revised into the next draft. A role's default is set by
// its practitioner or the clinic's admin. Anyone who is not staff gets ErrForbidden, staff of another clinic
// ErrNotFound. Every call is audited in its transaction, also when refused.
type Service struct {
	Ports
}

// NewService returns a Service on its ports.
func NewService(p Ports) *Service { return &Service{Ports: p} }

// op is what a call is audited as.
type op struct {
	name        string // in wrapped errors
	interaction string // AuditEvent.subtype
	resource    string // the entity's resource type
	action      auditevent.Action
}

var (
	opCreate       = op{"create", auditevent.CodeCreate, "Questionnaire", auditevent.ActionCreate}
	opUpdate       = op{"update draft", auditevent.CodeUpdate, "Questionnaire", auditevent.ActionUpdate}
	opDelete       = op{"delete draft", auditevent.CodeDelete, "Questionnaire", auditevent.ActionDelete}
	opPublish      = op{"publish", auditevent.CodeUpdate, "Questionnaire", auditevent.ActionUpdate}
	opRetire       = op{"retire", auditevent.CodeUpdate, "Questionnaire", auditevent.ActionUpdate}
	opRevise       = op{"revise", auditevent.CodeCreate, "Questionnaire", auditevent.ActionCreate}
	opRead         = op{"read", auditevent.CodeRead, "Questionnaire", auditevent.ActionRead}
	opSearch       = op{"search", auditevent.CodeSearchType, "Questionnaire", auditevent.ActionExecute}
	opDefault      = op{"default", auditevent.CodeRead, "PractitionerRole", auditevent.ActionRead}
	opSetDefault   = op{"set default", auditevent.CodeUpdate, "PractitionerRole", auditevent.ActionUpdate}
	opClearDefault = op{"clear default", auditevent.CodeDelete, "PractitionerRole", auditevent.ActionDelete}
)

// step is the work of one call inside its transaction, in the caller's clinic scope. It returns the ids of
// the entities the call disclosed or changed (one event each; none writes one event without an entity), a
// refusal (failed), which is audited and commits, or any other error, which rolls back.
type step func(ctx context.Context) (ids []string, failed, err error)

// do runs fn for c and audits it.
func (s *Service) do(ctx context.Context, c Caller, o op, fn step) error {
	if c.OrganizationID == "" {
		if err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
			return s.record(ctx, c, o, "", auditevent.OutcomeMinorFailure)
		}); err != nil {
			return fmt.Errorf("questionnaire: %s: %w", o.name, err)
		}
		return ErrForbidden
	}
	var failed error
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetOrganization(ctx, c.OrganizationID); err != nil {
			return err
		}
		ids, f, err := fn(ctx)
		if err != nil {
			return err
		}
		failed = f
		outcome := auditevent.OutcomeSuccess
		if failed != nil {
			outcome = auditevent.OutcomeMinorFailure
		}
		if len(ids) == 0 {
			return s.record(ctx, c, o, "", outcome)
		}
		for _, id := range ids {
			if err := s.record(ctx, c, o, id, outcome); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("questionnaire: %s: %w", o.name, err)
	}
	return failed
}

// record audits o by c on the entity id ("" for none). Staff act for healthcare operations, anyone else at
// their own request.
func (s *Service) record(ctx context.Context, c Caller, o op, id string, outcome auditevent.Outcome) error {
	return s.Audit.Record(ctx, entry(c, o, id, outcome))
}

// entry is the audit event of o by c on the entity id ("" for none).
func entry(c Caller, o op, id string, outcome auditevent.Outcome) auditevent.AuditEvent {
	purpose := auditevent.CodeHOPERAT
	if c.OrganizationID == "" {
		purpose = auditevent.CodePATRQT
	}
	e := auditevent.AuditEvent{
		Type:           auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:        auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: o.interaction},
		Purpose:        auditevent.Coding{System: auditevent.SystemActReason, Code: purpose},
		Action:         o.action,
		Outcome:        outcome,
		AgentAccountID: c.AccountID,
		OrganizationID: c.OrganizationID,
	}
	if id != "" {
		e.EntityReference = o.resource + "/" + id
	}
	return e
}

// version reads the version id of c's clinic, or returns ErrNotFound as failed. lock locks it until the
// transaction ends.
func (s *Service) version(ctx context.Context, c Caller, id string, lock bool) (q Questionnaire, failed, err error) {
	read := s.Repo.ByID
	if lock {
		read = s.Repo.Lock
	}
	q, err = read(ctx, id)
	switch {
	case errors.Is(err, ErrNotFound) || (err == nil && q.OrganizationID != c.OrganizationID): // not on row-level security alone
		return Questionnaire{}, ErrNotFound, nil
	case err != nil:
		return Questionnaire{}, nil, err
	}
	return q, nil, nil
}

// Create stores in as version 1, a draft, of a new questionnaire of c's clinic, in its language or else the
// clinic's default language, with its translations. An invalid one gives an *InvalidError.
func (s *Service) Create(ctx context.Context, c Caller, in Content) (Questionnaire, error) {
	var out Questionnaire
	err := s.do(ctx, c, opCreate, func(ctx context.Context) ([]string, error, error) {
		if err := validate(in, nil); err != nil {
			return nil, err, nil
		}
		lang := in.Language
		if lang == "" {
			var err error
			if lang, err = s.Clinics.DefaultLanguage(ctx, c.OrganizationID); err != nil {
				return nil, nil, err
			}
		}
		ts := freshen(nil, in.Items, nil, in.Translations)
		if err := validateTranslations(lang, in.Items, ts); err != nil {
			return nil, err, nil
		}
		var err error
		out, err = s.Repo.Create(ctx, Questionnaire{OrganizationID: c.OrganizationID, Title: in.Title, Description: in.Description, Items: in.Items, Language: lang}, s.Clock.Now())
		if err != nil {
			return nil, nil, err
		}
		out.Translations = ts
		if len(ts) > 0 {
			err = s.Repo.PutTranslations(ctx, out.ID, ts, s.Clock.Now())
		}
		return []string{out.ID}, nil, err
	})
	if err != nil {
		return Questionnaire{}, err
	}
	return out, nil
}

// UpdateDraft sets the content of the draft id at version, the version the caller last read: its language
// and translations when given, else the draft's (each translated text whose original changed is dropped,
// freshen). A published version gives ErrNotDraft, another version ErrConflict, an invalid content an
// *InvalidError (also a linkId an earlier version of the series gives another type).
func (s *Service) UpdateDraft(ctx context.Context, c Caller, id string, in Content, version int32) (Questionnaire, error) {
	var out Questionnaire
	err := s.do(ctx, c, opUpdate, func(ctx context.Context) ([]string, error, error) {
		cur, failed, err := s.version(ctx, c, id, false)
		switch {
		case err != nil || failed != nil:
			return nil, failed, err
		case cur.Status != StatusDraft:
			return []string{cur.ID}, ErrNotDraft, nil
		case cur.VersionID != version:
			return []string{cur.ID}, ErrConflict, nil
		}
		earlier, err := s.earlierKinds(ctx, cur)
		if err != nil {
			return nil, nil, err
		}
		if err := validate(in, earlier); err != nil {
			return []string{cur.ID}, err, nil
		}
		ts, err := s.translations(ctx, cur, in)
		if err != nil {
			return nil, nil, err
		}
		if in.Language != "" {
			cur.Language = in.Language
		}
		if err := validateTranslations(cur.Language, in.Items, ts); err != nil {
			return []string{cur.ID}, err, nil
		}
		cur.Title, cur.Description, cur.Items = in.Title, in.Description, in.Items
		out, err = s.Repo.UpdateDraft(ctx, cur, s.Clock.Now())
		if errors.Is(err, ErrConflict) { // changed since the read
			return []string{cur.ID}, ErrConflict, nil
		}
		if err != nil {
			return nil, nil, err
		}
		out.Translations = ts
		return []string{cur.ID}, nil, s.Repo.PutTranslations(ctx, cur.ID, ts, s.Clock.Now())
	})
	if err != nil {
		return Questionnaire{}, err
	}
	return out, nil
}

// translations is what the edit in of the draft cur stores as its translations: those sent, or else the
// stored ones, freshened against the draft's items.
func (s *Service) translations(ctx context.Context, cur Questionnaire, in Content) ([]Translation, error) {
	stored, err := s.Repo.Translations(ctx, cur.ID)
	if err != nil {
		return nil, err
	}
	ts := in.Translations
	if ts == nil {
		ts = stored
	}
	return freshen(cur.Items, in.Items, stored, ts), nil
}

// translate sets q's translations; complete keeps only the complete ones, which is all a patient reads.
func (s *Service) translate(ctx context.Context, q *Questionnaire, complete bool) error {
	ts, err := s.Repo.Translations(ctx, q.ID)
	if err != nil {
		return err
	}
	if complete {
		ts = slices.DeleteFunc(ts, func(t Translation) bool { return !isComplete(*q, t) })
	}
	q.Translations = ts
	return nil
}

// earlierKinds is the kind of every linkId of the other versions of q's series.
func (s *Service) earlierKinds(ctx context.Context, q Questionnaire) (map[string]string, error) {
	if q.Version == 1 {
		return nil, nil
	}
	versions, err := s.Repo.Search(ctx, Filter{SeriesID: q.SeriesID})
	if err != nil {
		return nil, err
	}
	kinds := map[string]string{}
	for _, v := range versions {
		if v.ID != q.ID && v.OrganizationID == q.OrganizationID {
			addKinds(kinds, v.Items)
		}
	}
	return kinds, nil
}

// DeleteDraft deletes the draft id. A published version gives ErrNotDraft.
func (s *Service) DeleteDraft(ctx context.Context, c Caller, id string) error {
	return s.do(ctx, c, opDelete, func(ctx context.Context) ([]string, error, error) {
		cur, failed, err := s.version(ctx, c, id, true)
		switch {
		case err != nil || failed != nil:
			return nil, failed, err
		case cur.Status != StatusDraft:
			return []string{cur.ID}, ErrNotDraft, nil
		}
		if err := s.Repo.DeleteDraft(ctx, cur.ID); err != nil {
			return nil, nil, err
		}
		return []string{cur.ID}, nil, nil
	})
}

// Read returns the version id of c's clinic, or, to a patient, a version their own responses reference
// (Referenced); to a practitioner, also another clinic's version a completed response of a visit shared with their
// clinic answers (the history, ADR 0014), without its translations.
func (s *Service) Read(ctx context.Context, c Caller, id string) (Questionnaire, error) {
	if patientCaller(c) {
		return s.Referenced(ctx, c, c.PatientID, id)
	}
	var out Questionnaire
	err := s.do(ctx, c, opRead, func(ctx context.Context) ([]string, error, error) {
		var failed, err error
		out, failed, err = s.version(ctx, c, id, false)
		if errors.Is(failed, ErrNotFound) && !c.Admin {
			return s.shared(ctx, id, &out)
		}
		if err != nil || failed != nil {
			return nil, failed, err
		}
		return []string{out.ID}, nil, s.translate(ctx, &out, false)
	})
	if err != nil {
		return Questionnaire{}, err
	}
	return out, nil
}

// shared reads into out another clinic's version id that a completed response of a visit shared with the clinic
// in scope answers (the history, ADR 0014), without its translations: for that clinic's practitioner to read the
// answers. ErrNotFound as the refusal for any other.
func (s *Service) shared(ctx context.Context, id string, out *Questionnaire) ([]string, error, error) {
	q, err := s.Repo.Shared(ctx, id)
	switch {
	case errors.Is(err, ErrNotFound):
		return nil, ErrNotFound, nil
	case err != nil:
		return nil, nil, err
	}
	*out = q
	return []string{q.ID}, nil, nil
}

// Referenced returns the version id to the patient patientID, c their account, read in the patient's scope,
// where row-level security shows a patient only the versions their own questionnaire responses reference. It
// serves the questionnaireresponse service, which has checked that the patient's response references it.
// Audited as the patient's read; ErrNotFound for any other version (and without a patient), audited without
// the entity.
func (s *Service) Referenced(ctx context.Context, c Caller, patientID, id string) (Questionnaire, error) {
	var out Questionnaire
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if patientID == "" {
			return s.record(ctx, c, opRead, "", auditevent.OutcomeMinorFailure)
		}
		if err := s.Repo.SetPatient(ctx, patientID); err != nil {
			return err
		}
		q, err := s.Repo.ByID(ctx, id)
		switch {
		case errors.Is(err, ErrNotFound):
			return s.record(ctx, c, opRead, "", auditevent.OutcomeMinorFailure)
		case err != nil:
			return err
		}
		if err := s.translate(ctx, &q, true); err != nil {
			return err
		}
		out = q
		return s.record(ctx, c, opRead, q.ID, auditevent.OutcomeSuccess)
	})
	switch {
	case err != nil:
		return Questionnaire{}, fmt.Errorf("questionnaire: referenced: %w", err)
	case out.ID == "":
		return Questionnaire{}, ErrNotFound
	}
	return out, nil
}

// Search returns the versions of c's clinic the filter gives, oldest first, or, to a patient, those of the
// versions their own responses reference (row-level security of the patient's scope). Each one returned is
// audited; a search that returns none writes one event without an entity.
func (s *Service) Search(ctx context.Context, c Caller, f Filter) ([]Questionnaire, error) {
	if patientCaller(c) {
		return s.searchReferenced(ctx, c, f)
	}
	var out []Questionnaire
	err := s.do(ctx, c, opSearch, func(ctx context.Context) ([]string, error, error) {
		all, err := s.Repo.Search(ctx, f)
		if err != nil {
			return nil, nil, err
		}
		ids := make([]string, 0, len(all))
		for _, q := range all {
			if q.OrganizationID == c.OrganizationID { // not on row-level security alone
				if err := s.translate(ctx, &q, false); err != nil {
					return nil, nil, err
				}
				out = append(out, q)
				ids = append(ids, q.ID)
			}
		}
		return ids, nil, nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// patientCaller tells whether c is a patient with a profile, who reads the versions their responses reference.
func patientCaller(c Caller) bool { return c.OrganizationID == "" && c.PatientID != "" }

// searchReferenced is Search in the scope of the patient c.
func (s *Service) searchReferenced(ctx context.Context, c Caller, f Filter) ([]Questionnaire, error) {
	var out []Questionnaire
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetPatient(ctx, c.PatientID); err != nil {
			return err
		}
		var err error
		if out, err = s.Repo.Search(ctx, f); err != nil {
			return err
		}
		if len(out) == 0 {
			return s.record(ctx, c, opSearch, "", auditevent.OutcomeSuccess)
		}
		for i := range out {
			if err := s.translate(ctx, &out[i], true); err != nil {
				return err
			}
			if err := s.record(ctx, c, opSearch, out[i].ID, auditevent.OutcomeSuccess); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("questionnaire: search: %w", err)
	}
	return out, nil
}
