package careplan

import (
	"context"
	"fmt"
	"slices"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// Read returns one plan of the staff's clinic, its steps in the order of their visits once booked; audited.
func (s *Service) Read(ctx context.Context, c Staff, id string) (Plan, error) {
	return s.staffTx(ctx, c, auditevent.CodeRead, func(ctx context.Context) (Plan, error) {
		p, err := s.seen(ctx, c, id)
		if err != nil {
			return Plan{}, err
		}
		return s.settle(ctx, c.OrganizationID, p)
	})
}

// ForPatient returns a patient's plans at the staff's clinic, drafts included, newest first; each is audited as read.
func (s *Service) ForPatient(ctx context.Context, c Staff, patientID string) ([]Plan, error) {
	var out []Plan
	_, err := s.staffTx(ctx, c, auditevent.CodeSearchType, func(ctx context.Context) (Plan, error) {
		if err := s.works(ctx, c, patientID); err != nil {
			return Plan{}, err
		}
		var err error
		if out, err = s.list(ctx, c.AccountID, c.OrganizationID, patientID, ""); err != nil {
			return Plan{}, err
		}
		return Plan{}, s.named(ctx, out)
	})
	return out, err
}

// Mine returns the patient's booked plans at every clinic, newest first; each is audited as read.
func (s *Service) Mine(ctx context.Context, c Patient) ([]Plan, error) {
	return s.MineAt(ctx, c, "")
}

// MineAt is Mine of the clinic org only, every clinic's when empty: the patient's export of one clinic reads, and
// audits, nothing of another's.
func (s *Service) MineAt(ctx context.Context, c Patient, org string) ([]Plan, error) {
	var out []Plan
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.p.Repo.SetPatient(ctx, c.PatientID); err != nil {
			return err
		}
		var err error
		out, err = s.list(ctx, c.AccountID, "", c.PatientID, org)
		return err
	})
	if err == nil {
		err = s.named(ctx, out) // each clinic's price list in its own scope, after the patient's
	}
	if err != nil {
		return nil, fmt.Errorf("careplan: mine: %w", err)
	}
	return out, nil
}

// list reads the patient's plans in the scope set (of the clinic at only, when set), settles each, and audits the
// search (naming the patient, also when nothing is found) and each plan as read by account.
func (s *Service) list(ctx context.Context, account, org, patientID, only string) ([]Plan, error) {
	all, err := s.p.Repo.ForPatient(ctx, patientID)
	if err != nil {
		return nil, err
	}
	if only != "" {
		all = slices.DeleteFunc(all, func(p Plan) bool { return p.OrganizationID != only })
	}
	search := entry(account, org, auditevent.CodeSearchType, "", auditevent.OutcomeSuccess)
	search.EntityReference = "Patient/" + patientID // the list's subject, also when it is empty
	if err := s.p.Audit.Record(ctx, search); err != nil {
		return nil, err
	}
	out := make([]Plan, 0, len(all))
	for _, p := range all {
		if p, err = s.settle(ctx, org, p); err != nil {
			return nil, err
		}
		if err := s.p.Audit.Record(ctx, entry(account, org, auditevent.CodeRead, p.ID, auditevent.OutcomeSuccess)); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

// settle orders a booked plan's steps by their visits' start (a moved step shows in its new place) and shows an active
// plan whose every visit has ended or was cancelled as completed. org is the clinic's scope, empty for the patient's.
func (s *Service) settle(ctx context.Context, org string, p Plan) (Plan, error) {
	if p.BookedAt == nil {
		return p, nil
	}
	visits, err := s.p.Visits.PlanVisits(ctx, org, visitIDs(p))
	if err != nil {
		return Plan{}, err
	}
	byID := make(map[string]appointment.Appointment, len(visits))
	ended := len(visits) == len(p.Steps)
	now := s.p.Clock.Now()
	for _, v := range visits {
		byID[v.ID] = v
		ended = ended && (!v.End.After(now) || v.Status == appointment.StatusCancelled)
	}
	slices.SortStableFunc(p.Steps, func(a, b Step) int { return byID[a.AppointmentID].Start.Compare(byID[b.AppointmentID].Start) })
	for i, st := range p.Steps {
		p.Steps[i].Position = i + 1
		if v, ok := byID[st.AppointmentID]; ok {
			start, end := v.Start, v.End
			p.Steps[i].VisitStart, p.Steps[i].VisitEnd, p.Steps[i].VisitStatus = &start, &end, string(v.Status)
		}
	}
	if p.Status == StatusActive && ended {
		p.Status = StatusCompleted
	}
	return p, nil
}

func visitIDs(p Plan) []string {
	ids := make([]string, 0, len(p.Steps))
	for _, st := range p.Steps {
		if st.AppointmentID != "" {
			ids = append(ids, st.AppointmentID)
		}
	}
	return ids
}
