package api_test

import (
	"os"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/internal/signup"
)

// The wave 3 enums in api/openapi.yaml are the Go constants (docs/clinic-cms, package 3.0).
func TestClinicWave3Contract_EnumsMatchTheGoConstants(t *testing.T) {
	t.Parallel()
	s := schemas(t)
	staff := []string{string(auth.RoleAdmin), string(auth.RolePractitioner), string(auth.RoleReception)}
	for name, want := range map[string][]string{
		"InviteRequest": staff,
		"Invite":        staff,
		"Account":       append(slices.Clone(staff), string(auth.RolePatient)),
	} {
		if got := s[name].Properties["role"].Enum; !slices.Equal(got, want) {
			t.Errorf("%s.role: enum %v, want %v", name, got, want)
		}
	}
	statuses := []string{string(appointment.StatusBooked), string(appointment.StatusArrived), string(appointment.StatusCancelled),
		string(appointment.StatusFulfilled), string(appointment.StatusNoShow)}
	if got := s["Appointment"].Properties["status"].Enum; !slices.Equal(got, statuses) {
		t.Errorf("Appointment.status: enum %v, want %v", got, statuses)
	}
	states := []string{string(signup.StepOpen), string(signup.StepDone), string(signup.StepSkipped)}
	if got := s["SetupStepState"].Enum; !slices.Equal(got, states) {
		t.Errorf("SetupStepState: enum %v, want %v", got, states)
	}
	steps := []string{string(signup.StepDetails), string(signup.StepBilling), string(signup.StepHours), string(signup.StepPractitioner), string(signup.StepServices)}
	if got := s["ClinicSetup"].Required; !slices.Equal(got, steps) {
		t.Errorf("ClinicSetup: steps %v, want %v", got, steps)
	}
}

// Every wave 3 route of the clinic system is in the contract and names the package that builds it.
func TestClinicWave3Contract_EveryRouteNamesItsPackage(t *testing.T) {
	t.Parallel()
	b, err := os.ReadFile("openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Paths map[string]map[string]any `yaml:"paths"`
	}
	if err := yaml.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	for route, want := range map[string]string{
		"get /clinic/patients":                      "cms-f2",
		"post /clinic/patients":                     "cms-f2",
		"post /clinic/appointments":                 "cms-f2",
		"post /clinic/appointments/{id}/reschedule": "cms-f2",
		"post /clinic/appointments/{id}/cancel":     "cms-f2",
		"post /clinic/appointments/{id}/arrive":     "cms-f2",
		"get /clinic/desk/today":                    "cms-f2",
		"post /signup":                              "cms-o1",
		"post /signup/verify":                       "cms-o1",
		"get /clinic/setup":                         "cms-o1",
		"put /clinic/setup":                         "cms-o1",
		"post /clinic/services/import":              "cms-o2",
		"post /clinic/export":                       "cms-o2",
		"get /clinic/export/{id}":                   "cms-o2",
		"get /clinic/export/{id}/archive":           "cms-o2",
		"get /clinic/quotes":                        "cms-x1",
		"post /clinic/quotes":                       "cms-x1",
		"get /clinic/quotes/{id}":                   "cms-x1",
		"get /me/quotes":                            "cms-x1",
		"get /me/quotes/{id}/pdf":                   "cms-x1",
		"get /clinic/profile":                       "cms-x2",
		"put /clinic/profile":                       "cms-x2",
		"get /organizations/{id}/profile":           "cms-x2",
	} {
		method, path, _ := strings.Cut(route, " ")
		op, _ := doc.Paths[path][method].(map[string]any)
		if got, _ := op["x-salus-package"].(string); got != want {
			t.Errorf("%s: x-salus-package %q, want %q", route, got, want)
		}
	}
}
