package chargeitemdefinition_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/chargeitemdefinition"
	"github.com/Shmeeshmee/salus/internal/schedule"
)

func ptr(v int32) *int32 { return &v }

// exam is a valid service as an admin sends it. The names are placeholders.
func exam() chargeitemdefinition.Definition {
	return chargeitemdefinition.Definition{
		Name: "Placeholder exam", Code: "EX-1", PriceMinor: 5000, DurationMinutes: 30,
		Modes: []schedule.VisitMode{schedule.ModeInPerson, schedule.ModeVideo}, Active: true,
		Translations: []chargeitemdefinition.Translation{{Language: "de", Name: "Platzhalter Untersuchung"}},
	}
}

func TestCreate_StoresTheAdminsServiceInTheirClinicAudited(t *testing.T) {
	repo, audit := &fakeRepository{}, &fakeAuditor{}
	got, err := newService(repo, audit).Create(t.Context(), admin, exam())
	if err != nil {
		t.Fatal(err)
	}
	if got.ID == "" || got.OrganizationID != clinicA || got.Currency != "BAM" || got.VersionID != 1 || !repo.writtenAt.Equal(now) {
		t.Errorf("Create() = %+v at %v; want an id, clinic A, BAM, version 1, at the clock's time", got, repo.writtenAt)
	}
	if repo.outside {
		t.Error("the repository was called outside a transaction")
	}
	want := auditevent.AuditEvent{
		Type:    auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype: auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: auditevent.CodeCreate},
		Purpose: auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeHOPERAT},
		Action:  auditevent.ActionCreate, Outcome: auditevent.OutcomeSuccess, AgentAccountID: adminA, OrganizationID: clinicA,
		EntityReference: "ChargeItemDefinition/" + got.ID,
	}
	if len(audit.events) != 1 || audit.events[0] != want {
		t.Errorf("audit = %+v, want %+v", audit.events, want)
	}
}

func TestCreateAndUpdate_RefuseAnyoneButTheClinicsAdminAudited(t *testing.T) {
	for name, tc := range map[string]struct {
		err    error
		caller chargeitemdefinition.Caller
	}{
		"other staff": {chargeitemdefinition.ErrForbidden, staff},
		"no clinic":   {chargeitemdefinition.ErrNotFound, noClinic},
	} {
		repo, audit := &fakeRepository{}, &fakeAuditor{}
		svc := newService(repo, audit)
		_, create := svc.Create(t.Context(), tc.caller, exam())
		_, update := svc.Update(t.Context(), tc.caller, exam())
		if !errors.Is(create, tc.err) || !errors.Is(update, tc.err) || len(repo.rows) != 0 {
			t.Errorf("%s: Create() = %v, Update() = %v, %d stored; want %v and nothing stored", name, create, update, len(repo.rows), tc.err)
		}
		if len(audit.events) != 2 || audit.events[0].Outcome != auditevent.OutcomeMinorFailure || audit.events[0].EntityReference != "" ||
			audit.events[1].Outcome != auditevent.OutcomeMinorFailure {
			t.Errorf("%s: audit = %+v, want two refusals", name, audit.events)
		}
	}
}

func TestCreate_RefusesAnInvalidService(t *testing.T) {
	for name, change := range map[string]func(d *chargeitemdefinition.Definition){
		"blank name":            func(d *chargeitemdefinition.Definition) { d.Name = "  " },
		"long name":             func(d *chargeitemdefinition.Definition) { d.Name = strings.Repeat("a", 201) },
		"long code":             func(d *chargeitemdefinition.Definition) { d.Code = strings.Repeat("a", 41) },
		"negative price":        func(d *chargeitemdefinition.Definition) { d.PriceMinor = -1 },
		"price over the limit":  func(d *chargeitemdefinition.Definition) { d.PriceMinor = 1_000_000_001 },
		"unknown currency":      func(d *chargeitemdefinition.Definition) { d.Currency = "EUR" },
		"short duration":        func(d *chargeitemdefinition.Definition) { d.DurationMinutes = 4 },
		"long duration":         func(d *chargeitemdefinition.Definition) { d.DurationMinutes = 481 },
		"negative VAT":          func(d *chargeitemdefinition.Definition) { d.VATRateBP = ptr(-1) },
		"VAT over 100 %":        func(d *chargeitemdefinition.Definition) { d.VATRateBP = ptr(10001) },
		"no mode":               func(d *chargeitemdefinition.Definition) { d.Modes = nil },
		"unknown mode":          func(d *chargeitemdefinition.Definition) { d.Modes = []schedule.VisitMode{"phone"} },
		"a mode twice":          func(d *chargeitemdefinition.Definition) { d.Modes = []schedule.VisitMode{"video", "video"} },
		"unknown language":      func(d *chargeitemdefinition.Definition) { d.Translations[0].Language = "xx" },
		"blank translation":     func(d *chargeitemdefinition.Definition) { d.Translations[0].Name = " " },
		"a language twice":      func(d *chargeitemdefinition.Definition) { d.Translations = append(d.Translations, d.Translations[0]) },
		"long translation name": func(d *chargeitemdefinition.Definition) { d.Translations[0].Name = strings.Repeat("a", 201) },
	} {
		repo := &fakeRepository{}
		d := exam()
		change(&d)
		if _, err := newService(repo, &fakeAuditor{}).Create(t.Context(), admin, d); !errors.Is(err, chargeitemdefinition.ErrInvalid) || len(repo.rows) != 0 {
			t.Errorf("%s: Create() = %v with %d stored, want ErrInvalid and nothing stored", name, err, len(repo.rows))
		}
	}
}

func TestCreate_TakesAnExemptServiceWithTheSlotsLengthAndATaxableOne(t *testing.T) {
	svc := newService(&fakeRepository{}, &fakeAuditor{})
	exempt := exam()
	exempt.DurationMinutes, exempt.Code, exempt.Translations, exempt.PriceMinor = 0, "", nil, 0
	taxable := exam()
	taxable.VATRateBP = ptr(1700)
	for name, d := range map[string]chargeitemdefinition.Definition{"exempt": exempt, "taxable": taxable} {
		got, err := svc.Create(t.Context(), admin, d)
		if err != nil || (got.VATRateBP == nil) != (d.VATRateBP == nil) {
			t.Errorf("%s: Create() = %+v, %v", name, got, err)
		}
	}
}

func TestUpdate_ReplacesTheServiceAtItsVersionAudited(t *testing.T) {
	repo, audit := &fakeRepository{}, &fakeAuditor{}
	svc := newService(repo, audit)
	made, err := svc.Create(t.Context(), admin, exam())
	if err != nil {
		t.Fatal(err)
	}
	next := exam()
	next.ID, next.VersionID, next.PriceMinor, next.Active = made.ID, made.VersionID, 6000, false
	got, err := svc.Update(t.Context(), admin, next)
	if err != nil || got.VersionID != 2 || got.PriceMinor != 6000 || got.Active || got.OrganizationID != clinicA {
		t.Fatalf("Update() = %+v, %v; want version 2 at the new price, inactive", got, err)
	}
	last := audit.events[len(audit.events)-1]
	if last.Subtype.Code != auditevent.CodeUpdate || last.Action != auditevent.ActionUpdate || last.Outcome != auditevent.OutcomeSuccess ||
		last.EntityReference != "ChargeItemDefinition/"+made.ID {
		t.Errorf("audit = %+v, want a successful update of the service", last)
	}

	// The version read is no longer the current one, and a service that is not the clinic's: refused, audited.
	_, stale := svc.Update(t.Context(), admin, next)
	next.ID = "d0000000-0000-4000-8000-0000000000ff"
	_, missing := svc.Update(t.Context(), admin, next)
	if !errors.Is(stale, chargeitemdefinition.ErrConflict) || !errors.Is(missing, chargeitemdefinition.ErrNotFound) {
		t.Errorf("Update(stale) = %v, Update(missing) = %v; want ErrConflict, ErrNotFound", stale, missing)
	}
	if n := len(audit.events); n != 4 || audit.events[2].Outcome != auditevent.OutcomeMinorFailure || audit.events[3].Outcome != auditevent.OutcomeMinorFailure {
		t.Errorf("audit = %+v, want the two refusals after the two writes", audit.events)
	}
	next.Name = ""
	if _, err := svc.Update(t.Context(), admin, next); !errors.Is(err, chargeitemdefinition.ErrInvalid) {
		t.Errorf("Update(blank name) = %v, want ErrInvalid", err)
	}
}

func TestList_ShowsStaffEveryServiceOfTheirClinicOnly(t *testing.T) {
	repo := &fakeRepository{rows: []chargeitemdefinition.Definition{
		{ID: "1", OrganizationID: clinicA, Name: "A", Active: true}, {ID: "2", OrganizationID: clinicA, Name: "B"},
		{ID: "3", OrganizationID: clinicB, Name: "C", Active: true},
	}}
	svc := newService(repo, &fakeAuditor{})
	for _, c := range []chargeitemdefinition.Caller{admin, staff} {
		got, err := svc.List(t.Context(), c)
		if err != nil || len(got) != 2 || got[0].ID != "1" || got[1].ID != "2" {
			t.Errorf("List(%+v) = %+v, %v; want clinic A's two", c, got, err)
		}
	}
	if _, err := svc.List(t.Context(), noClinic); !errors.Is(err, chargeitemdefinition.ErrNotFound) {
		t.Errorf("List(no clinic) = %v, want ErrNotFound", err)
	}
	if repo.outside {
		t.Error("the repository was called outside a transaction")
	}
	repo.err = errBroken
	if _, err := svc.List(t.Context(), admin); !errors.Is(err, errBroken) {
		t.Errorf("List() = %v, want the repository's error", err)
	}
}

// A service offered only in person is not shown for a video slot and the reverse; an inactive one never.
func TestOffered_ShowsTheActiveServicesOfAListedClinicInTheModeAndLanguage(t *testing.T) {
	inPerson, video := []schedule.VisitMode{schedule.ModeInPerson}, []schedule.VisitMode{schedule.ModeVideo}
	repo := &fakeRepository{rows: []chargeitemdefinition.Definition{
		{ID: "1", OrganizationID: clinicA, Name: "Pregled", Active: true, Modes: inPerson,
			Translations: []chargeitemdefinition.Translation{{Language: "de", Name: "Untersuchung"}}},
		{ID: "2", OrganizationID: clinicA, Name: "Video", Active: true, Modes: video},
		{ID: "3", OrganizationID: clinicA, Name: "Oboje", Active: true, Modes: []schedule.VisitMode{schedule.ModeInPerson, schedule.ModeVideo}},
		{ID: "4", OrganizationID: clinicA, Name: "Ukinuto", Modes: inPerson},
		{ID: "5", OrganizationID: clinicB, Name: "Tuđe", Active: true, Modes: inPerson},
	}}
	svc := newService(repo, &fakeAuditor{})
	names := func(mode schedule.VisitMode, language string) string {
		t.Helper()
		got, err := svc.Offered(t.Context(), clinicA, mode, language)
		if err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, d := range got {
			out = append(out, d.Name)
		}
		return strings.Join(out, ",")
	}
	if got := names(schedule.ModeInPerson, "de"); got != "Untersuchung,Oboje" {
		t.Errorf("in person, de: %s", got)
	}
	if got := names(schedule.ModeInPerson, ""); got != "Pregled,Oboje" {
		t.Errorf("in person, the clinic's language: %s", got)
	}
	if got := names(schedule.ModeVideo, "en"); got != "Video,Oboje" {
		t.Errorf("video, en: %s", got)
	}
	if repo.outside || !repo.directory {
		t.Error("the services were not read in the directory's scope in a transaction")
	}
	for name, tc := range map[string]struct {
		err            error
		org, mode, lng string
	}{
		"an unlisted clinic":  {chargeitemdefinition.ErrNotFound, clinicB, "video", "en"},
		"an unknown mode":     {chargeitemdefinition.ErrInvalid, clinicA, "phone", "en"},
		"no mode":             {chargeitemdefinition.ErrInvalid, clinicA, "", "en"},
		"an unknown language": {chargeitemdefinition.ErrInvalid, clinicA, "video", "xx"},
	} {
		if _, err := svc.Offered(t.Context(), tc.org, schedule.VisitMode(tc.mode), tc.lng); !errors.Is(err, tc.err) {
			t.Errorf("%s: Offered() = %v, want %v", name, err, tc.err)
		}
	}
}

// A staff booking and the desk read the clinic's whole price list in its own scope, listed or not (docs/clinic-cms F2):
// another clinic's services are dropped even if the scope showed them.
func TestOfClinic_GivesTheClinicsPriceListInItsScope(t *testing.T) {
	repo := &fakeRepository{rows: []chargeitemdefinition.Definition{
		{ID: "1", OrganizationID: clinicA, Name: "Pregled", Active: true},
		{ID: "4", OrganizationID: clinicA, Name: "Ukinuto"},
		{ID: "5", OrganizationID: clinicB, Name: "Tuđe", Active: true},
	}}
	got, err := newService(repo, &fakeAuditor{}).OfClinic(t.Context(), clinicA)
	if err != nil || len(got) != 2 || got[0].ID != "1" || got[1].ID != "4" {
		t.Errorf("OfClinic() = %+v, %v; want clinic A's two services, the inactive one included", got, err)
	}
	if repo.outside || repo.directory || repo.clinic != clinicA {
		t.Error("the services were not read in the clinic's scope in a transaction")
	}
	repo.err = errors.New("synthetic failure")
	if _, err := newService(repo, &fakeAuditor{}).OfClinic(t.Context(), clinicA); err == nil {
		t.Error("OfClinic() = nil error on a failing repository")
	}
}

// A receipt copies its lines: the name, unit price and VAT rate as they are now, and the line's total.
func TestLines_PricesThePicksFromTheClinicsActiveServices(t *testing.T) {
	repo := &fakeRepository{rows: []chargeitemdefinition.Definition{
		{ID: "1", OrganizationID: clinicA, Name: "Pregled", Currency: "BAM", PriceMinor: 5000, Active: true},
		{ID: "2", OrganizationID: clinicA, Name: "Nalaz", Currency: "BAM", PriceMinor: 1170, VATRateBP: ptr(1700), Active: true},
		{ID: "3", OrganizationID: clinicA, Name: "Ukinuto", Currency: "BAM", PriceMinor: 100},
		{ID: "4", OrganizationID: clinicB, Name: "Tuđe", Currency: "BAM", PriceMinor: 100, Active: true},
	}}
	svc := newService(repo, &fakeAuditor{})
	got, err := svc.Lines(t.Context(), clinicA, []chargeitemdefinition.Pick{
		{DefinitionID: "1", Quantity: 2, DiscountMinor: 1000}, {DefinitionID: "2", Quantity: 1},
	})
	if err != nil || len(got) != 2 {
		t.Fatalf("Lines() = %+v, %v", got, err)
	}
	first := chargeitemdefinition.Line{DefinitionID: "1", Name: "Pregled", Currency: "BAM", UnitPriceMinor: 5000, DiscountMinor: 1000, TotalMinor: 9000, Quantity: 2}
	if got[0] != first {
		t.Errorf("first line = %+v, want %+v", got[0], first)
	}
	if got[1].TotalMinor != 1170 || got[1].VATRateBP == nil || *got[1].VATRateBP != 1700 || got[1].Name != "Nalaz" {
		t.Errorf("second line = %+v, want the taxable service at 1170", got[1])
	}
	for name, tc := range map[string]struct {
		err   error
		picks []chargeitemdefinition.Pick
	}{
		"no picks":                   {chargeitemdefinition.ErrInvalid, nil},
		"an inactive service":        {chargeitemdefinition.ErrNotFound, []chargeitemdefinition.Pick{{DefinitionID: "3", Quantity: 1}}},
		"another clinic's service":   {chargeitemdefinition.ErrNotFound, []chargeitemdefinition.Pick{{DefinitionID: "4", Quantity: 1}}},
		"no quantity":                {chargeitemdefinition.ErrInvalid, []chargeitemdefinition.Pick{{DefinitionID: "1"}}},
		"a negative quantity":        {chargeitemdefinition.ErrInvalid, []chargeitemdefinition.Pick{{DefinitionID: "1", Quantity: -1}}},
		"a negative discount":        {chargeitemdefinition.ErrInvalid, []chargeitemdefinition.Pick{{DefinitionID: "1", Quantity: 1, DiscountMinor: -1}}},
		"a discount above the line":  {chargeitemdefinition.ErrInvalid, []chargeitemdefinition.Pick{{DefinitionID: "1", Quantity: 1, DiscountMinor: 5001}}},
		"a discount of the line all": {nil, []chargeitemdefinition.Pick{{DefinitionID: "1", Quantity: 1, DiscountMinor: 5000}}},
	} {
		if _, err := svc.Lines(t.Context(), clinicA, tc.picks); !errors.Is(err, tc.err) {
			t.Errorf("%s: Lines() = %v, want %v", name, err, tc.err)
		}
	}
	if repo.outside || repo.clinic != clinicA {
		t.Error("the prices were not read in the clinic's scope in a transaction")
	}
}

func TestService_ReturnsTheAuditorsError(t *testing.T) {
	repo, audit := &fakeRepository{}, &fakeAuditor{err: errBroken}
	svc := newService(repo, audit)
	_, create := svc.Create(t.Context(), admin, exam())
	_, refused := svc.Create(t.Context(), staff, exam())
	if !errors.Is(create, errBroken) || !errors.Is(refused, errBroken) {
		t.Errorf("Create() = %v, refused Create() = %v; want the auditor's error", create, refused)
	}
}
