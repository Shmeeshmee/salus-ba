package api_test

import (
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/api"
)

// §2a: with ANALYTICS_PLATFORM_ENABLED off, the All of Salus routes are not mounted (404) and GET /auth/me lists no
// feature; on, staff of a contributing clinic read the figures, a patient may not, and an opted-out clinic's staff
// get analytics_not_contributing.
func TestNew_ThePlatformViewFollowsItsFlag(t *testing.T) {
	for _, on := range []bool{false, true} {
		cfg := testConfig(t)
		cfg.Analytics.PlatformEnabled, cfg.Analytics.MinCount = on, 10
		app, err := api.New(t.Context(), cfg, testLiveKit(), testObjectStore(t), testDrafter(), testSummarizer(), slog.New(slog.DiscardHandler))
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		t.Cleanup(app.Close)
		owner := ownerPool(t, cfg)
		d := newDirectory(t, app.Handler, owner)
		me := d.admin.do(http.MethodGet, "/auth/me", "")
		listed := strings.Contains(me.Body.String(), `"features":["analytics_platform"]`)
		months := d.practitioner.do(http.MethodGet, "/analytics/platform/months", "")
		if !on {
			if listed || months.Code != http.StatusNotFound {
				t.Errorf("off: /auth/me %s, months %d; want no feature and 404", me.Body, months.Code)
			}
			continue
		}
		if !listed || months.Code != http.StatusOK || !strings.Contains(months.Body.String(), `"minCount":10`) {
			t.Errorf("on: /auth/me %s, months %d %s; want the feature and 200", me.Body, months.Code, months.Body)
		}
		if rec := d.patient.do(http.MethodGet, "/analytics/platform/months", ""); rec.Code != http.StatusForbidden {
			t.Errorf("a patient = %d, want 403", rec.Code)
		}
		if _, err := owner.Exec(t.Context(), `UPDATE organization SET analytics_contributes = false`); err != nil {
			t.Fatal(err)
		}
		rec := d.admin.do(http.MethodGet, "/analytics/platform/cells?table=gd&from=2026-09&to=2026-10&by=region", "")
		if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "analytics_not_contributing") {
			t.Errorf("an opted-out clinic = %d %s, want 403 analytics_not_contributing", rec.Code, rec.Body)
		}
	}
}
