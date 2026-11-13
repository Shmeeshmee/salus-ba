package clinicstats_test

import (
	"testing"

	"github.com/Shmeeshmee/salus/internal/clinicstats"
)

// SAL-36: an admin also gets the figures per specialty; a practitioner who is not an admin never does (C2: a
// specialty of one practitioner is that colleague's figures), nor the no-shows per practitioner.
func TestAppointments_OnlyAnAdminGetsTheSpecialties(t *testing.T) {
	t.Parallel()
	f := newReadFixture()
	got, err := f.svc.Appointments(t.Context(), admin, year)
	if err != nil || len(got.Specialties) != 1 || got.Specialties[0].Booked != 3 {
		t.Errorf("the admin's Appointments() = %+v, %v; want the specialties", got, err)
	}
	f = newReadFixture()
	got, err = f.svc.Appointments(t.Context(), doctor, year)
	if err != nil || got.Specialties != nil || got.Practitioners != nil {
		t.Errorf("a practitioner's Appointments() = %+v, %v; want neither the specialties nor the practitioners", got, err)
	}
	for _, q := range f.repo.asked {
		if _, ok := q.(clinicstats.AppointmentQuery); !ok {
			t.Errorf("asked %+v", q)
		}
	}
	if len(f.repo.asked) != 2 {
		t.Errorf("a practitioner's read asked %d queries, want the series and the heatmap only", len(f.repo.asked))
	}
}
