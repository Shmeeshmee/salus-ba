package auth_test

import (
	"errors"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// A job that warns a clinic's admins (a data request's deadline, docs/clinic-cms/WAVE5.md D5) reads their e-mails and
// languages: the open admin accounts of that clinic only, audited as one read with the job as agent.
func TestClinicAdmins_TheClinicsOpenAdminsAuditedWithTheJobAsAgent(t *testing.T) {
	other := staff()
	other.ID, other.Email, other.Language, other.OrganizationID = "0b6c8a52-5d0e-4a8e-9a43-3f2f5f0f0a02", "dee@example.test", "hr", "c0000000-0000-4000-8000-0000000000cc"
	doctor := staff()
	doctor.ID, doctor.Email, doctor.Role = "0b6c8a52-5d0e-4a8e-9a43-3f2f5f0f0a03", "doc@example.test", auth.RolePractitioner
	admin := staff()
	admin.Language = "en"
	f := newFixture(admin, other, doctor)

	got, err := f.svc.ClinicAdmins(t.Context(), "data_request_deadline_warning", clinicID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != (auth.Contact{Email: "ben@example.test", Language: "en"}) {
		t.Errorf("ClinicAdmins() = %+v, want the clinic's admin only", got)
	}
	e := f.onlyEvent(t)
	if e.AgentName != "data_request_deadline_warning" || e.AgentAccountID != "" || e.Action != auditevent.ActionRead ||
		e.OrganizationID != clinicID || e.EntityName != "account" || e.EntityCount != 1 {
		t.Errorf("audit event %+v, want the job's read of one account at the clinic", e)
	}
	boom := errors.New("boom")
	f = newFixture(admin)
	f.repo.err = boom
	if _, err := f.svc.ClinicAdmins(t.Context(), "data_request_deadline_warning", clinicID); !errors.Is(err, boom) {
		t.Errorf("a failing repository: error = %v, want it wrapped", err)
	}
}

func TestPostgres_ClinicAdminsAreTheClinicsAdminAccounts(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	ctx := t.Context()
	if _, err := owner.Exec(ctx, `INSERT INTO organization (id, name) VALUES ($1, 'Synthetic clinic'), ('c0000000-0000-4000-8000-0000000000cd', 'Other clinic')`, clinicID); err != nil {
		t.Fatal(err)
	}
	_, err := owner.Exec(ctx, `INSERT INTO account (email, password_hash, role, organization_id, language) VALUES
		  ('admin-a@example.test', 'x', 'admin', $1, 'hr'), ('admin-b@example.test', 'x', 'admin', $1, NULL),
		  ('desk@example.test', 'x', 'reception', $1, 'bs'), ('admin-c@example.test', 'x', 'admin', 'c0000000-0000-4000-8000-0000000000cd', 'en')`, clinicID)
	if err != nil {
		t.Fatal(err)
	}
	got, err := auth.NewPostgresRepository(app).ClinicAdmins(ctx, clinicID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Email != "admin-a@example.test" || got[0].Language != "hr" || got[1].Email != "admin-b@example.test" {
		t.Errorf("ClinicAdmins() = %+v, want the clinic's two admins", got)
	}
}
