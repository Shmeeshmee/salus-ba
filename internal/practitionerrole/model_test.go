package practitionerrole_test

import (
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/practitionerrole"
)

// The owner's twelve specialties (docs/booking/SPEC.md "Specialty"), with the code and display of each from
// ValueSet c80-practice-codes in hl7.fhir.r4.core 4.0.1 (docs/booking/PROGRESS.md, package 4). Text is the
// owner's name for it.
var twelve = []practitionerrole.Specialty{
	{Code: "394814009", Display: "General practice", Text: "General practice"},
	{Code: "419192003", Display: "Internal medicine", Text: "Internal medicine"},
	{Code: "394579002", Display: "Cardiology", Text: "Cardiology"},
	{Code: "394582007", Display: "Dermatology", Text: "Dermatology"},
	{Code: "394586005", Display: "Gynecology", Text: "Gynaecology"},
	{Code: "394733009", Display: "Medical specialty--OTHER--NOT LISTED", Text: "Paediatrics"},
	{Code: "394587001", Display: "Psychiatry", Text: "Psychiatry"},
	{Code: "394591006", Display: "Neurology", Text: "Neurology"},
	{Code: "418960008", Display: "Otolaryngology", Text: "ENT"},
	{Code: "394594003", Display: "Ophthalmology", Text: "Ophthalmology"},
	{Code: "394801008", Display: "Surgery-Trauma and orthopedics", Text: "Orthopaedics"},
	{Code: "394583002", Display: "Endocrinology", Text: "Endocrinology"},
}

func TestSpecialties_AreTheOwnersTwelveFromThePracticeCodes(t *testing.T) {
	t.Parallel()
	if got := practitionerrole.Specialties(); !slices.Equal(got, twelve) {
		t.Errorf("Specialties() =\n%+v\nwant\n%+v", got, twelve)
	}
	if practitionerrole.SystemSNOMED != "http://snomed.info/sct" {
		t.Errorf("SystemSNOMED = %q", practitionerrole.SystemSNOMED)
	}
}

func TestSpecialties_GivesACopy(t *testing.T) {
	t.Parallel()
	practitionerrole.Specialties()[0].Code = "changed"
	if practitionerrole.Specialties()[0].Code != "394814009" {
		t.Error("a change to the returned list reached the next caller")
	}
}

func TestSpecialtyByCode_FindsOnlyAListedCode(t *testing.T) {
	t.Parallel()
	for _, s := range twelve {
		if got, ok := practitionerrole.SpecialtyByCode(s.Code); !ok || got != s {
			t.Errorf("SpecialtyByCode(%q) = %+v, %v; want %+v", s.Code, got, ok, s)
		}
	}
	// In the value set but not one of the twelve, blank, padded.
	for _, code := range []string{"394592004", "", " 394579002", "Cardiology"} {
		if got, ok := practitionerrole.SpecialtyByCode(code); ok {
			t.Errorf("SpecialtyByCode(%q) = %+v, true; want false", code, got)
		}
	}
}
