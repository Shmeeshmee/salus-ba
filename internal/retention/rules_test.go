package retention_test

import (
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/retention"
)

func TestRules_AreCompleteAndNameEachTableOnce(t *testing.T) {
	seen := map[string]bool{}
	for _, r := range retention.Rules() {
		if r.Table == "" || r.Basis == "" || r.Then == "" {
			t.Errorf("rule %+v: want a table, a basis and what happens then", r)
		}
		// A row kept while its subject exists, or restricted once it is no longer active, has no period; a row
		// removed by range has a positive one; a deleted row may be deleted as soon as it stops being used (a
		// questionnaire response once due), so its period is zero or more.
		switch r.Then {
		case retention.ThenKeep, retention.ThenRestrict:
			if r.Keep != 0 {
				t.Errorf("rule %+v: want no period for a kept or restricted row", r)
			}
		case retention.ThenRemoveRange:
			if r.Keep <= 0 {
				t.Errorf("rule %+v: want a positive period for a row removed by range", r)
			}
		case retention.ThenDelete:
			if r.Keep < 0 {
				t.Errorf("rule %+v: want a period of zero or more for a deleted row", r)
			}
		default:
			t.Errorf("rule %+v: unknown then", r)
		}
		if seen[r.Table] {
			t.Errorf("table %s has two rules", r.Table)
		}
		seen[r.Table] = true
	}
}

func TestRules_KeepAuditEventsTenYearsThenRemoveThemByRange(t *testing.T) {
	for _, r := range retention.Rules() {
		if r.Table != "audit_event" {
			continue
		}
		if tenYears := time.Date(2036, 1, 1, 0, 0, 0, 0, time.UTC).Sub(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)); r.Keep < tenYears {
			t.Errorf("audit_event kept %v, want at least ten calendar years (%v)", r.Keep, tenYears)
		}
		if r.Then != retention.ThenRemoveRange {
			t.Errorf("audit_event then %q, want %q", r.Then, retention.ThenRemoveRange)
		}
		return
	}
	t.Fatal("no rule for audit_event")
}

func TestRules_KeepAccountsWhileTheyExist(t *testing.T) {
	for _, r := range retention.Rules() {
		if r.Table == "account" {
			if r.Then != retention.ThenKeep {
				t.Errorf("account then %q, want %q", r.Then, retention.ThenKeep)
			}
			return
		}
	}
	t.Fatal("no rule for account")
}

func TestRules_DeleteSessionsThirtyDaysAfterTheyEnd(t *testing.T) {
	for _, r := range retention.Rules() {
		if r.Table == "session" {
			if r.Then != retention.ThenDelete || r.Keep != 30*24*time.Hour {
				t.Errorf("session kept %v then %q, want 30 days then %q", r.Keep, r.Then, retention.ThenDelete)
			}
			return
		}
	}
	t.Fatal("no rule for session")
}

func TestRules_RestrictPractitionersAndTheirRolesAfterTheyLeave(t *testing.T) {
	want := map[string]bool{"practitioner": true, "practitioner_role": true}
	for _, r := range retention.Rules() {
		if !want[r.Table] {
			continue
		}
		if r.Then != retention.ThenRestrict {
			t.Errorf("%s then %q, want %q", r.Table, r.Then, retention.ThenRestrict)
		}
		delete(want, r.Table)
	}
	for table := range want {
		t.Errorf("no rule for %s", table)
	}
}

func TestRules_DeleteInvitesThirtyDaysAfterTheyEnd(t *testing.T) {
	for _, r := range retention.Rules() {
		if r.Table == "invite" {
			if r.Then != retention.ThenDelete || r.Keep != 30*24*time.Hour {
				t.Errorf("invite kept %v then %q, want 30 days then %q", r.Keep, r.Then, retention.ThenDelete)
			}
			return
		}
	}
	t.Fatal("no rule for invite")
}

func TestRules_KeepPatientsWhileTheAccountExists(t *testing.T) {
	for _, r := range retention.Rules() {
		if r.Table == "patient" {
			if r.Then != retention.ThenKeep {
				t.Errorf("patient then %q, want %q", r.Then, retention.ThenKeep)
			}
			return
		}
	}
	t.Fatal("no rule for patient")
}

func TestRules_KeepConsentsWithTheClinicalRecord(t *testing.T) {
	for _, r := range retention.Rules() {
		if r.Table == "consent" {
			if r.Then != retention.ThenKeep {
				t.Errorf("consent then %q, want %q", r.Then, retention.ThenKeep)
			}
			return
		}
	}
	t.Fatal("no rule for consent")
}

func TestRules_KeepTranscriptionConsentsWithTheClinicalRecord(t *testing.T) {
	for _, r := range retention.Rules() {
		if r.Table == "transcription_consent" {
			if r.Then != retention.ThenKeep {
				t.Errorf("transcription_consent then %q, want %q", r.Then, retention.ThenKeep)
			}
			return
		}
	}
	t.Fatal("no rule for transcription_consent")
}

func TestRules_KeepAppointmentsWithTheClinicalRecord(t *testing.T) {
	for _, r := range retention.Rules() {
		if r.Table == "appointment" {
			if r.Then != retention.ThenKeep {
				t.Errorf("appointment then %q, want %q", r.Then, retention.ThenKeep)
			}
			return
		}
	}
	t.Fatal("no rule for appointment")
}

func TestRules_RestrictSchedulesAndDeleteFreeSlotsThirtyDaysAfterTheyEnd(t *testing.T) {
	want := map[string]retention.Rule{
		"schedule": {Table: "schedule", Then: retention.ThenRestrict},
		"slot":     {Table: "slot", Then: retention.ThenDelete, Keep: 30 * 24 * time.Hour},
	}
	for _, r := range retention.Rules() {
		if w, ok := want[r.Table]; ok {
			if r.Then != w.Then || r.Keep != w.Keep {
				t.Errorf("%s then %q after %v, want %q after %v", r.Table, r.Then, r.Keep, w.Then, w.Keep)
			}
			delete(want, r.Table)
		}
	}
	for table := range want {
		t.Errorf("no rule for %s", table)
	}
}

// An unsubmitted questionnaire response is deleted once its due time (the appointment's start) has passed; a
// completed one is the purger's to keep (docs/questionnaire/SPEC.md §5).
func TestRules_DeleteQuestionnaireResponsesOnceDue(t *testing.T) {
	for _, r := range retention.Rules() {
		if r.Table == "questionnaire_response" {
			if r.Then != retention.ThenDelete || r.Keep != 0 {
				t.Errorf("questionnaire_response then %q after %v, want %q at once", r.Then, r.Keep, retention.ThenDelete)
			}
			return
		}
	}
	t.Fatal("no rule for questionnaire_response")
}

// A file's row lives as long as its questionnaire response and goes with it through the foreign key; a
// rejected row, and an object without a row, are deleted 24 h on by binary's purger (docs/attachments/SPEC.md
// §5 "Retention").
func TestRules_DeleteRejectedBinariesAfterTwentyFourHours(t *testing.T) {
	for _, r := range retention.Rules() {
		if r.Table == "binary" {
			if r.Then != retention.ThenDelete || r.Keep != 24*time.Hour {
				t.Errorf("binary then %q after %v, want %q after 24 h", r.Then, r.Keep, retention.ThenDelete)
			}
			return
		}
	}
	t.Fatal("no rule for binary")
}

// The consultation's transcript, suggestions and drafts are kept with the clinical record, like the encounter
// (docs/plans/consult-intelligence.md, "Personal data").
func TestRules_KeepTheConsultationsTranscriptSuggestionsAndDrafts(t *testing.T) {
	want := map[string]bool{"transcript_segment": true, "suggestion": true, "document_draft": true}
	for _, r := range retention.Rules() {
		if !want[r.Table] {
			continue
		}
		if r.Then != retention.ThenKeep {
			t.Errorf("%s then %q, want %q", r.Table, r.Then, retention.ThenKeep)
		}
		delete(want, r.Table)
	}
	for table := range want {
		t.Errorf("no rule for %s", table)
	}
}

// The term list's reviews name the reviewer's account (00037): kept with the term, as the account is.
func TestRules_KeepTheTermListsReviewsWithTheirReviewer(t *testing.T) {
	want := map[string]bool{"term": true, "term_translation": true, "term_explanation": true}
	for _, r := range retention.Rules() {
		if !want[r.Table] {
			continue
		}
		if r.Then != retention.ThenKeep {
			t.Errorf("%s then %q, want %q", r.Table, r.Then, retention.ThenKeep)
		}
		delete(want, r.Table)
	}
	for table := range want {
		t.Errorf("no rule for %s", table)
	}
}

// The reviewer permission (00039) is kept while the operator grants it: revoking it, or deleting the account,
// deletes the row.
func TestRules_KeepTheReviewerPermissionWhileGranted(t *testing.T) {
	for _, r := range retention.Rules() {
		if r.Table == "term_reviewer" {
			if r.Then != retention.ThenKeep {
				t.Errorf("term_reviewer then %q, want %q", r.Then, retention.ThenKeep)
			}
			return
		}
	}
	t.Error("no rule for term_reviewer")
}

// The patient history's tables (00043, ADR 0014) are kept with the clinical record: allergies and medications as
// part of it (a patient's own unconfirmed entry is deleted when they remove it, by the service, not by this rule),
// the history consent and its selections as proof of what was shared when (Art. 7(1)).
func TestRules_KeepThePatientHistoryWithTheClinicalRecord(t *testing.T) {
	want := map[string]bool{"allergy_intolerance": true, "medication_statement": true, "history_consent": true, "history_consent_encounter": true}
	for _, r := range retention.Rules() {
		if !want[r.Table] {
			continue
		}
		if r.Then != retention.ThenKeep {
			t.Errorf("%s then %q, want %q", r.Table, r.Then, retention.ThenKeep)
		}
		delete(want, r.Table)
	}
	for table := range want {
		t.Errorf("no rule for %s", table)
	}
}
