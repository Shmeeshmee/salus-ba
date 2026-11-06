package questionnaire_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
	"github.com/Shmeeshmee/salus/internal/questionnaire"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

type txKey struct{}

// fakeTransactor runs fn in a "transaction" and rolls back the fake repository's rows when fn fails.
type fakeTransactor struct{ repo *fakeRepository }

func (t fakeTransactor) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if ctx.Value(txKey{}) != nil {
		return fn(ctx)
	}
	saved := t.repo.snapshot()
	if err := fn(context.WithValue(ctx, txKey{}, true)); err != nil {
		t.repo.restore(saved)
		return err
	}
	return nil
}

// fakeRepository holds questionnaires and defaults and, unless leak is set, shows only those of the clinic
// set, as row-level security does. Calls outside a transaction are recorded in outside.
type fakeRepository struct {
	err        error
	referenced map[string][]string // the versions each patient's responses reference
	clinic     string
	patient    string
	rows       []questionnaire.Questionnaire
	defaults   []questionnaire.Default
	outside    []string
	next       int
	leak       bool
}

type fakeState struct {
	rows     []questionnaire.Questionnaire
	defaults []questionnaire.Default
}

func (r *fakeRepository) snapshot() fakeState {
	return fakeState{rows: slices.Clone(r.rows), defaults: slices.Clone(r.defaults)}
}

func (r *fakeRepository) restore(s fakeState) { r.rows, r.defaults = s.rows, s.defaults }

func (r *fakeRepository) seen(ctx context.Context, call string) {
	if ctx.Value(txKey{}) == nil {
		r.outside = append(r.outside, call)
	}
}

func (r *fakeRepository) visible(org string) bool { return r.leak || org == r.clinic }

// shows tells whether the scope shows the version q: the clinic's own, or one the patient's responses
// reference.
func (r *fakeRepository) shows(q questionnaire.Questionnaire) bool {
	return r.visible(q.OrganizationID) || (r.patient != "" && slices.Contains(r.referenced[r.patient], q.ID))
}

func (r *fakeRepository) SetOrganization(ctx context.Context, id string) error {
	r.seen(ctx, "SetOrganization")
	r.clinic = id
	return nil
}

func (r *fakeRepository) SetPatient(ctx context.Context, id string) error {
	r.seen(ctx, "SetPatient")
	r.patient = id
	return nil
}

func (r *fakeRepository) index(id string) int {
	return slices.IndexFunc(r.rows, func(q questionnaire.Questionnaire) bool { return q.ID == id && r.shows(q) })
}

func (r *fakeRepository) ByID(ctx context.Context, id string) (questionnaire.Questionnaire, error) {
	r.seen(ctx, "ByID")
	if r.err != nil {
		return questionnaire.Questionnaire{}, r.err
	}
	if i := r.index(id); i >= 0 {
		return r.rows[i], nil
	}
	return questionnaire.Questionnaire{}, questionnaire.ErrNotFound
}

func (r *fakeRepository) Lock(ctx context.Context, id string) (questionnaire.Questionnaire, error) {
	return r.ByID(ctx, id)
}

func (r *fakeRepository) Search(ctx context.Context, f questionnaire.Filter) ([]questionnaire.Questionnaire, error) {
	r.seen(ctx, "Search")
	var out []questionnaire.Questionnaire
	for _, q := range r.rows {
		if r.shows(q) && (f.SeriesID == "" || q.SeriesID == f.SeriesID) &&
			(f.Status == "" || q.Status == f.Status) && (f.Version == 0 || q.Version == f.Version) {
			out = append(out, q)
		}
	}
	return out, r.err
}

func (r *fakeRepository) newID() string {
	r.next++
	return fmt.Sprintf("a8000000-0000-4000-8000-%012d", r.next)
}

func (r *fakeRepository) Create(ctx context.Context, q questionnaire.Questionnaire, at time.Time) (questionnaire.Questionnaire, error) {
	r.seen(ctx, "Create")
	q.ID, q.SeriesID, q.Version, q.VersionID, q.Status, q.LastUpdated = r.newID(), r.newID(), 1, 1, questionnaire.StatusDraft, at
	r.rows = append(r.rows, q)
	return q, nil
}

func (r *fakeRepository) CreateVersion(ctx context.Context, from questionnaire.Questionnaire, at time.Time) (questionnaire.Questionnaire, error) {
	r.seen(ctx, "CreateVersion")
	next := int32(0)
	for _, q := range r.rows {
		if q.SeriesID == from.SeriesID {
			if q.Status == questionnaire.StatusDraft {
				return questionnaire.Questionnaire{}, questionnaire.ErrDraftExists
			}
			next = max(next, q.Version)
		}
	}
	q := questionnaire.Questionnaire{
		ID: r.newID(), SeriesID: from.SeriesID, OrganizationID: from.OrganizationID, Version: next + 1, VersionID: 1,
		Status: questionnaire.StatusDraft, Title: from.Title, Description: from.Description, Items: from.Items, LastUpdated: at,
	}
	r.rows = append(r.rows, q)
	return q, nil
}

func (r *fakeRepository) UpdateDraft(ctx context.Context, q questionnaire.Questionnaire, at time.Time) (questionnaire.Questionnaire, error) {
	r.seen(ctx, "UpdateDraft")
	i := r.index(q.ID)
	if i < 0 || r.rows[i].Status != questionnaire.StatusDraft || r.rows[i].VersionID != q.VersionID {
		return questionnaire.Questionnaire{}, questionnaire.ErrConflict
	}
	cur := &r.rows[i]
	cur.Title, cur.Description, cur.Items, cur.VersionID, cur.LastUpdated = q.Title, q.Description, q.Items, cur.VersionID+1, at
	return *cur, nil
}

func (r *fakeRepository) DeleteDraft(ctx context.Context, id string) error {
	r.seen(ctx, "DeleteDraft")
	i := r.index(id)
	if i < 0 || r.rows[i].Status != questionnaire.StatusDraft {
		return questionnaire.ErrNotFound
	}
	r.rows = slices.Delete(r.rows, i, i+1)
	return nil
}

func (r *fakeRepository) setStatus(id string, from, to questionnaire.Status, at time.Time) (questionnaire.Questionnaire, error) {
	i := r.index(id)
	if i < 0 || r.rows[i].Status != from {
		return questionnaire.Questionnaire{}, questionnaire.ErrConflict
	}
	cur := &r.rows[i]
	cur.Status, cur.VersionID, cur.LastUpdated = to, cur.VersionID+1, at
	if to == questionnaire.StatusActive {
		cur.PublishedAt = at
	} else {
		cur.RetiredAt = at
	}
	return *cur, nil
}

func (r *fakeRepository) Publish(ctx context.Context, id string, at time.Time) (questionnaire.Questionnaire, error) {
	r.seen(ctx, "Publish")
	for _, q := range r.rows {
		if q.Status == questionnaire.StatusActive && r.index(id) >= 0 && q.SeriesID == r.rows[r.index(id)].SeriesID {
			return questionnaire.Questionnaire{}, errors.New("fake: a second active version")
		}
	}
	return r.setStatus(id, questionnaire.StatusDraft, questionnaire.StatusActive, at)
}

func (r *fakeRepository) Retire(ctx context.Context, id string, at time.Time) (questionnaire.Questionnaire, error) {
	r.seen(ctx, "Retire")
	return r.setStatus(id, questionnaire.StatusActive, questionnaire.StatusRetired, at)
}

func (r *fakeRepository) RetireActive(ctx context.Context, seriesID string, at time.Time) error {
	r.seen(ctx, "RetireActive")
	for _, q := range r.rows {
		if q.SeriesID == seriesID && q.Status == questionnaire.StatusActive && r.visible(q.OrganizationID) {
			if _, err := r.setStatus(q.ID, questionnaire.StatusActive, questionnaire.StatusRetired, at); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *fakeRepository) Default(ctx context.Context, roleID string) (questionnaire.Default, error) {
	r.seen(ctx, "Default")
	for _, d := range r.defaults {
		if d.PractitionerRoleID == roleID && r.visible(d.OrganizationID) {
			return d, nil
		}
	}
	return questionnaire.Default{}, questionnaire.ErrNotFound
}

func (r *fakeRepository) SetDefault(ctx context.Context, d questionnaire.Default, at time.Time) (questionnaire.Default, error) {
	r.seen(ctx, "SetDefault")
	d.LastUpdated, d.VersionID = at, 1
	for i, cur := range r.defaults {
		if cur.PractitionerRoleID == d.PractitionerRoleID {
			d.VersionID = cur.VersionID + 1
			r.defaults[i] = d
			return d, nil
		}
	}
	r.defaults = append(r.defaults, d)
	return d, nil
}

func (r *fakeRepository) ClearDefault(ctx context.Context, roleID string) error {
	r.seen(ctx, "ClearDefault")
	r.defaults = slices.DeleteFunc(r.defaults, func(d questionnaire.Default) bool {
		return d.PractitionerRoleID == roleID && r.visible(d.OrganizationID)
	})
	return nil
}

// fakeAuditor records events, or fails with err.
type fakeAuditor struct {
	err    error
	events []auditevent.AuditEvent
}

func (a *fakeAuditor) Record(ctx context.Context, e auditevent.AuditEvent) error {
	if ctx.Value(txKey{}) == nil {
		return errors.New("fake: audit outside a transaction")
	}
	if a.err != nil {
		return a.err
	}
	a.events = append(a.events, e)
	return nil
}

// fakeRoles holds the roles of every clinic and shows one only in its own clinic, as practitionerrole's
// Lookup does.
type fakeRoles struct {
	roles []practitionerrole.PractitionerRole
}

func (f fakeRoles) Lookup(_ context.Context, org, id string) (practitionerrole.PractitionerRole, error) {
	for _, r := range f.roles {
		if r.ID == id && r.OrganizationID == org {
			return r, nil
		}
	}
	return practitionerrole.PractitionerRole{}, practitionerrole.ErrNotFound
}

type fixture struct {
	svc   *questionnaire.Service
	repo  *fakeRepository
	audit *fakeAuditor
}

// newFixture gives clinic A series A (active version 1, draft version 2) and clinic B an active series B,
// and the roles A1 (practitioner 1), A2 (practitioner 2) and B.
func newFixture() fixture {
	repo := &fakeRepository{rows: []questionnaire.Questionnaire{
		{ID: activeA, SeriesID: seriesA, OrganizationID: clinicA, Version: 1, VersionID: 2, Status: questionnaire.StatusActive, Title: placeholderTitle, Items: items(), PublishedAt: now.Add(-time.Hour)},
		{ID: draftA, SeriesID: seriesA, OrganizationID: clinicA, Version: 2, VersionID: 1, Status: questionnaire.StatusDraft, Title: placeholderTitle, Items: items()},
		{ID: activeB, SeriesID: seriesB, OrganizationID: clinicB, Version: 1, VersionID: 2, Status: questionnaire.StatusActive, Title: placeholderTitle, Items: items(), PublishedAt: now.Add(-time.Hour)},
	}}
	audit := &fakeAuditor{}
	roles := fakeRoles{roles: []practitionerrole.PractitionerRole{
		{ID: roleA1, OrganizationID: clinicA, PractitionerID: practitioner1},
		{ID: roleA2, OrganizationID: clinicA, PractitionerID: practitioner2},
		{ID: roleB, OrganizationID: clinicB, PractitionerID: "b3000000-0000-4000-8000-0000000000b1"},
	}}
	svc := questionnaire.NewService(questionnaire.Ports{
		Repo: repo, Audit: audit, Tx: fakeTransactor{repo: repo}, Clock: clock.Fixed{T: now}, Roles: roles,
	})
	return fixture{svc: svc, repo: repo, audit: audit}
}

func (f fixture) row(id string) questionnaire.Questionnaire {
	for _, q := range f.repo.rows {
		if q.ID == id {
			return q
		}
	}
	return questionnaire.Questionnaire{}
}

// event is the part of an AuditEvent the tests compare.
type event struct {
	agent, job, org, subtype, purpose, entity string
	action                                    auditevent.Action
	outcome                                   auditevent.Outcome
}

func events(a *fakeAuditor) []event {
	out := make([]event, 0, len(a.events))
	for _, e := range a.events {
		if e.Type != (auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest}) ||
			e.Subtype.System != auditevent.SystemRestfulInteraction || e.Purpose.System != auditevent.SystemActReason {
			out = append(out, event{entity: "unexpected codings"})
			continue
		}
		out = append(out, event{
			agent: e.AgentAccountID, job: e.AgentName, org: e.OrganizationID, subtype: e.Subtype.Code, purpose: e.Purpose.Code,
			entity: e.EntityReference, action: e.Action, outcome: e.Outcome,
		})
	}
	return out
}

func staffEvent(c questionnaire.Caller, subtype string, action auditevent.Action, entity string, outcome auditevent.Outcome) event {
	return event{agent: c.AccountID, org: c.OrganizationID, subtype: subtype, purpose: auditevent.CodeHOPERAT, entity: entity, action: action, outcome: outcome}
}

func assertEvents(t *testing.T, a *fakeAuditor, want ...event) {
	t.Helper()
	if got := events(a); !slices.Equal(got, want) {
		t.Errorf("audit events:\n got %+v\nwant %+v", got, want)
	}
}

// assertNoContent fails when err's text holds a title, an item text or an option.
func assertNoContent(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}
	for _, s := range []string{placeholderTitle, placeholderText, "Placeholder choice", "One", "Two"} {
		if strings.Contains(err.Error(), s) {
			t.Errorf("error %q holds questionnaire content %q", err, s)
		}
	}
}
