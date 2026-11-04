package retention

import "time"

// day is a calendar day for retention periods; a period in days is rounded up so it is never short.
const day = 24 * time.Hour

// Rules returns the rule of every table with personal data. The periods are defaults that need a legal
// decision before production (docs/accounts/SPEC.md section 4).
func Rules() []Rule {
	return []Rule{
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
	}
}
