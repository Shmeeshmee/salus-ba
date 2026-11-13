package retention

import (
	"slices"
	"time"
)

// day is a calendar day for retention periods; a period in days is rounded up so it is never short.
const day = 24 * time.Hour

// Rules returns the rule of every table with personal data. The periods are defaults that need a legal
// decision before production (docs/accounts/SPEC.md section 4).
func Rules() []Rule {
	return append([]Rule{
		{
			Table: "audit_event",
			Basis: "Accountability and security of processing (GDPR Art. 5(2), 32; Art. 6(1)(c), (f))",
			Keep:  3653 * day, // ten years, leap days included
			Then:  ThenRemoveRange,
		},
		{
			Table: "account",
			Basis: "Performance of the contract the user signed up for (GDPR Art. 6(1)(b)); closure and erasure come with the subject-rights feature",
			Then:  ThenKeep,
		},
		{
			Table: "session",
			Basis: "Performance of the contract (GDPR Art. 6(1)(b)) while signed in; then security of processing (Art. 32) for 30 days",
			Keep:  30 * day, // after revoked_at or expires_at
			Then:  ThenDelete,
		},
		{
			Table: "invite",
			Basis: "The clinic's employment relationship and the contract with the clinic (GDPR Art. 6(1)(b), (f)) until the invite is accepted or expires; then accountability (Art. 5(2)) for 30 days",
			Keep:  30 * day, // after accepted_at or expires_at
			Then:  ThenDelete,
		},
		{
			Table: "practitioner",
			Basis: "The clinic's employment relationship and the contract with the clinic (GDPR Art. 6(1)(b), (f)); after leaving, the medical-record retention that references the practitioner",
			Then:  ThenRestrict,
		},
		{
			Table: "practitioner_role",
			Basis: "The clinic's employment relationship and the contract with the clinic (GDPR Art. 6(1)(b), (f)); after leaving, the medical-record retention that references the role",
			Then:  ThenRestrict,
		},
		{
			Table: "patient",
			Basis: "Provision of health care (GDPR Art. 9(2)(h)) and the contract with the patient (Art. 6(1)(b)) while the account exists; erasure comes with the subject-rights feature (docs/booking/SPEC.md section 4)",
			Then:  ThenKeep,
		},
		{
			Table: "consent",
			Basis: "Proof of the patient's decision (GDPR Art. 7(1)), kept with the clinical record (Art. 9(2)(h)); needs a legal decision before production (docs/booking/SPEC.md section 4)",
			Then:  ThenKeep,
		},
		{
			Table: "appointment",
			Basis: "The clinic's care record (GDPR Art. 9(2)(h)) and the contract with the patient (Art. 6(1)(b)); kept with the clinical record, needs a legal decision before production (docs/booking/SPEC.md sections 3 and 4)",
			Then:  ThenKeep,
		},
		{
			Table: "schedule",
			Basis: "The clinic's employment relationship and its legitimate interest in being booked (GDPR Art. 6(1)(b), (f)) while the role is active; after the practitioner leaves, restricted like practitioner_role (docs/booking/SPEC.md section 4)",
			Then:  ThenRestrict,
		},
		{
			Table: "slot",
			Basis: "Offering bookable times (GDPR Art. 6(1)(b), (f)); a free slot is deleted 30 days after its end, a busy one is kept with its appointment (docs/booking/SPEC.md section 4)",
			Keep:  30 * day, // after the end of a free slot
			Then:  ThenDelete,
		},
		{
			Table: "video_session",
			Basis: "Running the consultation (GDPR Art. 9(2)(h), with the consent for sharing before care) until it ends; then 30 days for operations and disputes; needs a legal decision before production (docs/consultation/SPEC.md sections 4 and 5)",
			Keep:  30 * day, // after ended_at, or after the join window's close for a session that never ended
			Then:  ThenDelete,
		},
		{
			Table: "encounter",
			Basis: "The clinic's care record that the consultation took place and when (GDPR Art. 9(2)(h)); kept with the clinical record, needs a legal decision before production (docs/consultation/SPEC.md sections 4 and 5)",
			Then:  ThenKeep,
		},
		{
			Table: "questionnaire_response",
			Basis: "Preparing the appointment (GDPR Art. 9(2)(h), with the consent for sharing before care) until it starts; an unsubmitted response is then deleted, a submitted one kept with the clinical record; needs a legal decision before production (docs/questionnaire/SPEC.md sections 4 and 5)",
			Then:  ThenDelete, // unsubmitted ones once due_at has passed; completed ones are kept
		},
		{
			Table: "binary",
			Basis: "The patient's files attached to a questionnaire answer (GDPR Art. 9(2)(h), with the consent for sharing before care): each row lives exactly as long as its questionnaire response and is deleted with it (foreign key) or with its answer. " +
				"The files of a chat message (SAL-30; care between visits, Art. 9(2)(h), with the care consent for the clinic's access) are part of the clinical record with their message and kept with it, never deleted while attached (a trigger refuses it, as for the message); a chat upload never attached to a message is deleted 24 h after its upload. " +
				"A rejected row is deleted 24 h after its scan, and a stored object without a row (a deleted file's ciphertext or thumbnail, an upload that never reached its answer or message) 24 h after it was written; needs a legal decision before production (docs/attachments/SPEC.md sections 4 and 5, docs/plans/chat.md)",
			Keep: day, // after scanned_at of a rejected row; after created_at of a chat upload never attached; after an object's last write
			Then: ThenDelete,
		},
	}, slices.Concat(consultIntelligenceRules(), termReviewRules(), chatRules(), chatNotificationRules(), historyRules(), appointmentRules())...)
}

// chatRules are the rules of the chat between the patient and their clinic's doctors (docs/plans/chat.md, SAL-27):
// the messages are part of the clinical record, never deleted by anyone (ADR 0012), and kept with it.
func chatRules() []Rule {
	basis := "Care between visits (GDPR Art. 9(2)(h), with the care consent for the clinic's access): the messages, their " +
		"earlier versions, who takes part and their read position are part of the clinical record and kept with it, like " +
		"the encounter and the transcript; never deleted by a user or the app; needs a legal decision before production " +
		"(docs/plans/chat.md \"Personal data\")"
	return []Rule{
		{Table: "conversation", Basis: basis, Then: ThenKeep},
		{Table: "conversation_participant", Basis: basis, Then: ThenKeep},
		{Table: "communication", Basis: basis, Then: ThenKeep},
		{Table: "communication_revision", Basis: basis, Then: ThenKeep},
		{
			Table: "chat_consent",
			Basis: "Proof of the patient's decision to let a doctor of another clinic into their conversation (GDPR Art. 7(1), Art. 9(2)(a)), kept with the clinical record like consent; needs a legal decision before production (docs/plans/chat.md, SAL-28)",
			Then:  ThenKeep,
		},
	}
}

// chatNotificationRules are the rules of the chat's notifications (docs/plans/chat.md, SAL-29).
func chatNotificationRules() []Rule {
	return []Rule{
		{
			Table: "push_device",
			Basis: "Telling the patient of a new message (GDPR Art. 6(1)(b), the contract with the patient) while the app is " +
				"registered on the device: deleted when the app unregisters, when the platform says the token is gone, and " +
				"with the account (foreign key; erasure comes with the subject-rights feature)",
			Then: ThenKeep,
		},
		{
			Table: "webhook_endpoint",
			Basis: "The clinic's integration (GDPR Art. 6(1)(b), (f), the contract with the clinic): who created it, as an " +
				"account id; kept while the clinic keeps the endpoint, deleted by its admin",
			Then: ThenKeep,
		},
		{
			Table: "webhook_dead_letter",
			Basis: "Operating the clinic's integration (GDPR Art. 6(1)(f)): the ids of an event its endpoint did not take, " +
				"for its admin and support to look into, for 30 days; deleted with the endpoint",
			Keep: 30 * day, // after created_at, the last attempt
			Then: ThenDelete,
		},
	}
}

// termReviewRules are the rules of the term list's reviews (docs/plans/v1-finish.md "Contract"): the reference
// data itself holds nothing personal; reviewed_by names the reviewer's account (owner, 2026-11-10).
func termReviewRules() []Rule {
	basis := "Accountability for the reviewed medical term list the clinics work with (GDPR Art. 5(2), Art. 6(1)(f)): " +
		"who approved or flagged an entry, as an account id; kept with the term while the account exists, the reference " +
		"cleared with the account's erasure (subject-rights feature); needs a legal decision before production"
	return []Rule{
		{Table: "term", Basis: basis, Then: ThenKeep},
		{Table: "term_translation", Basis: basis, Then: ThenKeep},
		{Table: "term_explanation", Basis: basis, Then: ThenKeep},
		{
			Table: "term_reviewer",
			Basis: "The operator's grant of the term list's review permission to a staff account (GDPR Art. 6(1)(b), (f)): kept while granted; revoking it deletes the row, and so does deleting the account (foreign key)",
			Then:  ThenKeep,
		},
	}
}

// consultIntelligenceRules are the rules of the consultation's transcript, suggestions and drafts
// (docs/plans/consult-intelligence.md), and of the patient's consent to them (SAL-12): all kept with the clinical
// record, like the encounter and the care consent.
func consultIntelligenceRules() []Rule {
	return []Rule{
		{
			Table: "transcription_consent",
			Basis: "Proof of the patient's decision on transcription, translation and AI assistance in their consultations (GDPR Art. 7(1)), kept with the clinical record like consent (Art. 9(2)(h)); needs a legal decision before production (docs/plans/v1-finish.md, SAL-12)",
			Then:  ThenKeep,
		},
		{
			Table: "transcript_segment",
			Basis: "The record of what was said in the consultation, with its translation (GDPR Art. 9(2)(h), with the consent for sharing before care); kept with the clinical record like the encounter, needs a legal decision before production (docs/plans/consult-intelligence.md, \"Personal data\")",
			Then:  ThenKeep,
		},
		{
			Table: "suggestion",
			Basis: "What was suggested to the practitioner during the consultation and what they did with it, for their accountability (GDPR Art. 9(2)(h), Art. 5(2)); kept with the clinical record, needs a legal decision before production (docs/plans/consult-intelligence.md, \"Personal data\")",
			Then:  ThenKeep,
		},
		{
			Table: "condition",
			Basis: "The diagnoses of the signed documents, part of the clinical record (GDPR Art. 9(2)(h)); kept with the record like document_draft, the source of the clinic's statistics; needs a legal decision before production (docs/plans/analytics.md §7)",
			Then:  ThenKeep,
		},
		{
			Table: "document_draft",
			Basis: "The document the practitioner writes about the patient during the consultation (GDPR Art. 9(2)(h)); kept with the clinical record, a discarded one as entered-in-error; needs a legal decision before production (docs/plans/consult-intelligence.md, \"Personal data\")",
			Then:  ThenKeep,
		},
	}
}

// historyRules are the rules of the patient history's tables (00043, docs/plans/patient-history.md, ADR 0014): the
// allergies and medications are part of the clinical record, the history consent and its selections the proof of
// what the patient shared with whom and when.
func historyRules() []Rule {
	return []Rule{
		{
			Table: "allergy_intolerance",
			Basis: "The patient's allergies, safety-critical for their care (GDPR Art. 9(2)(h)); kept with the clinical record, a mistaken entry as entered-in-error, and the patient's own unconfirmed entry deleted when they remove it; needs a legal decision before production (docs/plans/patient-history.md, \"Personal data\")",
			Then:  ThenKeep,
		},
		{
			Table: "medication_statement",
			Basis: "The patient's current medications, for their care (GDPR Art. 9(2)(h)); kept as allergy_intolerance is; needs a legal decision before production (docs/plans/patient-history.md, \"Personal data\")",
			Then:  ThenKeep,
		},
		{
			Table: "history_consent",
			Basis: "Proof of the patient's decision to share visits from other clinics with a clinic (GDPR Art. 7(1)), kept with the clinical record like consent (Art. 9(2)(h)); needs a legal decision before production (docs/plans/patient-history.md)",
			Then:  ThenKeep,
		},
		{
			Table: "history_consent_encounter",
			Basis: "Proof of which visits the patient shared with the clinic and when (GDPR Art. 7(1)), kept with its history consent; needs a legal decision before production (docs/plans/patient-history.md)",
			Then:  ThenKeep,
		},
	}
}

// appointmentRules are the rules of the appointments epic (docs/plans/appointments.md, SAL-32, ADR 0015); the periods
// are placeholders that need a legal decision before production (the plan's Q9).
func appointmentRules() []Rule {
	return []Rule{
		{
			Table: "appointment_change",
			Basis: "The clinic's care record and its accountability (GDPR Art. 9(2)(h), Art. 5(2)): who confirmed, rescheduled, " +
				"marked or corrected an appointment, from what to what; append-only and kept with the appointment; needs a legal " +
				"decision before production (docs/plans/appointments.md \"Personal data\")",
			Then: ThenKeep,
		},
		{
			Table: "appointment_action_token",
			Basis: "The contract with the patient (GDPR Art. 6(1)(b)): a reminder's one-time link to confirm or cancel, stored as " +
				"its hash; deleted 30 days after it expires (the appointment's start), and with the account; needs a legal " +
				"decision before production (docs/plans/appointments.md \"Personal data\")",
			Keep: 30 * day, // after expires_at
			Then: ThenDelete,
		},
	}
}
