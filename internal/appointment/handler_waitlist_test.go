package appointment_test

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
)

// stubWaitlist answers with what it is given and keeps what it was asked.
type stubWaitlist struct {
	err     error
	filter  appointment.WaitlistFilter
	appt    appointment.Appointment
	caller  appointment.Caller
	args    []string
	entries []appointment.WaitlistEntry
	in      appointment.NewEntry
	entry   appointment.WaitlistEntry
	clinic  appointment.ClinicEntry
}

func (s *stubWaitlist) Join(_ context.Context, c appointment.Caller, in appointment.NewEntry) (appointment.WaitlistEntry, error) {
	s.caller, s.in, s.args = c, in, []string{"Join"}
	return s.entry, s.err
}

func (s *stubWaitlist) MyWaitlist(_ context.Context, c appointment.Caller) ([]appointment.WaitlistEntry, error) {
	s.caller, s.args = c, []string{"MyWaitlist"}
	return s.entries, s.err
}

func (s *stubWaitlist) ClinicWaitlist(_ context.Context, c appointment.Caller, f appointment.WaitlistFilter) ([]appointment.ClinicEntry, error) {
	s.caller, s.filter, s.args = c, f, []string{"ClinicWaitlist"}
	return []appointment.ClinicEntry{s.clinic}, s.err
}

func (s *stubWaitlist) Withdraw(_ context.Context, c appointment.Caller, id string) error {
	s.caller, s.args = c, []string{"Withdraw", id}
	return s.err
}

func (s *stubWaitlist) SetPriority(_ context.Context, c appointment.Caller, id string, p int) (appointment.ClinicEntry, error) {
	s.caller, s.args = c, []string{"SetPriority", id, string(rune('0' + p/10))}
	return s.clinic, s.err
}

func (s *stubWaitlist) Accept(_ context.Context, c appointment.Caller, id string) (appointment.Appointment, error) {
	s.caller, s.args = c, []string{"Accept", id}
	return s.appt, s.err
}

func (s *stubWaitlist) Decline(_ context.Context, c appointment.Caller, id string) error {
	s.caller, s.args = c, []string{"Decline", id}
	return s.err
}

const entryID = "e1000000-0000-4000-8000-000000000001"

var (
	theEntry = appointment.WaitlistEntry{
		ID: entryID, PatientID: patientA, OrganizationID: clinicA, PractitionerRoleID: roleA1, Earliest: day(now), Latest: day(start),
		Times: appointment.Times{Start: 480, End: 1440}, Status: appointment.WaitlistActive, Created: now, Priority: 5,
		Offer: appointment.Offer{ID: "0f000000-0000-4000-8000-000000000001", Start: start, End: start.Add(30 * time.Minute), ExpiresAt: now.Add(15 * time.Minute)},
	}
	entryJSON = `{"created_at":"2030-01-01T08:00:00Z","times":{"start":"08:00","end":"24:00"},"offer":{"slot_start":"2030-01-07T09:00:00Z",` +
		`"slot_end":"2030-01-07T09:30:00Z","expires_at":"2030-01-01T08:15:00Z","id":"0f000000-0000-4000-8000-000000000001"},"id":"` + entryID +
		`","organization_id":"` + clinicA + `","practitioner_role_id":"` + roleA1 + `","earliest":"2030-01-01","latest":"2030-01-07","status":"active"}`
)

func serveWaitlist(t *testing.T, svc *stubWaitlist, c appointment.Caller, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	appointment.NewWaitlistHandler(svc, as(c), slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))).Routes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body)))
	return rec
}

func TestWaitlistHandler_Joins(t *testing.T) {
	t.Parallel()
	svc := &stubWaitlist{entry: theEntry}
	rec := serveWaitlist(t, svc, me, http.MethodPost, "/waitlist",
		`{"organization_id":"`+clinicA+`","practitioner_role_id":"`+roleA1+`","earliest":"2030-01-01","latest":"2030-01-07","times":{"start":"08:00","end":"24:00"}}`)
	if rec.Code != http.StatusCreated || rec.Body.String() != entryJSON+"\n" || rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("POST /waitlist = %d %s,\nwant 201 %s", rec.Code, rec.Body, entryJSON)
	}
	if want := (appointment.NewEntry{OrganizationID: clinicA, PractitionerRoleID: roleA1, Earliest: day(now), Latest: day(start),
		Times: appointment.Times{Start: 480, End: 1440}}); svc.in != want || svc.caller != me {
		t.Errorf("asked %+v as %+v, want %+v", svc.in, svc.caller, want)
	}
	for name, body := range map[string]string{
		"no clinic":      `{"specialty":"394814009","earliest":"2030-01-01","latest":"2030-01-07"}`,
		"a bad date":     `{"organization_id":"` + clinicA + `","specialty":"394814009","earliest":"1 Jan","latest":"2030-01-07"}`,
		"a bad time":     `{"organization_id":"` + clinicA + `","specialty":"394814009","earliest":"2030-01-01","latest":"2030-01-07","times":{"start":"8","end":"09:00"}}`,
		"an end of 0:00": `{"organization_id":"` + clinicA + `","specialty":"394814009","earliest":"2030-01-01","latest":"2030-01-07","times":{"start":"08:00","end":"00:00"}}`,
		"an extra field": `{"organization_id":"` + clinicA + `","specialty":"394814009","earliest":"2030-01-01","latest":"2030-01-07","priority":9}`,
	} {
		if rec := serveWaitlist(t, &stubWaitlist{}, me, http.MethodPost, "/waitlist", body); rec.Code != http.StatusBadRequest {
			t.Errorf("POST /waitlist with %s = %d, want 400", name, rec.Code)
		}
	}
}

func TestWaitlistHandler_MapsTheRefusals(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		err    error
		code   string
		status int
	}{
		{appointment.ErrInvalidEntry, "invalid_request", http.StatusBadRequest},
		{appointment.ErrForbidden, "forbidden", http.StatusForbidden},
		{appointment.ErrConsentRequired, "consent_required", http.StatusConflict},
		{appointment.ErrWaitlistOff, "waitlist_off", http.StatusConflict},
		{appointment.ErrTooManyEntries, "too_many_entries", http.StatusConflict},
		{appointment.ErrOfferExpired, "offer_expired", http.StatusConflict},
		{appointment.ErrSlotUnavailable, "slot_unavailable", http.StatusConflict},
		{appointment.ErrNotFound, "not_found", http.StatusNotFound},
	} {
		rec := serveWaitlist(t, &stubWaitlist{err: tt.err}, me, http.MethodPost, "/waitlist",
			`{"organization_id":"`+clinicA+`","specialty":"394814009","earliest":"2030-01-01","latest":"2030-01-07"}`)
		if rec.Code != tt.status || rec.Body.String() != `{"error":"`+tt.code+`"}`+"\n" {
			t.Errorf("%v: %d %s, want %d %s", tt.err, rec.Code, rec.Body, tt.status, tt.code)
		}
	}
	rec := serveWaitlist(t, &stubWaitlist{err: errBackend}, me, http.MethodGet, "/me/waitlist", "")
	if rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), "database") {
		t.Errorf("GET /me/waitlist failing = %d %s, want a 500 without the error", rec.Code, rec.Body)
	}
}

func TestWaitlistHandler_ListsThePatientsAndTheClinicsEntries(t *testing.T) {
	t.Parallel()
	svc := &stubWaitlist{entries: []appointment.WaitlistEntry{theEntry}}
	if rec := serveWaitlist(t, svc, me, http.MethodGet, "/me/waitlist", ""); rec.Code != http.StatusOK || rec.Body.String() != `{"entries":[`+entryJSON+`]}`+"\n" {
		t.Errorf("GET /me/waitlist = %d %s", rec.Code, rec.Body)
	}
	svc = &stubWaitlist{clinic: appointment.ClinicEntry{Entry: theEntry, Patient: nameA, Named: true}}
	rec := serveWaitlist(t, svc, admin, http.MethodGet, "/clinic/waitlist?practitioner_role="+roleA1+"&status=active", "")
	want := `{"entries":[{"patient_id":"` + patientA + `","patient_name":"Ada Example","entry":` + entryJSON + `,"priority":5}]}` + "\n"
	if rec.Code != http.StatusOK || rec.Body.String() != want || svc.filter != (appointment.WaitlistFilter{PractitionerRoleID: roleA1, Status: appointment.WaitlistActive}) {
		t.Errorf("GET /clinic/waitlist = %d %s (filter %+v),\nwant %s", rec.Code, rec.Body, svc.filter, want)
	}
	svc = &stubWaitlist{clinic: appointment.ClinicEntry{Entry: theEntry}}
	if rec := serveWaitlist(t, svc, pract, http.MethodGet, "/clinic/waitlist", ""); strings.Contains(rec.Body.String(), "patient_name") {
		t.Errorf("an entry without a name sent one: %s", rec.Body)
	}
	if rec := serveWaitlist(t, &stubWaitlist{}, admin, http.MethodGet, "/clinic/waitlist?status=lost", ""); rec.Code != http.StatusBadRequest {
		t.Errorf("an unknown status = %d, want 400", rec.Code)
	}
	if rec := serveWaitlist(t, &stubWaitlist{err: appointment.ErrForbidden}, me, http.MethodGet, "/clinic/waitlist", ""); rec.Code != http.StatusNotFound {
		t.Errorf("a patient's GET /clinic/waitlist = %d, want 404", rec.Code)
	}
}

func TestWaitlistHandler_ActsOnEntriesAndOffers(t *testing.T) {
	t.Parallel()
	svc := &stubWaitlist{appt: bookedA, clinic: appointment.ClinicEntry{Entry: theEntry}}
	for _, tt := range []struct {
		method string
		path   string
		body   string
		caller appointment.Caller
		want   []string
		status int
	}{
		{caller: me, method: http.MethodDelete, path: "/waitlist/" + entryID, body: "", status: http.StatusNoContent, want: []string{"Withdraw", entryID}},
		{caller: me, method: http.MethodPost, path: "/waitlist/offers/of/accept", body: "", status: http.StatusCreated, want: []string{"Accept", "of"}},
		{caller: me, method: http.MethodPost, path: "/waitlist/offers/of/decline", body: "", status: http.StatusNoContent, want: []string{"Decline", "of"}},
		{caller: admin, method: http.MethodPut, path: "/clinic/waitlist/" + entryID + "/priority", body: `{"priority":40}`, status: http.StatusOK, want: []string{"SetPriority", entryID, "4"}},
	} {
		rec := serveWaitlist(t, svc, tt.caller, tt.method, tt.path, tt.body)
		if rec.Code != tt.status || strings.Join(svc.args, " ") != strings.Join(tt.want, " ") {
			t.Errorf("%s %s = %d %s (asked %v), want %d %v", tt.method, tt.path, rec.Code, rec.Body, svc.args, tt.status, tt.want)
		}
	}
	if rec := serveWaitlist(t, svc, me, http.MethodPost, "/waitlist/offers/of/accept", ""); !strings.Contains(rec.Body.String(), `"slotId":"`+slotFree+`"`) {
		t.Errorf("the acceptance's body %s, want the Appointment", rec.Body)
	}
	if rec := serveWaitlist(t, svc, admin, http.MethodPut, "/clinic/waitlist/"+entryID+"/priority", `{}`); rec.Code != http.StatusBadRequest {
		t.Errorf("a priority missing = %d, want 400", rec.Code)
	}
	for path, method := range map[string]string{"/waitlist/offers/of/accept": http.MethodPost, "/clinic/waitlist/" + entryID + "/priority": http.MethodPut} {
		if rec := serveWaitlist(t, &stubWaitlist{err: appointment.ErrForbidden}, admin, method, path, `{"priority":1}`); rec.Code != http.StatusNotFound {
			t.Errorf("%s %s refused = %d, want 404", method, path, rec.Code)
		}
	}
	unauthenticated := func(context.Context) (appointment.Caller, bool, error) { return appointment.Caller{}, false, nil }
	mux := http.NewServeMux()
	appointment.NewWaitlistHandler(svc, unauthenticated, slog.New(slog.DiscardHandler)).Routes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/me/waitlist", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("GET /me/waitlist signed out = %d, want 401", rec.Code)
	}
}
