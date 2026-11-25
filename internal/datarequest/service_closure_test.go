package datarequest_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/internal/datarequest"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

func (f *closureFixture) closure() *datarequest.Closure {
	return datarequest.NewClosure(datarequest.ClosurePorts{
		Tx: closureTx{}, Repo: f.repo, Scope: f.scope, Accounts: f.accounts, Profiles: closureProfiles{id: f.profile},
		Consents: f.consents, Devices: f.devices, Clock: clock.Fixed{T: closedNow},
	})
}

func careAt(id, org string, kind consent.Kind, status consent.Status) consent.Consent {
	return consent.Consent{ID: id, PatientID: profileID, OrganizationID: org, Kind: kind, Status: status}
}

func TestCloseAccount_ClosesRevokesAndFilesTheErasureRequestInProgress(t *testing.T) {
	t.Parallel()
	f := newClosureFixture(profileID,
		careAt("t1", clinicA, consent.KindTranscription, consent.StatusActive),
		careAt("c1", clinicA, consent.KindCare, consent.StatusActive),
		careAt("c2", clinicB, consent.KindCare, consent.StatusInactive),
		careAt("h2", clinicB, consent.KindHistory, consent.StatusActive),
	)

	got, err := f.closure().CloseAccount(t.Context(), datarequest.Patient{AccountID: accountID}, password)
	if err != nil {
		t.Fatalf("CloseAccount() error = %v", err)
	}
	if !f.accounts.closed || f.accounts.closedFor != datarequest.EraseAfter {
		t.Errorf("closed %v for %v, want closed for %v", f.accounts.closed, f.accounts.closedFor, datarequest.EraseAfter)
	}
	// The care consent first: its revocation ends the transcription consent with the clinic (and cancels the visits
	// there through consent_revoked); the history consent at the other clinic on its own.
	if !slices.Equal(f.consents.revoked, []string{"c1", "h2"}) {
		t.Errorf("revoked %v, want [c1 h2]", f.consents.revoked)
	}
	for _, c := range f.consents.callers {
		if c.AccountID != accountID || c.PatientID != profileID || c.OrganizationID != "" {
			t.Errorf("consent caller %+v, want the patient", c)
		}
	}
	if !slices.Equal(f.devices.removed, []string{accountID}) {
		t.Errorf("devices removed for %v, want the account's", f.devices.removed)
	}
	want := datarequest.Request{
		ID: got.ID, AccountID: accountID, PatientID: profileID, FiledBy: accountID, Kind: datarequest.KindErasure,
		Status: datarequest.StatusInProgress, ReceivedAt: closedNow, DueAt: closedNow.AddDate(0, datarequest.DueMonths, 0), Version: 2,
	}
	if got != want || len(f.repo.requests) != 1 || f.repo.requests[0] != want {
		t.Errorf("request %+v (stored %+v), want %+v", got, f.repo.requests, want)
	}
	if !got.DueAt.After(closedNow.Add(datarequest.EraseAfter)) {
		t.Errorf("due %v, not after the erasure at %v", got.DueAt, closedNow.Add(datarequest.EraseAfter))
	}
	if n := len(f.repo.changes); n != 2 || f.repo.changes[0].ChangedBy != accountID || f.repo.changes[1].ChangedBy != "" {
		t.Errorf("history %+v, want received by the patient and in progress by the closure", f.repo.changes)
	}
}

// The patient's chat consents end with the closure too, in its transaction, so a doctor of another clinic they invited
// into a conversation keeps no access (owner, 2026-11-25).
func TestCloseAccount_RevokesTheChatConsents(t *testing.T) {
	t.Parallel()
	f := newClosureFixture(profileID, careAt("c1", clinicA, consent.KindCare, consent.StatusActive))
	f.consents.chats = 2
	if _, err := f.closure().CloseAccount(t.Context(), datarequest.Patient{AccountID: accountID}, password); err != nil {
		t.Fatalf("CloseAccount() error = %v", err)
	}
	if !slices.Equal(f.consents.revoked, []string{"c1", "chat1", "chat2"}) || f.consents.chats != 0 {
		t.Errorf("revoked %v, %d chat consents left; want the care consent, then both chat consents", f.consents.revoked, f.consents.chats)
	}
}

func TestCloseAccount_AnAccountWithoutAProfileIsClosedToo(t *testing.T) {
	t.Parallel()
	f := newClosureFixture("")
	got, err := f.closure().CloseAccount(t.Context(), datarequest.Patient{AccountID: accountID}, password)
	if err != nil {
		t.Fatalf("CloseAccount() error = %v", err)
	}
	if got.PatientID != "" || !f.accounts.closed || len(f.consents.callers) != 0 {
		t.Errorf("request %+v, closed %v, consent calls %d; want closed with no profile", got, f.accounts.closed, len(f.consents.callers))
	}
}

func TestCloseAccount_Refuses(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		want     error
		password string
		closed   bool
	}{
		"a wrong password": {password: "not the password", want: datarequest.ErrWrongPassword},
		"a closed account": {password: password, closed: true, want: datarequest.ErrNotFound},
	} {
		t.Run(name, func(t *testing.T) {
			f := newClosureFixture(profileID, careAt("c1", clinicA, consent.KindCare, consent.StatusActive))
			f.accounts.closed = tc.closed
			if _, err := f.closure().CloseAccount(t.Context(), datarequest.Patient{AccountID: accountID}, tc.password); !errors.Is(err, tc.want) {
				t.Fatalf("CloseAccount() error = %v, want %v", err, tc.want)
			}
			if len(f.consents.revoked) != 0 || len(f.repo.requests) != 0 || len(f.devices.removed) != 0 {
				t.Errorf("revoked %v, filed %v, devices %v; want nothing", f.consents.revoked, f.repo.requests, f.devices.removed)
			}
		})
	}
}

func TestReopenAccount_WithdrawsTheErasureRequestAndReopens(t *testing.T) {
	t.Parallel()
	f := newClosureFixture(profileID)
	if _, err := f.closure().CloseAccount(t.Context(), datarequest.Patient{AccountID: accountID}, password); err != nil {
		t.Fatal(err)
	}
	if _, err := f.closure().ReopenAccount(t.Context(), accountID); err != nil {
		t.Fatalf("ReopenAccount() error = %v", err)
	}
	if !f.accounts.reopened || f.repo.requests[0].Status != datarequest.StatusWithdrawn || f.repo.requests[0].Version != 3 {
		t.Errorf("reopened %v, request %+v; want reopened and the request withdrawn", f.accounts.reopened, f.repo.requests[0])
	}
	if _, err := f.closure().ReopenAccount(t.Context(), accountID); !errors.Is(err, datarequest.ErrNotFound) {
		t.Errorf("a second ReopenAccount() error = %v, want ErrNotFound", err)
	}
}
