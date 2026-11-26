package consent

import (
	"context"
	"fmt"
	"slices"

	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// ErasureAgent names the erasure job as the agent of its reads (docs/clinic-cms/WAVE5.md S6).
const ErasureAgent = "data_request_erasure"

// ClinicsOf returns, sorted, the clinics the patient ever gave a consent to, active or not: those that keep a record
// of the patient (a booking and a clinic's registration need one). The erasure job's read, in the patient's scope,
// audited as one search by the job with the number of consents read.
func (s *Service) ClinicsOf(ctx context.Context, patientID string) ([]string, error) {
	var out []string
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.repo.SetPatient(ctx, patientID); err != nil {
			return err
		}
		all, err := s.repo.ByPatient(ctx, patientID)
		if err != nil {
			return err
		}
		n := int64(0)
		for _, x := range all {
			if x.PatientID != patientID { // not on row-level security alone
				continue
			}
			n++
			if !slices.Contains(out, x.OrganizationID) {
				out = append(out, x.OrganizationID)
			}
		}
		return s.audit.Record(ctx, auditevent.AuditEvent{
			Type:        auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
			Subtype:     auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: auditevent.CodeSearchType},
			Purpose:     auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeHOPERAT},
			Action:      auditevent.ActionExecute,
			Outcome:     auditevent.OutcomeSuccess,
			AgentName:   ErasureAgent,
			EntityName:  "consent",
			EntityCount: n,
		})
	})
	if err != nil {
		return nil, fmt.Errorf("consent: clinics of: %w", err)
	}
	slices.Sort(out)
	return out, nil
}

// CareClinics returns, sorted, the clinics the patient has a care consent with, active or revoked, in the patient's
// scope: those that keep the patient's visits and records (every booking needs one), and whose public reads the
// patient keeps once the clinic's membership stops (organization.Service.Served). Not audited, as State: the access
// check of the caller's own read, which discloses nothing of the patient. None for no patient.
func (s *Service) CareClinics(ctx context.Context, patientID string) ([]string, error) {
	if patientID == "" {
		return nil, nil
	}
	var out []string
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.repo.SetPatient(ctx, patientID); err != nil {
			return err
		}
		all, err := s.repo.ByPatient(ctx, patientID)
		for _, x := range all {
			if x.Kind == KindCare && x.PatientID == patientID && !slices.Contains(out, x.OrganizationID) { // not on row-level security alone
				out = append(out, x.OrganizationID)
			}
		}
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("consent: care clinics: %w", err)
	}
	slices.Sort(out)
	return out, nil
}
