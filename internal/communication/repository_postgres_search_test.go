package communication_test

import (
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/communication"
)

// Search returns only messages the user may see (SAL-30, acceptance), on Postgres's text search and row-level
// security: the words case and accents aside; across the caller's conversations a patient's own, a practitioner's
// after their history_from_seq; nothing for staff once the patient revokes the consent, or for another patient.
func TestPostgres_SearchReturnsOnlyTheMessagesTheCallerMayRead(t *testing.T) {
	t.Parallel()
	f, owner := pgFixture(t)
	c := f.open(t)
	for i, body := range []string{"Synthetic mučnina since Monday", "Synthetic placeholder", "Mucnina again, synthetic"} {
		f.clock.set(start.Add(time.Duration(i) * time.Minute))
		f.send(t, me, c.ID, body)
	}
	if _, err := owner.Exec(t.Context(), `INSERT INTO conversation_participant (conversation_id, practitioner_role_id, role, added_by, history_from_seq)
		VALUES ($1, $2, 'practitioner', $3, 2)`, c.ID, roleA2, pract.AccountID); err != nil {
		t.Fatal(err)
	}
	both := []string{"Mucnina again, synthetic", "Synthetic mučnina since Monday"}
	type check struct {
		name string
		got  func() ([]string, error)
		want []string
	}
	search := func(who communication.Caller, q string) func() ([]string, error) {
		return func() ([]string, error) {
			rs, err := f.svc.Search(t.Context(), who, c.ID, q)
			return bodies(rs), err
		}
	}
	searchAll := func(who communication.Caller, q string) func() ([]string, error) {
		return func() ([]string, error) {
			rs, err := f.svc.SearchAll(t.Context(), who, q)
			return bodies(rs), err
		}
	}
	for _, tt := range []check{
		{"the patient in the conversation, without the diacritics", search(me, "MUCN"), both},
		{"the treating practitioner across their conversations", searchAll(pract, "mučnina"), both},
		{"the patient across theirs", searchAll(me, "mucnina monday"), both[1:]},
		{"a practitioner who joined after message 2, in it", search(pract2, "mucnina"), both[:1]},
		{"a practitioner who joined after message 2, across", searchAll(pract2, "mucnina"), both[:1]},
		{"another patient across theirs", searchAll(other, "mucnina"), nil},
	} {
		got, err := tt.got()
		if err != nil || !slices.Equal(got, tt.want) {
			t.Errorf("%s: %v, %v; want %v", tt.name, got, err, tt.want)
		}
	}
	if _, err := owner.Exec(t.Context(), `UPDATE consent SET status = 'inactive', revoked_at = now() WHERE patient_id = $1`, patientA); err != nil {
		t.Fatal(err)
	}
	if got, err := f.svc.SearchAll(t.Context(), pract, "mucnina"); err != nil || len(got) != 0 {
		t.Errorf("the practitioner after the revocation: %v, %v; want none", bodies(got), err)
	}
	if got, err := f.svc.SearchAll(t.Context(), me, "mucnina"); err != nil || len(got) != 2 {
		t.Errorf("the patient after the revocation: %v, %v; want their own two", bodies(got), err)
	}
}
