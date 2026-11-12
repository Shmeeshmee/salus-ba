package condition_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/condition"
	"github.com/Shmeeshmee/salus/pkg/fhir"
)

// fakeRepo is the Repository: it keeps what it is given and answers Diagnoses with diagnoses.
type fakeRepo struct {
	inserted  []condition.Condition
	diagnoses []condition.Diagnosis
	asked     [3]any
	err       error
}

func (f *fakeRepo) Insert(_ context.Context, cs []condition.Condition) error {
	f.inserted = append(f.inserted, cs...)
	return f.err
}

func (f *fakeRepo) Diagnoses(_ context.Context, org string, from, to time.Time) ([]condition.Diagnosis, error) {
	f.asked = [3]any{org, from, to}
	return f.diagnoses, f.err
}

func (f *fakeRepo) Unconfirmed(context.Context) ([]string, error) {
	return []string{"00000000-0000-4000-8000-0000000000c1"}, f.err
}

const (
	clinic  = "00000000-0000-4000-8000-0000000000aa"
	role    = "00000000-0000-4000-8000-0000000000bb"
	patient = "00000000-0000-4000-8000-0000000000cc"
	other   = "00000000-0000-4000-8000-0000000000dd"
)

var signedAt = time.Date(2026, 11, 2, 9, 30, 0, 0, time.UTC)

func valid() condition.Condition {
	return condition.Condition{
		RecordedAt: signedAt, OrganizationID: clinic, PatientID: patient, EncounterID: other, DocumentID: other,
		PractitionerRoleID: role, FieldKey: "diagnosis", Code: "J06.9", VerificationStatus: condition.VerificationConfirmed,
	}
}

func TestRecord_WritesTheConditions(t *testing.T) {
	repo := &fakeRepo{}
	if err := condition.NewService(repo).Record(context.Background(), []condition.Condition{valid()}); err != nil {
		t.Fatal(err)
	}
	want := valid()
	if len(repo.inserted) != 1 || !fhir.IsUUID(repo.inserted[0].ID) {
		t.Fatalf("inserted %+v, want one with a new id", repo.inserted)
	}
	if want.ID = repo.inserted[0].ID; repo.inserted[0] != want {
		t.Errorf("inserted %+v, want %+v", repo.inserted[0], want)
	}
}

func TestRecord_RefusesAConditionTheModelDoesNotTake(t *testing.T) {
	for name, edit := range map[string]func(*condition.Condition){
		"a malformed code":     func(c *condition.Condition) { c.Code = "J6" },
		"a lower-case code":    func(c *condition.Condition) { c.Code = "j06.9" },
		"an unknown status":    func(c *condition.Condition) { c.VerificationStatus = "refuted" },
		"no field key":         func(c *condition.Condition) { c.FieldKey = "" },
		"no recording time":    func(c *condition.Condition) { c.RecordedAt = time.Time{} },
		"a patient not a UUID": func(c *condition.Condition) { c.PatientID = "ada" },
	} {
		c := valid()
		edit(&c)
		repo := &fakeRepo{}
		if err := condition.NewService(repo).Record(context.Background(), []condition.Condition{valid(), c}); !errors.Is(err, condition.ErrInvalid) || len(repo.inserted) != 0 {
			t.Errorf("%s: Record = %v with %d written, want ErrInvalid and none", name, err, len(repo.inserted))
		}
	}
}

// The builds get each confirmed diagnosis with a pseudonym of its patient under the build's key: the same patient
// gives the same pseudonym within a build, another key another one, and without a key the pseudonym is zero (a
// build that does not count patients).
func TestRecorded_PseudonymisesThePatientUnderTheKey(t *testing.T) {
	repo := &fakeRepo{diagnoses: []condition.Diagnosis{
		{RecordedAt: signedAt, PatientID: patient, PractitionerRoleID: role, Code: "J00"},
		{RecordedAt: signedAt, PatientID: patient, PractitionerRoleID: role, Code: "I10"},
		{RecordedAt: signedAt, PatientID: other, PractitionerRoleID: role, Code: "J00"},
	}}
	svc := condition.NewService(repo)
	from, to := signedAt.Add(-time.Hour), signedAt.Add(time.Hour)
	a, err := svc.Recorded(context.Background(), clinic, from, to, []byte("key one, 32 bytes long, synthet"))
	if err != nil {
		t.Fatal(err)
	}
	if repo.asked != [3]any{clinic, from, to} {
		t.Errorf("asked the repository %v", repo.asked)
	}
	b, _ := svc.Recorded(context.Background(), clinic, from, to, []byte("key two, 32 bytes long, synthet"))
	none, _ := svc.Recorded(context.Background(), clinic, from, to, nil)
	var zero [32]byte
	switch {
	case len(a) != 3 || a[0].Code != "J00" || a[0].PractitionerRoleID != role || !a[0].RecordedAt.Equal(signedAt):
		t.Errorf("Recorded = %+v", a)
	case a[0].Patient != a[1].Patient || a[0].Patient == a[2].Patient || a[0].Patient == zero:
		t.Error("the pseudonyms do not follow the patients")
	case a[0].Patient == b[0].Patient:
		t.Error("two keys give the same pseudonym")
	case none[0].Patient != zero:
		t.Error("no key gives a pseudonym")
	}
}

func TestService_WrapsTheRepositorysError(t *testing.T) {
	boom := errors.New("boom")
	svc := condition.NewService(&fakeRepo{err: boom})
	if err := svc.Record(context.Background(), []condition.Condition{valid()}); !errors.Is(err, boom) {
		t.Errorf("Record = %v", err)
	}
	if _, err := svc.Recorded(context.Background(), clinic, signedAt, signedAt, nil); !errors.Is(err, boom) {
		t.Errorf("Recorded = %v", err)
	}
	if _, err := svc.Unconfirmed(context.Background()); !errors.Is(err, boom) {
		t.Errorf("Unconfirmed = %v", err)
	}
}

func TestRecord_WritesNothingForNoConditions(t *testing.T) {
	repo := &fakeRepo{err: errors.New("not called")}
	if err := condition.NewService(repo).Record(context.Background(), nil); err != nil {
		t.Errorf("Record(nil) = %v", err)
	}
}
