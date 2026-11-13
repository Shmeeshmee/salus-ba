//go:build e2e

package e2e_test

import (
	"net/http"
	"slices"
	"testing"
)

// The All of Salus view, on in make e2e's stack (deployments/docker-compose.e2e.yml, docs/plans/analytics.md §2a): a
// clinic's admin sees the feature, sets the clinic's location in Clinic settings (Organization.address), reads the
// platform figures (every region a key, a count or null), and loses them once the clinic opts out (O6).
func TestPlatform_AnAdminSetsTheLocationReadsTheFiguresAndOptsOut(t *testing.T) {
	waitReady(t)
	adminAddr := email(t, "admin")
	clinic := field(t, salusctl(t, "create-clinic", "-name", "E2E synthetic clinic "+suffix(t), "-admin-email", adminAddr), "organization")
	admin, _ := enrolFromInvite(t, adminAddr)

	var me struct {
		Features []string `json:"features"`
	}
	admin.must(http.StatusOK, http.MethodGet, "/auth/me", nil, &me)
	if !slices.Contains(me.Features, "analytics_platform") {
		t.Fatalf("GET /auth/me features %v, want analytics_platform", me.Features)
	}

	const city = "https://salus.invalid/fhir/StructureDefinition/organization-city"
	const contributes = "https://salus.invalid/fhir/StructureDefinition/organization-analytics-contributes"
	update := func(changes map[string]any) map[string]any {
		var org map[string]any
		admin.must(http.StatusOK, http.MethodGet, "/fhir/R4/Organization/"+clinic, nil, &org)
		version, _ := org["meta"].(map[string]any)["versionId"].(string)
		for k, v := range changes {
			org[k] = v
		}
		var out map[string]any
		admin.mustWith(http.Header{"If-Match": {`W/"` + version + `"`}}, http.StatusOK, http.MethodPut, "/fhir/R4/Organization/"+clinic, org, &out)
		return out
	}
	org := update(map[string]any{"address": []map[string]any{{
		"state": "BA-09", "extension": []map[string]any{{"url": city, "valueCode": "Q1270190"}},
	}}})
	address, _ := org["address"].([]any)
	if len(address) != 1 || address[0].(map[string]any)["city"] != "Centar" {
		t.Errorf("the address = %v, want Centar in Kanton Sarajevo", org["address"])
	}

	var months struct {
		MinCount int `json:"minCount"`
	}
	admin.must(http.StatusOK, http.MethodGet, "/analytics/platform/months", nil, &months)
	if months.MinCount < 10 {
		t.Errorf("minCount %d, want at least 10", months.MinCount)
	}
	var cells struct {
		Rows []struct {
			Count *int   `json:"count"`
			Key   string `json:"key"`
		} `json:"rows"`
	}
	admin.must(http.StatusOK, http.MethodGet, "/analytics/platform/cells?table=gd&from=2026-01&to=2026-10&by=region", nil, &cells)
	if len(cells.Rows) != 12 {
		t.Errorf("%d regions, want every one of the 12 as a key", len(cells.Rows))
	}

	update(map[string]any{"extension": []map[string]any{{"url": contributes, "valueBoolean": false}}})
	if status := admin.do(http.MethodGet, "/analytics/platform/months", nil, nil); status != http.StatusForbidden {
		t.Errorf("an opted-out clinic's admin = %d, want 403", status)
	}
}
