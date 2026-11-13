package composition

import (
	"bytes"
	"context"
	"crypto/sha1" //nolint:gosec // R4's Attachment.hash is SHA-1; it is no security control
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/condition"
	"github.com/Shmeeshmee/salus/internal/encounter"
	"github.com/Shmeeshmee/salus/pkg/envelope"
	"github.com/Shmeeshmee/salus/pkg/pdfdoc"
)

// Sign makes the draft id, read at version, final for its encounter's practitioner with the patient's active
// consent: every field takes the value the practitioner left (one value per field; a value as drafted keeps its
// sources, an edited or typed one has none, an empty one is missing), the draft becomes the final Composition
// signed now by the appointment's PractitionerRole, its PDF is rendered and stored encrypted in the object
// store as a DocumentReference, and a Provenance records the signature. The three writes are audited, the
// patient's e-mail that a document is ready is enqueued with them (and, with a diagnosis, the refresh of the clinic's
// statistics for the day), and the final draft is pushed on the
// encounter's stream. The access, status and version are checked again once the
// PDF is stored, so a revocation meanwhile signs nothing (and the object is deleted).
//
// Refusals, audited: ErrNotFound, ErrForbidden, ErrConsentRequired, ErrNotPreliminary, ErrVersionMismatch, and a
// *FieldsError of ErrInvalidValues (a value of no field, a field without or with two values, a value not of
// its field's type) or ErrRequiredMissing (a required field left empty).
func (s *Service) Sign(ctx context.Context, c appointment.Caller, id string, values []Value, version int32) (Draft, error) {
	var d Draft
	var refused error
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		d, _, refused, err = s.signable(ctx, c, id, version)
		if refused == nil && err == nil {
			if d.Fields, refused = merge(d.Fields, values); refused != nil {
				return s.record(ctx, entry{c: c, interaction: auditevent.CodeUpdate, reference: "Composition/" + d.ID, outcome: auditevent.OutcomeMinorFailure})
			}
		}
		return err
	})
	if err := wrapErr("sign", err, refused); err != nil {
		return Draft{}, err
	}
	at := s.Clock.Now()
	d.SignedAt, d.DocumentReferenceID, d.ProvenanceID = at, newID(), newID()
	if d.PDF, err = s.seal(ctx, d); err != nil {
		return Draft{}, wrapErr("sign", err, nil)
	}
	err = s.Tx.WithTx(ctx, func(ctx context.Context) error {
		cur, enc, r, err := s.signable(ctx, c, id, version)
		if refused = r; r != nil || err != nil {
			return err
		}
		d.Status, d.SignerRoleID, d.VersionID = StatusFinal, enc.PractitionerRoleID, cur.VersionID
		if d, err = s.Repo.Sign(ctx, d, at); err != nil {
			return err
		}
		conditions := diagnoses(d, enc, at)
		if err := s.Conditions.Record(ctx, conditions); err != nil {
			return err
		}
		if len(conditions) > 0 {
			if err := s.Stats.EnqueueStatsRefresh(ctx, enc.OrganizationID, at); err != nil {
				return err
			}
		}
		entries := []entry{
			{interaction: auditevent.CodeUpdate, reference: "Composition/" + d.ID},
			{interaction: auditevent.CodeCreate, reference: "DocumentReference/" + d.DocumentReferenceID},
			{interaction: auditevent.CodeCreate, reference: "Provenance/" + d.ProvenanceID},
		}
		for _, c := range conditions {
			entries = append(entries, entry{interaction: auditevent.CodeCreate, reference: "Condition/" + c.ID})
		}
		for _, e := range entries {
			e.c, e.org = c, enc.OrganizationID
			if err := s.record(ctx, e); err != nil {
				return err
			}
		}
		return s.Jobs.EnqueueSigned(ctx, enc.AppointmentID, enc.OrganizationID)
	})
	if err := wrapErr("sign", err, refused); err != nil {
		_ = s.Store.Delete(context.WithoutCancel(ctx), ObjectKey(d.DocumentReferenceID))
		return Draft{}, err
	}
	s.publish(ctx, d)
	return d, nil
}

// signable returns the draft id with its encounter when the caller may sign it at version, in the transaction
// in ctx, or the refusal, audited.
func (s *Service) signable(ctx context.Context, c appointment.Caller, id string, version int32) (Draft, encounter.Encounter, error, error) {
	e := entry{c: c, interaction: auditevent.CodeUpdate, outcome: auditevent.OutcomeMinorFailure}
	d, enc, err := s.reach(ctx, c, id, s.lockedDraft, false)
	if refused := refusal(err); refused != nil {
		e.reference = encounterRef(enc.ID)
		return Draft{}, encounter.Encounter{}, refused, s.record(ctx, e)
	}
	if err != nil {
		return Draft{}, encounter.Encounter{}, nil, err
	}
	var refused error
	switch {
	case d.Status != StatusPreliminary:
		refused = ErrNotPreliminary
	case d.VersionID != version:
		refused = ErrVersionMismatch
	default:
		return d, enc, nil, nil
	}
	e.reference = "Composition/" + d.ID
	return Draft{}, encounter.Encounter{}, refused, s.record(ctx, e)
}

func (s *Service) lockedDraft(ctx context.Context, id string) (Draft, error) {
	return s.Repo.Draft(ctx, id, true)
}

// reach finds a draft by find in the caller's scope (the clinic's, or the patient's own signed documents) and
// checks the caller may reach its encounter, to read it when read is set (a signed document is then also its
// patient's), else to write it: the encounter as far as the caller may know it with a refusal (errRefused).
func (s *Service) reach(ctx context.Context, c appointment.Caller, id string, find func(ctx context.Context, id string) (Draft, error), read bool) (Draft, encounter.Encounter, error) {
	var err error
	switch {
	case c.OrganizationID != "":
		err = s.Repo.SetClinic(ctx, c.OrganizationID)
	case c.PatientID != "":
		err = s.Repo.SetPatient(ctx, c.PatientID)
	default:
		return Draft{}, encounter.Encounter{}, errRefused{ErrNotFound}
	}
	if err != nil {
		return Draft{}, encounter.Encounter{}, err
	}
	d, err := find(ctx, id)
	switch {
	case errors.Is(err, ErrNotFound), err == nil && c.OrganizationID == "" && d.Status != StatusFinal:
		return Draft{}, encounter.Encounter{}, errRefused{ErrNotFound}
	case err != nil:
		return Draft{}, encounter.Encounter{}, err
	}
	enc, err := s.allowed(ctx, c, d.EncounterID, read && d.Status == StatusFinal)
	return d, enc, err
}

// merge sets each field to the practitioner's value, or returns the *FieldsError.
func merge(fields []DraftField, values []Value) ([]DraftField, error) {
	byKey := make(map[string]string, len(values))
	confirmed := make(map[string]bool, len(values))
	var bad, unconfirmed []string
	for _, v := range values {
		if _, twice := byKey[v.Key]; twice && !slices.Contains(bad, v.Key) {
			bad = append(bad, v.Key)
		}
		byKey[v.Key], confirmed[v.Key] = v.Value, v.Confirmed
	}
	unknown := false
	for key := range byKey {
		if !slices.ContainsFunc(fields, func(f DraftField) bool { return f.Key == key }) {
			unknown = true
		}
	}
	out := slices.Clone(fields)
	var missing []string
	for i, f := range out {
		v, ok := byKey[f.Key]
		value, typed := normalise(f.Type, v)
		switch {
		case !ok:
			bad = append(bad, f.Key)
		case slices.Contains(bad, f.Key):
		case strings.TrimSpace(v) == "":
			out[i].Field = Field{Key: f.Key, Missing: true}
			if f.Required {
				missing = append(missing, f.Key)
			}
		case !typed:
			bad = append(bad, f.Key)
		case f.Type == FieldCoded && !confirmed[f.Key]:
			unconfirmed = append(unconfirmed, f.Key)
		case value == f.Value && !f.Missing:
		default:
			out[i].Field = Field{Key: f.Key, Value: value}
		}
	}
	switch {
	case len(bad) > 0 || unknown:
		return nil, &FieldsError{Err: ErrInvalidValues, Keys: inOrder(fields, bad)}
	case len(missing) > 0:
		return nil, &FieldsError{Err: ErrRequiredMissing, Keys: missing}
	case len(unconfirmed) > 0:
		return nil, &FieldsError{Err: ErrDiagnosisUnconfirmed, Keys: unconfirmed}
	}
	return out, nil
}

// diagnoses are the Conditions of the signed draft d: one per filled coded field, confirmed by the signature
// (merge refused the unconfirmed), of the encounter's patient and clinic, recorded at by the signing role.
func diagnoses(d Draft, enc encounter.Encounter, at time.Time) []condition.Condition {
	var out []condition.Condition
	for _, f := range d.Fields {
		if f.Type != FieldCoded || f.Missing || f.Value == "" {
			continue
		}
		out = append(out, condition.Condition{
			ID: newID(), RecordedAt: at, OrganizationID: enc.OrganizationID, PatientID: enc.PatientID, EncounterID: enc.ID,
			DocumentID: d.ID, PractitionerRoleID: d.SignerRoleID, FieldKey: f.Key, Code: f.Value,
			VerificationStatus: condition.VerificationConfirmed,
		})
	}
	return out
}

// inOrder is the keys in the fields' order.
func inOrder(fields []DraftField, keys []string) []string {
	var out []string
	for _, f := range fields {
		if slices.Contains(keys, f.Key) {
			out = append(out, f.Key)
		}
	}
	return out
}

// seal renders the signed draft d as PDF and stores it encrypted under its DocumentReference's id, outside any
// transaction, and returns what the row keeps of it.
func (s *Service) seal(ctx context.Context, d Draft) (PDF, error) {
	b, err := pdfdoc.Render(document(d))
	if err != nil {
		return PDF{}, err
	}
	dek, err := envelope.NewDataKey()
	if err != nil {
		return PDF{}, err
	}
	p := PDF{KEKID: s.KEK.ID(), Size: int64(len(b))}
	if p.WrappedKey, err = s.KEK.Wrap(dek, d.DocumentReferenceID); err != nil {
		return PDF{}, err
	}
	sum := sha1.Sum(b) //nolint:gosec // see the import
	p.SHA1 = sum[:]
	var sealed bytes.Buffer
	w, err := envelope.NewWriter(&sealed, dek, d.DocumentReferenceID)
	if err != nil {
		return PDF{}, err
	}
	if _, err := w.Write(b); err != nil {
		return PDF{}, err
	}
	if err := w.Close(); err != nil {
		return PDF{}, err
	}
	return p, s.Store.Put(ctx, ObjectKey(d.DocumentReferenceID), &sealed, int64(sealed.Len()))
}

// document is the signed draft as the PDF shows it: its title, when it was signed (UTC), and each field under
// its label, a list as its items, a missing field as a dash.
func document(d Draft) pdfdoc.Document {
	doc := pdfdoc.Document{Created: d.SignedAt, Title: d.Title, Meta: []string{d.SignedAt.UTC().Format("2006-01-02 15:04") + " UTC"}}
	for _, f := range d.Fields {
		sec := pdfdoc.Section{Heading: f.Label}
		switch {
		case f.Missing:
		case f.Type == FieldList:
			sec.Items = strings.Split(f.Value, "\n")
		default:
			sec.Paragraphs = []string{f.Value}
		}
		doc.Sections = append(doc.Sections, sec)
	}
	return doc
}
