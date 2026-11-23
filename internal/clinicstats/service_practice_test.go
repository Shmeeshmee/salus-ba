package clinicstats_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/clinicstats"
	"github.com/Shmeeshmee/salus/internal/invoice"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/schedule"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// practiceSources are the online practice report's sources: clinic A's schedules, appointments and receipts.
type practiceSources struct {
	err       error
	schedules []schedule.Schedule
	visits    []appointment.Visit
	receipts  []invoice.Invoice
	asked     []string
	periods   []invoice.Period
}

func (p *practiceSources) Read(_ context.Context, c organization.Caller, id string) (organization.Organization, error) {
	if c.OrganizationID != id {
		return organization.Organization{}, organization.ErrNotFound
	}
	return organization.Organization{ID: id, TimeZone: "Europe/Sarajevo"}, nil
}

func (p *practiceSources) OfClinic(_ context.Context, org string) ([]schedule.Schedule, error) {
	p.asked = append(p.asked, "schedules "+org)
	return p.schedules, p.err
}

func (p *practiceSources) Visits(_ context.Context, org string, from, to time.Time) ([]appointment.Visit, error) {
	p.asked = append(p.asked, "visits "+org+" "+from.UTC().Format(time.RFC3339)+" "+to.UTC().Format(time.RFC3339))
	return p.visits, nil
}

func (p *practiceSources) List(_ context.Context, c invoice.Caller, period invoice.Period) ([]invoice.Invoice, error) {
	if !c.Admin || c.OrganizationID == "" {
		return nil, invoice.ErrForbidden
	}
	p.periods = append(p.periods, period)
	return p.receipts, nil
}

// Monday 2 to Sunday 8 March 2026, local days in Sarajevo (UTC+1).
var (
	weekFrom = time.Date(2026, 3, 2, 0, 0, 0, 0, sarajevo)
	weekTo   = time.Date(2026, 3, 9, 0, 0, 0, 0, sarajevo)
	reportAt = time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC)
)

const (
	visitVideo    = "e0000000-0000-4000-8000-000000000001"
	visitInPerson = "e0000000-0000-4000-8000-000000000002"
	visitNoShow   = "e0000000-0000-4000-8000-000000000003"
	visitOther    = "e0000000-0000-4000-8000-000000000004"
)

func newPractice() (*practiceSources, *fakeAuditor, *clinicstats.Service) {
	src := &practiceSources{
		schedules: []schedule.Schedule{
			{OrganizationID: clinicA, PractitionerRoleID: roleA1, Rules: []schedule.Rule{
				{Day: 1, Start: "08:00", End: "12:00", Mode: schedule.ModeInPerson}, // Monday mornings at the clinic
				{Day: 1, Start: "19:00", End: "21:30", Mode: schedule.ModeVideo},    // Monday evenings from home
				{Day: 3, Start: "20:00", End: "21:00"},                              // a rule from before the modes: video
			}},
			{OrganizationID: clinicA, PractitionerRoleID: roleA2, Rules: []schedule.Rule{{Day: 6, Start: "09:00", End: "10:00", Mode: schedule.ModeVideo}}},
		},
		visits: []appointment.Visit{
			{ID: visitVideo, PractitionerRoleID: roleA1, Mode: schedule.ModeVideo, Status: appointment.StatusFulfilled},
			{ID: visitInPerson, PractitionerRoleID: roleA1, Mode: schedule.ModeInPerson, Status: appointment.StatusFulfilled},
			{ID: visitNoShow, PractitionerRoleID: roleA1, Mode: schedule.ModeVideo, Status: appointment.StatusNoShow},
			{ID: visitOther, PractitionerRoleID: roleA2, Mode: schedule.ModeVideo, Status: appointment.StatusBooked},
		},
		receipts: []invoice.Invoice{
			{ID: "r1", AppointmentID: visitVideo, TotalMinor: 6000},
			{ID: "r2", AppointmentID: visitVideo, TotalMinor: 1500},
			{ID: "r3", AppointmentID: visitInPerson, TotalMinor: 4000, ReversedByID: "r4"}, // reversed: nets out
			{ID: "r4", AppointmentID: visitInPerson, TotalMinor: -4000, ReversesID: "r3"},
			{ID: "r5", AppointmentID: visitInPerson, TotalMinor: 3500},
			{ID: "r6", AppointmentID: visitNoShow, TotalMinor: 2000}, // not a visit held
			{ID: "r7", TotalMinor: 900},                              // a sale on its own
		},
	}
	audit := &fakeAuditor{}
	svc := clinicstats.NewService(clinicstats.Ports{
		Repo: &fakeRepo{}, Audit: audit, Tx: &fakeTx{}, Clock: clock.Fixed{T: reportAt},
		Clinics: src, Schedules: src, Visits: src, Receipts: src,
	})
	return src, audit, svc
}

func TestPractice_SumsHoursVisitsAndReceiptsPerPractitionerAndMode(t *testing.T) {
	src, audit, svc := newPractice()
	got, err := svc.Practice(t.Context(), admin, clinicstats.PracticeQuery{From: weekFrom, To: weekTo})
	want := []clinicstats.PracticeRow{
		{PractitionerRoleID: roleA1, Mode: schedule.ModeInPerson, MinutesOffered: 240, Visits: 1, ReceiptedMinor: 3500},
		{PractitionerRoleID: roleA1, Mode: schedule.ModeVideo, MinutesOffered: 150 + 60, Visits: 1, ReceiptedMinor: 7500},
		{PractitionerRoleID: roleA2, Mode: schedule.ModeVideo, MinutesOffered: 60},
	}
	if err != nil || !slices.Equal(got, want) {
		t.Fatalf("Practice() = %+v, %v;\nwant %+v", got, err, want)
	}
	if want := []string{"schedules " + clinicA, "visits " + clinicA + " 2026-03-01T23:00:00Z 2026-03-08T23:00:00Z"}; !slices.Equal(src.asked, want) {
		t.Errorf("asked %v, want %v", src.asked, want)
	}
	if want := []invoice.Period{{From: weekFrom, To: reportAt}}; !slices.EqualFunc(src.periods, want, func(a, b invoice.Period) bool { return a.From.Equal(b.From) && a.To.Equal(b.To) }) {
		t.Errorf("receipts of %v, want those issued from the period's start until now", src.periods)
	}
	if len(audit.events) != 1 || audit.events[0].Outcome != auditevent.OutcomeSuccess || audit.events[0].EntityReference != "Organization/"+clinicA {
		t.Errorf("audit %+v, want one successful read of the clinic", audit.events)
	}
}

// The hours offered are cut to the period: a period of one Monday evening hour holds an hour of video only.
func TestPractice_CutsTheHoursOfferedToThePeriod(t *testing.T) {
	src, _, svc := newPractice()
	src.visits, src.receipts = nil, nil
	from := time.Date(2026, 3, 2, 20, 0, 0, 0, sarajevo)
	got, err := svc.Practice(t.Context(), admin, clinicstats.PracticeQuery{From: from, To: from.Add(time.Hour)})
	want := []clinicstats.PracticeRow{{PractitionerRoleID: roleA1, Mode: schedule.ModeVideo, MinutesOffered: 60}}
	if err != nil || !slices.Equal(got, want) {
		t.Fatalf("Practice() = %+v, %v; want %+v", got, err, want)
	}
}

// A period that has not begun has no receipts to ask for.
func TestPractice_AsksForNoReceiptsBeforeThePeriodBegins(t *testing.T) {
	src, _, svc := newPractice()
	if _, err := svc.Practice(t.Context(), admin, clinicstats.PracticeQuery{From: reportAt.Add(time.Hour), To: reportAt.Add(48 * time.Hour)}); err != nil || len(src.periods) != 0 {
		t.Fatalf("Practice() = %v, receipts asked %v; want none asked", err, src.periods)
	}
}

func TestPractice_IsTheAdminsOnlyAndRefusesABadPeriod(t *testing.T) {
	for name, c := range map[string]clinicstats.Caller{
		"a practitioner": doctor,
		"a patient":      {AccountID: "acc-patient"},
	} {
		src, audit, svc := newPractice()
		_, err := svc.Practice(t.Context(), c, clinicstats.PracticeQuery{From: weekFrom, To: weekTo})
		if !errors.Is(err, clinicstats.ErrForbidden) || len(src.asked) != 0 || len(audit.events) != 1 || audit.events[0].Outcome != auditevent.OutcomeMinorFailure {
			t.Errorf("%s: Practice() = %v, asked %v, audit %+v; want ErrForbidden, nothing read, the refusal audited", name, err, src.asked, audit.events)
		}
	}
	for name, q := range map[string]clinicstats.PracticeQuery{
		"no start":       {To: weekTo},
		"reversed":       {From: weekTo, To: weekFrom},
		"empty":          {From: weekFrom, To: weekFrom},
		"too long a one": {From: weekFrom, To: weekFrom.AddDate(11, 0, 0)},
	} {
		_, audit, svc := newPractice()
		if _, err := svc.Practice(t.Context(), admin, q); !errors.Is(err, clinicstats.ErrInvalidQuery) || len(audit.events) != 0 {
			t.Errorf("%s: Practice() = %v, want ErrInvalidQuery, not audited", name, err)
		}
	}
}

func TestPractice_FailsWhenASourceFails(t *testing.T) {
	src, _, svc := newPractice()
	src.err = errors.New("synthetic failure")
	if _, err := svc.Practice(t.Context(), admin, clinicstats.PracticeQuery{From: weekFrom, To: weekTo}); err == nil {
		t.Fatal("Practice() error = nil, want the schedules' failure")
	}
}
