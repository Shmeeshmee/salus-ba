package organization_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/organization"
)

const (
	cityURL        = "https://salus.invalid/fhir/StructureDefinition/organization-city"
	contributesURL = "https://salus.invalid/fhir/StructureDefinition/organization-analytics-contributes"
)

// The clinic's location is Organization.address: the leaf region's ISO 3166-2 code as the state, the city's Latin
// name as the city, and the city's Wikidata id in the organization-city extension (a name is not unique); whether
// it contributes to the platform figures is the organization-analytics-contributes extension (O6).
func TestHandlerRead_SendsTheLocationAndTheContribution(t *testing.T) {
	o := clinic(3)
	o.Region, o.City, o.CityName, o.LocationSince = "BA-09", "Q1270190", "Centar", time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	o.AnalyticsContributes = true
	rec := newOrgServer(&stubClinics{org: o}).do(t, http.MethodGet, clinicPath, "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d %s, want 200", rec.Code, rec.Body)
	}
	assertGolden(t, "organization_location.json", rec.Body.Bytes())
}

// The public directory shows neither the location nor the contribution.
func TestHandlerSearch_ShowsNoContribution(t *testing.T) {
	o := clinic(3)
	o.Listed, o.AnalyticsContributes = true, true
	rec := newOrgServer(&stubClinics{all: []organization.Organization{o}}).do(t, http.MethodGet, "/fhir/R4/Organization?listed=true", "", "")
	if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), contributesURL) || strings.Contains(rec.Body.String(), "address") {
		t.Errorf("search = %d %s, want no contribution and no address", rec.Code, rec.Body)
	}
}

func TestHandlerUpdate_SetsTheLocationAndTheContribution(t *testing.T) {
	address := `"address":[{"state":"BA-09","city":"Centar","extension":[{"url":"` + cityURL + `","valueCode":"Q1270190"}]}]`
	contributes := `"extension":[{"url":"` + contributesURL + `","valueBoolean":false}]`
	s := newOrgServer(&stubClinics{org: clinic(4)})
	body := `{"resourceType":"Organization","id":"` + clinicA + `","name":"Synthetic clinic",` + address + `,` + contributes + `}`
	rec := s.do(t, http.MethodPut, clinicPath, `W/"3"`, body)
	ch := s.svc.changes
	if rec.Code != http.StatusOK || ch.Region == nil || *ch.Region != "BA-09" || ch.City == nil || *ch.City != "Q1270190" ||
		ch.AnalyticsContributes == nil || *ch.AnalyticsContributes {
		t.Errorf("status = %d %s with %+v, want 200 and the location and the opt-out", rec.Code, rec.Body, ch)
	}
	s = newOrgServer(&stubClinics{org: clinic(4)})
	rec = s.do(t, http.MethodPut, clinicPath, `W/"3"`, renameBody(clinicA))
	if ch := s.svc.changes; rec.Code != http.StatusOK || ch.Region != nil || ch.City != nil || ch.AnalyticsContributes != nil {
		t.Errorf("left out: status = %d with %+v, want both kept", rec.Code, ch)
	}
	for name, a := range map[string]string{
		"no state":        `"address":[{"extension":[{"url":"` + cityURL + `","valueCode":"Q1270190"}]}]`,
		"no city":         `"address":[{"state":"BA-09"}]`,
		"two addresses":   `"address":[{"state":"BA-09","extension":[{"url":"` + cityURL + `","valueCode":"Q1"}]},{"state":"BA-01","extension":[{"url":"` + cityURL + `","valueCode":"Q2"}]}]`,
		"a boolean city":  `"address":[{"state":"BA-09","extension":[{"url":"` + cityURL + `","valueBoolean":true}]}]`,
		"a code opt-out":  `"extension":[{"url":"` + contributesURL + `","valueCode":"false"}]`,
		"two cities":      `"address":[{"state":"BA-09","extension":[{"url":"` + cityURL + `","valueCode":"Q1"},{"url":"` + cityURL + `","valueCode":"Q2"}]}]`,
		"an empty region": `"address":[{"state":"","extension":[{"url":"` + cityURL + `","valueCode":"Q1"}]}]`,
	} {
		s := newOrgServer(&stubClinics{})
		body := `{"resourceType":"Organization","id":"` + clinicA + `","name":"Synthetic clinic",` + a + `}`
		if rec := s.do(t, http.MethodPut, clinicPath, `W/"3"`, body); rec.Code != http.StatusBadRequest || s.svc.calls != 0 {
			t.Errorf("%s: status = %d after %d calls, want 400 and none", name, rec.Code, s.svc.calls)
		}
	}
}

// A location the service refuses is a 400 with the issue code code-invalid.
func TestHandlerUpdate_AnswersAnInvalidLocation(t *testing.T) {
	s := newOrgServer(&stubClinics{err: organization.ErrInvalidLocation})
	body := `{"resourceType":"Organization","id":"` + clinicA + `","name":"Synthetic clinic","address":[{"state":"BA-01","extension":[{"url":"` +
		cityURL + `","valueCode":"Q1270190"}]}]}`
	rec := s.do(t, http.MethodPut, clinicPath, `W/"3"`, body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d %s, want 400", rec.Code, rec.Body)
	}
	assertGolden(t, "invalid_location.json", rec.Body.Bytes())
}
