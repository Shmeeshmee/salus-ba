package retention

import "time"

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
			Basis: "The patient's files attached to a questionnaire answer (GDPR Art. 9(2)(h), with the consent for sharing before care): each row lives exactly as long as its questionnaire response and is deleted with it (foreign key) or with its answer; a rejected row is deleted 24 h after its scan, and a stored object without a row (a deleted file's ciphertext, an upload that never reached its answer) 24 h after it was written; needs a legal decision before production (docs/attachments/SPEC.md sections 4 and 5)",
			Keep:  day, // after scanned_at of a rejected row; after an object's last write
			Then:  ThenDelete,
		},
	}, consultIntelligenceRules()...)
}

// consultIntelligenceRules are the rules of the consultation's transcript, suggestions and drafts
// (docs/plans/consult-intelligence.md): all kept with the clinical record, like the encounter.
func consultIntelligenceRules() []Rule {
	return []Rule{
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
			Table: "document_draft",
			Basis: "The document the practitioner writes about the patient during the consultation (GDPR Art. 9(2)(h)); kept with the clinical record, a discarded one as entered-in-error; needs a legal decision before production (docs/plans/consult-intelligence.md, \"Personal data\")",
			Then:  ThenKeep,
		},
	}
}
