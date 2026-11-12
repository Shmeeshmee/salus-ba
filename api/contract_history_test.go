package api_test

import (
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/allergyintolerance"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/internal/history"
	"github.com/Shmeeshmee/salus/internal/medicationstatement"
)

// The patient history's schemas (docs/plans/patient-history.md "Contract", generated into the web and mobile
// clients) have the Go types' fields, required exactly when Go always encodes them.
func TestHistorySchemas_MatchTheirGoTypes(t *testing.T) {
	t.Parallel()
	s := schemas(t)
	for name, value := range map[string]any{
		"HistoryClinic":            history.Clinic{},
		"HistoryItem":              history.Item{},
		"HistoryPage":              history.Page{},
		"HistorySummary":           history.Summary{},
		"SummaryAllergy":           history.SummaryAllergy{},
		"SummaryMedication":        history.SummaryMedication{},
		"OpenItem":                 history.OpenItem{},
		"Dosage":                   medicationstatement.Dosage{},
		"AllergyIntoleranceInput":  allergyintolerance.Input{},
		"MedicationStatementInput": medicationstatement.Input{},
		"SharingCandidate":         consent.Candidate{},
		"HistorySharing":           consent.Sharing{},
		"SetHistorySharingRequest": consent.SetSharing{},
	} {
		sch, ok := s[name]
		if !ok {
			t.Errorf("no schema %s", name)
			continue
		}
		all, required := jsonFields(value)
		props := slices.Sorted(func(yield func(string) bool) {
			for p := range sch.Properties {
				if !yield(p) {
					return
				}
			}
		})
		if !slices.Equal(props, all) {
			t.Errorf("%s: properties %v, want the Go fields %v", name, props, all)
		}
		if req := slices.Sorted(slices.Values(sch.Required)); !slices.Equal(req, required) {
			t.Errorf("%s: required %v, want %v", name, req, required)
		}
	}
}

// The history's enums are the Go constants, in order.
func TestHistoryEnums_AreTheGoConstants(t *testing.T) {
	t.Parallel()
	s := schemas(t)
	for name, want := range map[string][]string{
		"HistoryItemType": {string(history.TypeVisit), string(history.TypeDocument), string(history.TypeQuestionnaireResponse),
			string(history.TypeAttachment), string(history.TypeCommunication)},
		"AllergyCategory": {string(allergyintolerance.CategoryFood), string(allergyintolerance.CategoryMedication),
			string(allergyintolerance.CategoryEnvironment), string(allergyintolerance.CategoryBiologic)},
		"AllergySeverity": {string(allergyintolerance.SeverityMild), string(allergyintolerance.SeverityModerate),
			string(allergyintolerance.SeveritySevere)},
		"AllergyClinicalStatus": {string(allergyintolerance.ClinicalActive), string(allergyintolerance.ClinicalInactive),
			string(allergyintolerance.ClinicalResolved)},
		"MedicationStatus": {string(medicationstatement.StatusActive), string(medicationstatement.StatusStopped)},
		"DoseUnit":         medicationstatement.DoseUnits(),
		"PeriodUnit":       medicationstatement.PeriodUnits(),
	} {
		if got := s[name].Enum; !slices.Equal(got, want) {
			t.Errorf("%s enum = %v, want %v", name, got, want)
		}
	}
	if got, want := s["OpenItem"].Properties["type"].Enum, []string{string(history.OpenQuestionnaire), string(history.OpenDraft)}; !slices.Equal(got, want) {
		t.Errorf("OpenItem.type enum = %v, want %v", got, want)
	}
}
