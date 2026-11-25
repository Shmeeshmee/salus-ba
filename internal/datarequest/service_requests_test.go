package datarequest_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/datarequest"
)

// 2026-11-24 09:00 in Sarajevo.
var filedAt = time.Date(2026, 11, 24, 8, 0, 0, 0, time.UTC)

func sarajevo(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Europe/Sarajevo")
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

// A patient files to their account or to a clinic where they have a record; the request is received, due at the end
// of the same date a month later in the answerer's time zone (S1, S3), and its filing is audited as their own request.
func TestFile_ToTheAccountOrAClinicOfTheirs(t *testing.T) {
	t.Parallel()
	f := newFixture(filedAt)
	ctx := t.Context()
	toAccount, err := f.svc.File(ctx, patientA, datarequest.Filing{Kind: datarequest.KindAccess, Target: datarequest.TargetAccount})
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 12, 24, 23, 59, 59, 0, sarajevo(t))
	if toAccount.Status != datarequest.StatusReceived || toAccount.AccountID != accountA || toAccount.PatientID != profileA ||
		toAccount.FiledBy != accountA || toAccount.OrganizationID != "" || !toAccount.ReceivedAt.Equal(filedAt) ||
		!toAccount.DueAt.Equal(want) || toAccount.Version != 1 {
		t.Errorf("to the account: %+v, want received, patient A's, due %v", toAccount, want)
	}
	toClinic, err := f.svc.File(ctx, patientA, datarequest.Filing{Kind: datarequest.KindErasure, Target: datarequest.TargetClinic, OrganizationID: clinicA})
	if err != nil {
		t.Fatal(err)
	}
	if toClinic.OrganizationID != clinicA || toClinic.Target() != datarequest.TargetClinic || !toClinic.DueAt.Equal(want) {
		t.Errorf("to clinic A: %+v", toClinic)
	}
	if len(f.audit.events) != 3 { // the access request's, its export's (D2) and the erasure request's
		t.Fatalf("%d audit events, want 3", len(f.audit.events))
	}
	e := f.audit.events[2]
	if e.Action != auditevent.ActionCreate || e.Purpose.Code != auditevent.CodePATRQT || e.AgentAccountID != accountA ||
		e.EntityReference != "Patient/"+profileA || e.EntityName != "data_request" || e.OrganizationID != clinicA {
		t.Errorf("audit event %+v", e)
	}
}

// Filing an access or portability request queues its export in the filing's transaction; while the patient's export
// is still queued another one is refused with ErrExportInProgress (D1 with D2, S4). Other kinds queue nothing.
func TestFile_AnAccessRequestQueuesItsExport(t *testing.T) {
	t.Parallel()
	f := newFixture(filedAt)
	ctx := t.Context()
	r, err := f.svc.File(ctx, patientA, datarequest.Filing{Kind: datarequest.KindAccess, Target: datarequest.TargetAccount})
	if err != nil {
		t.Fatal(err)
	}
	if len(f.jobs.queued) != 1 || f.jobs.queued[0] != (datarequest.ExportArgs{RequestID: r.ID, AccountID: accountA}) {
		t.Fatalf("queued %+v, want the export of %s", f.jobs.queued, r.ID)
	}
	if _, err := f.svc.File(ctx, patientA, datarequest.Filing{Kind: datarequest.KindRectification, Target: datarequest.TargetAccount}); err != nil {
		t.Fatal(err)
	}
	if len(f.jobs.queued) != 1 {
		t.Errorf("a rectification queued %+v, want no export", f.jobs.queued[1:])
	}
	_, err = f.svc.File(ctx, patientA, datarequest.Filing{Kind: datarequest.KindPortability, Target: datarequest.TargetAccount})
	if !errors.Is(err, datarequest.ErrExportInProgress) {
		t.Errorf("a portability request while the export is queued: %v, want ErrExportInProgress", err)
	}
}

// An objection to the analytics marks the profile from the request on (S9), whether the patient files it or the desk
// files it for a patient the clinic registered; other kinds mark nothing.
func TestFile_AnObjectionMarksTheProfile(t *testing.T) {
	t.Parallel()
	f := newFixture(filedAt)
	ctx := t.Context()
	if _, err := f.svc.File(ctx, patientA, datarequest.Filing{Kind: datarequest.KindRectification, Target: datarequest.TargetAccount}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.File(ctx, patientA, datarequest.Filing{Kind: datarequest.KindObjection, Target: datarequest.TargetClinic, OrganizationID: clinicA}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.FileOnBehalf(ctx, desk, registered, datarequest.KindObjection); err != nil {
		t.Fatal(err)
	}
	if want := []string{profileA, registered}; !slices.Equal(f.objections.marked, want) {
		t.Errorf("marked %v, want %v", f.objections.marked, want)
	}
}

func TestFile_Refusals(t *testing.T) {
	t.Parallel()
	stranger := datarequest.Patient{AccountID: "00000000-0000-4000-8000-0000000000a9"} // no profile
	for name, tc := range map[string]struct {
		want    error
		filing  datarequest.Filing
		patient datarequest.Patient
	}{
		"unknown kind":              {datarequest.ErrInvalid, datarequest.Filing{Kind: "deletion", Target: datarequest.TargetAccount}, patientA},
		"unknown target":            {datarequest.ErrInvalid, datarequest.Filing{Kind: datarequest.KindAccess, Target: "practitioner"}, patientA},
		"a clinic target, none":     {datarequest.ErrInvalid, datarequest.Filing{Kind: datarequest.KindAccess, Target: datarequest.TargetClinic}, patientA},
		"the account with a clinic": {datarequest.ErrInvalid, datarequest.Filing{Kind: datarequest.KindAccess, Target: datarequest.TargetAccount, OrganizationID: clinicA}, patientA},
		"a clinic of no record":     {datarequest.ErrNotFound, datarequest.Filing{Kind: datarequest.KindAccess, Target: datarequest.TargetClinic, OrganizationID: clinicB}, patientA},
		"a clinic, no profile":      {datarequest.ErrNotFound, datarequest.Filing{Kind: datarequest.KindAccess, Target: datarequest.TargetClinic, OrganizationID: clinicA}, stranger},
		"no account":                {datarequest.ErrForbidden, datarequest.Filing{Kind: datarequest.KindAccess, Target: datarequest.TargetAccount}, datarequest.Patient{}},
	} {
		f := newFixture(filedAt)
		if _, err := f.svc.File(t.Context(), tc.patient, tc.filing); !errors.Is(err, tc.want) {
			t.Errorf("%s: %v, want %v", name, err, tc.want)
		}
		if len(f.repo.requests) != 0 {
			t.Errorf("%s: a request was stored", name)
		}
	}
}

// An account without a profile files to their account (S1); the request names no profile.
func TestFile_AnAccountWithoutAProfileFilesToTheAccount(t *testing.T) {
	t.Parallel()
	f := newFixture(filedAt)
	r, err := f.svc.File(t.Context(), datarequest.Patient{AccountID: "00000000-0000-4000-8000-0000000000a9"},
		datarequest.Filing{Kind: datarequest.KindErasure, Target: datarequest.TargetAccount})
	if err != nil || r.PatientID != "" {
		t.Errorf("File = %+v, %v; want a request with no profile", r, err)
	}
}

// The due date is the same date a month later, or the month's last day when it has no such date, at the end of that
// day in the answerer's time zone (S3); a clinic in another time zone counts in its own.
func TestFile_DueDateInTheAnswerersTimeZone(t *testing.T) {
	t.Parallel()
	tokyo, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatal(err)
	}
	for name, tc := range map[string]struct {
		now  time.Time
		want time.Time
		org  string
	}{
		"January 31st to February's last": {time.Date(2027, 1, 31, 12, 0, 0, 0, time.UTC), time.Date(2027, 2, 28, 23, 59, 59, 0, sarajevo(t)), ""},
		"late evening, already next day":  {time.Date(2026, 11, 30, 23, 30, 0, 0, time.UTC), time.Date(2027, 1, 1, 23, 59, 59, 0, sarajevo(t)), ""},
		"Tokyo's clinic, its own date":    {time.Date(2026, 11, 24, 20, 0, 0, 0, time.UTC), time.Date(2026, 12, 25, 23, 59, 59, 0, tokyo), clinicB},
	} {
		f := newFixture(tc.now)
		var r datarequest.Request
		var err error
		if tc.org == "" {
			r, err = f.svc.File(t.Context(), patientA, datarequest.Filing{Kind: datarequest.KindAccess, Target: datarequest.TargetAccount})
		} else {
			r, err = f.svc.FileOnBehalf(t.Context(), datarequest.Staff{AccountID: deskA, OrganizationID: tc.org, Reception: true}, registered, datarequest.KindAccess)
		}
		if err != nil || !r.DueAt.Equal(tc.want) {
			t.Errorf("%s: due %v, %v; want %v", name, r.DueAt, err, tc.want)
		}
	}
}

// The patient lists their own requests, newest first, audited as one search.
func TestMine_ThePatientsOwnRequests(t *testing.T) {
	t.Parallel()
	f := newFixture(filedAt)
	f.repo.put(datarequest.Request{ID: "r-old", AccountID: accountA, ReceivedAt: filedAt.Add(-48 * time.Hour), Status: datarequest.StatusFulfilled, Version: 2})
	f.repo.put(datarequest.Request{ID: "r-new", AccountID: accountA, ReceivedAt: filedAt, Status: datarequest.StatusReceived, Version: 1})
	f.repo.put(datarequest.Request{ID: "r-other", AccountID: "someone else", ReceivedAt: filedAt, Version: 1})
	got, err := f.svc.Mine(t.Context(), patientA)
	if err != nil || len(got) != 2 || got[0].ID != "r-new" || got[1].ID != "r-old" {
		t.Fatalf("Mine = %+v, %v", got, err)
	}
	if len(f.audit.events) != 1 || f.audit.events[0].Subtype.Code != auditevent.CodeSearchType || f.audit.events[0].EntityCount != 2 {
		t.Errorf("audit %+v, want one search of 2", f.audit.events)
	}
	if _, err := f.svc.Mine(t.Context(), datarequest.Patient{}); !errors.Is(err, datarequest.ErrForbidden) {
		t.Errorf("Mine without an account: %v, want ErrForbidden", err)
	}
}

// The patient withdraws their open request at its version; it keeps its history (S1), and is never deleted.
func TestWithdraw(t *testing.T) {
	t.Parallel()
	open := datarequest.Request{ID: "r1", AccountID: accountA, FiledBy: accountA, Status: datarequest.StatusReceived, ReceivedAt: filedAt, Version: 1}
	for name, tc := range map[string]struct {
		want    error
		stored  datarequest.Request
		version int
	}{
		"open":          {nil, open, 1},
		"stale version": {datarequest.ErrVersionConflict, open, 2},
		"another's":     {datarequest.ErrNotFound, func() datarequest.Request { r := open; r.AccountID = "someone else"; return r }(), 1},
		"answered":      {datarequest.ErrClosed, func() datarequest.Request { r := open; r.Status = datarequest.StatusRefused; return r }(), 1},
		"withdrawn":     {datarequest.ErrClosed, func() datarequest.Request { r := open; r.Status = datarequest.StatusWithdrawn; return r }(), 1},
	} {
		f := newFixture(filedAt)
		f.repo.put(tc.stored)
		got, err := f.svc.Withdraw(t.Context(), patientA, "r1", tc.version)
		if !errors.Is(err, tc.want) {
			t.Errorf("%s: %v, want %v", name, err, tc.want)
			continue
		}
		if tc.want != nil {
			continue
		}
		if got.Status != datarequest.StatusWithdrawn || got.Version != 2 {
			t.Errorf("%s: %+v, want withdrawn at version 2", name, got)
		}
		h, _ := f.repo.History(t.Context(), "r1")
		if len(h) != 2 || h[1].Status != datarequest.StatusWithdrawn || h[1].ChangedBy != accountA {
			t.Errorf("%s: history %+v", name, h)
		}
		if len(f.audit.events) != 1 || f.audit.events[0].Action != auditevent.ActionUpdate {
			t.Errorf("%s: audit %+v", name, f.audit.events)
		}
	}
}

// Staff file on behalf of a patient the clinic registered (S10): the admin and the receptionist, never a practitioner;
// the request targets their clinic, names no account and is filed by the staff account.
func TestFileOnBehalf(t *testing.T) {
	t.Parallel()
	f := newFixture(filedAt)
	r, err := f.svc.FileOnBehalf(t.Context(), desk, registered, datarequest.KindRectification)
	if err != nil {
		t.Fatal(err)
	}
	if r.OrganizationID != clinicA || r.PatientID != registered || r.AccountID != "" || r.FiledBy != deskA || r.Kind != datarequest.KindRectification {
		t.Errorf("FileOnBehalf = %+v", r)
	}
	if len(f.audit.events) != 1 || f.audit.events[0].Purpose.Code != auditevent.CodeHOPERAT || f.audit.events[0].EntityReference != "Patient/"+registered {
		t.Errorf("audit %+v", f.audit.events)
	}
	if _, err := f.svc.FileOnBehalf(t.Context(), admin, registered, datarequest.KindErasure); err != nil {
		t.Errorf("as the admin: %v", err)
	}
	practitioner := datarequest.Staff{AccountID: "00000000-0000-4000-8000-0000000000d3", OrganizationID: clinicA}
	if _, err := f.svc.FileOnBehalf(t.Context(), practitioner, registered, datarequest.KindErasure); !errors.Is(err, datarequest.ErrForbidden) {
		t.Errorf("as a practitioner: %v, want ErrForbidden", err)
	}
	if _, err := f.svc.FileOnBehalf(t.Context(), desk, registered, "deletion"); !errors.Is(err, datarequest.ErrInvalid) {
		t.Errorf("an unknown kind: %v, want ErrInvalid", err)
	}
	f.repo.insertErr = datarequest.ErrNotFound // a profile the clinic did not register: row-level security refuses it
	f.audit.events = nil
	if _, err := f.svc.FileOnBehalf(t.Context(), desk, "00000000-0000-4000-8000-0000000000e9", datarequest.KindErasure); !errors.Is(err, datarequest.ErrNotFound) {
		t.Errorf("another's patient: %v, want ErrNotFound", err)
	}
	// BG4: the refusal has its audit row, naming nothing it did not reach.
	if len(f.audit.events) != 1 || f.audit.events[0].Outcome != auditevent.OutcomeMinorFailure || f.audit.events[0].EntityReference != "" ||
		f.audit.events[0].OrganizationID != clinicA || f.audit.events[0].AgentAccountID != deskA {
		t.Errorf("audit of the refused filing %+v, want one refusal by the desk in clinic A", f.audit.events)
	}
}

// BG4: a patient's filing that row-level security refuses (a profile or clinic out of the scope) is audited too.
func TestFile_AuditsAFilingRowLevelSecurityRefuses(t *testing.T) {
	t.Parallel()
	f := newFixture(filedAt)
	f.repo.insertErr = datarequest.ErrNotFound
	if _, err := f.svc.File(t.Context(), patientA, datarequest.Filing{Kind: datarequest.KindAccess, Target: datarequest.TargetAccount}); !errors.Is(err, datarequest.ErrNotFound) {
		t.Fatalf("File() = %v, want ErrNotFound", err)
	}
	if len(f.audit.events) != 1 || f.audit.events[0].Outcome != auditevent.OutcomeMinorFailure || f.audit.events[0].AgentAccountID != accountA {
		t.Errorf("audit %+v, want one refusal by the patient", f.audit.events)
	}
}

// Only the clinic's admin lists and reads the requests targeting the clinic; refusals are audited.
func TestClinicRequests_TheAdminsOnly(t *testing.T) {
	t.Parallel()
	f := newFixture(filedAt)
	f.repo.put(datarequest.Request{ID: "r1", AccountID: accountA, PatientID: profileA, OrganizationID: clinicA, Status: datarequest.StatusReceived, ReceivedAt: filedAt, DueAt: filedAt.AddDate(0, 1, 0), Version: 1})
	f.repo.put(datarequest.Request{ID: "r2", AccountID: accountA, PatientID: profileA, OrganizationID: clinicB, Status: datarequest.StatusReceived, ReceivedAt: filedAt, DueAt: filedAt.AddDate(0, 1, 0), Version: 1})
	list, err := f.svc.ClinicRequests(t.Context(), admin)
	if err != nil || len(list) != 1 || list[0].ID != "r1" {
		t.Errorf("ClinicRequests = %+v, %v; want clinic A's", list, err)
	}
	r, history, err := f.svc.ClinicRequest(t.Context(), admin, "r1")
	if err != nil || r.ID != "r1" || len(history) != 1 {
		t.Errorf("ClinicRequest = %+v %+v, %v", r, history, err)
	}
	if _, _, err := f.svc.ClinicRequest(t.Context(), admin, "r2"); !errors.Is(err, datarequest.ErrNotFound) {
		t.Errorf("another clinic's: %v, want ErrNotFound", err)
	}
	n := len(f.audit.events)
	if _, err := f.svc.ClinicRequests(t.Context(), desk); !errors.Is(err, datarequest.ErrForbidden) {
		t.Errorf("the desk lists: %v, want ErrForbidden", err)
	}
	if _, _, err := f.svc.ClinicRequest(t.Context(), desk, "r1"); !errors.Is(err, datarequest.ErrForbidden) {
		t.Errorf("the desk reads: %v, want ErrForbidden", err)
	}
	if len(f.audit.events) != n+2 || f.audit.events[n].Outcome != auditevent.OutcomeMinorFailure {
		t.Errorf("refusals not audited: %+v", f.audit.events[n:])
	}
}

// The admin answers an open request with a code that fits: fulfilled with done or done_partly_kept, refused with a
// refusal's code; at its version, once.
func TestAnswer(t *testing.T) {
	t.Parallel()
	open := datarequest.Request{ID: "r1", AccountID: accountA, PatientID: profileA, OrganizationID: clinicA, Status: datarequest.StatusInProgress, ReceivedAt: filedAt, Version: 2}
	for name, tc := range map[string]struct {
		want   error
		answer datarequest.Answer
		stored datarequest.Request
	}{
		"fulfilled, done":          {nil, datarequest.Answer{Status: datarequest.StatusFulfilled, Code: datarequest.AnswerDone, Version: 2}, open},
		"refused, retention law":   {nil, datarequest.Answer{Status: datarequest.StatusRefused, Code: datarequest.AnswerRetentionLaw, Version: 2}, open},
		"fulfilled with a refusal": {datarequest.ErrInvalid, datarequest.Answer{Status: datarequest.StatusFulfilled, Code: datarequest.AnswerExcessive, Version: 2}, open},
		"refused with done":        {datarequest.ErrInvalid, datarequest.Answer{Status: datarequest.StatusRefused, Code: datarequest.AnswerDone, Version: 2}, open},
		"withdrawn is no answer":   {datarequest.ErrInvalid, datarequest.Answer{Status: datarequest.StatusWithdrawn, Code: datarequest.AnswerDone, Version: 2}, open},
		"an unknown code":          {datarequest.ErrInvalid, datarequest.Answer{Status: datarequest.StatusRefused, Code: "because", Version: 2}, open},
		"stale":                    {datarequest.ErrVersionConflict, datarequest.Answer{Status: datarequest.StatusRefused, Code: datarequest.AnswerNoData, Version: 1}, open},
		"already answered":         {datarequest.ErrClosed, datarequest.Answer{Status: datarequest.StatusRefused, Code: datarequest.AnswerNoData, Version: 2}, func() datarequest.Request { r := open; r.Status = datarequest.StatusFulfilled; return r }()},
		"another clinic's":         {datarequest.ErrNotFound, datarequest.Answer{Status: datarequest.StatusRefused, Code: datarequest.AnswerNoData, Version: 2}, func() datarequest.Request { r := open; r.OrganizationID = clinicB; return r }()},
	} {
		f := newFixture(filedAt)
		f.repo.put(tc.stored)
		got, err := f.svc.Answer(t.Context(), admin, "r1", tc.answer)
		if !errors.Is(err, tc.want) {
			t.Errorf("%s: %v, want %v", name, err, tc.want)
			continue
		}
		if tc.want == nil && (got.Status != tc.answer.Status || got.AnswerCode != tc.answer.Code || !got.AnsweredAt.Equal(filedAt) || got.Version != 3) {
			t.Errorf("%s: %+v", name, got)
		}
	}
	f := newFixture(filedAt)
	f.repo.put(open)
	if _, err := f.svc.Answer(t.Context(), desk, "r1", datarequest.Answer{Status: datarequest.StatusRefused, Code: datarequest.AnswerNoData, Version: 2}); !errors.Is(err, datarequest.ErrForbidden) {
		t.Errorf("the desk answers: %v, want ErrForbidden", err)
	}
}

// The admin extends an open request once, by two months from its due date, with a reason (S3, Art. 12(3)).
func TestExtend(t *testing.T) {
	t.Parallel()
	due := time.Date(2026, 12, 31, 23, 59, 59, 0, sarajevo(t))
	open := datarequest.Request{ID: "r1", AccountID: accountA, PatientID: profileA, OrganizationID: clinicA, Status: datarequest.StatusReceived, ReceivedAt: filedAt, DueAt: due, Version: 1}
	f := newFixture(filedAt)
	f.repo.put(open)
	got, err := f.svc.Extend(t.Context(), admin, "r1", datarequest.ExtensionComplex, 1)
	if err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2027, 2, 28, 23, 59, 59, 0, sarajevo(t)); !got.ExtendedUntil.Equal(want) || got.ExtensionReason != datarequest.ExtensionComplex ||
		got.Version != 2 || got.Status != datarequest.StatusReceived || !got.Deadline().Equal(want) {
		t.Errorf("Extend = %+v, want until %v", got, want)
	}
	if _, err := f.svc.Extend(t.Context(), admin, "r1", datarequest.ExtensionManyRequests, 2); !errors.Is(err, datarequest.ErrAlreadyExtended) {
		t.Errorf("a second extension: %v, want ErrAlreadyExtended", err)
	}
	for name, tc := range map[string]struct {
		want   error
		reason datarequest.ExtensionReason
		stored datarequest.Request
	}{
		"an unknown reason": {datarequest.ErrInvalid, "busy", open},
		"withdrawn":         {datarequest.ErrClosed, datarequest.ExtensionComplex, func() datarequest.Request { r := open; r.Status = datarequest.StatusWithdrawn; return r }()},
		"overdue (S12)":     {datarequest.ErrOverdue, datarequest.ExtensionComplex, func() datarequest.Request { r := open; r.DueAt = filedAt.Add(-time.Second); return r }()},
	} {
		f := newFixture(filedAt)
		f.repo.put(tc.stored)
		if _, err := f.svc.Extend(t.Context(), admin, "r1", tc.reason, 1); !errors.Is(err, tc.want) {
			t.Errorf("%s: %v, want %v", name, err, tc.want)
		}
	}
}

// An extension never ends later than two months after the due date as the database counts them (in UTC): in the
// autumn, the end of the day in Sarajevo is an hour past that (S3; data_request_extension).
func TestExtend_NeverPastTheDatabasesTwoMonths(t *testing.T) {
	t.Parallel()
	due := time.Date(2027, 8, 31, 23, 59, 59, 0, sarajevo(t)) // 21:59:59 UTC, summer time
	f := newFixture(filedAt)
	f.repo.put(datarequest.Request{ID: "r1", OrganizationID: clinicA, PatientID: profileA, Status: datarequest.StatusReceived, ReceivedAt: filedAt, DueAt: due, Version: 1})
	got, err := f.svc.Extend(t.Context(), admin, "r1", datarequest.ExtensionComplex, 1)
	if err != nil {
		t.Fatal(err)
	}
	if limit := time.Date(2027, 10, 31, 21, 59, 59, 0, time.UTC); got.ExtendedUntil.After(limit) || got.ExtendedUntil.Before(limit.Add(-time.Hour)) {
		t.Errorf("extended until %v, want at most %v", got.ExtendedUntil.UTC(), limit)
	}
}
