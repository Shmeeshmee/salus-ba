package history_test

import (
	"context"

	"github.com/Shmeeshmee/salus/internal/allergyintolerance"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/history"
	"github.com/Shmeeshmee/salus/internal/medicationstatement"
)

// fakeAllergies answers Seen with the entries of each patient, as SAL-41's service will after its access rule; it
// records who asked.
type fakeAllergies struct {
	err     error
	seen    map[string][]allergyintolerance.Seen // by patient
	callers []appointment.Caller
}

func (f *fakeAllergies) Seen(_ context.Context, c appointment.Caller, patientID string) ([]allergyintolerance.Seen, error) {
	f.callers = append(f.callers, c)
	return f.seen[patientID], f.err
}

// fakeMedications is fakeAllergies for medications.
type fakeMedications struct {
	err     error
	seen    map[string][]medicationstatement.Seen
	callers []appointment.Caller
}

func (f *fakeMedications) Seen(_ context.Context, c appointment.Caller, patientID string) ([]medicationstatement.Seen, error) {
	f.callers = append(f.callers, c)
	return f.seen[patientID], f.err
}

var (
	_ history.Allergies   = (*fakeAllergies)(nil)
	_ history.Medications = (*fakeMedications)(nil)
)
