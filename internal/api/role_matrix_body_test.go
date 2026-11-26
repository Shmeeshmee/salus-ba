package api_test

import (
	"encoding/json"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/Shmeeshmee/salus/pkg/clock"
)

// contractDoc is api/openapi.yaml as plain maps, read once for the package: the role matrix builds every write's body
// from its request schema (docs/clinic-cms/WAVE7.md RD1), so that a write reaches its access check instead of being
// rejected as malformed, and a write added later gets its body without a line here.
var contractDoc = sync.OnceValues(func() (map[string]any, error) {
	b, err := os.ReadFile("../../api/openapi.yaml")
	if err != nil {
		return nil, err
	}
	var doc map[string]any
	return doc, yaml.Unmarshal(b, &doc)
})

// bodyIDs are the fields of a request body that name a resource, and the fixture's id kind each takes.
var bodyIDs = map[string]string{
	"patientId": "patient", "appointmentId": "visit", "serviceId": "service", "slotId": "slot",
	"organizationId": "org", "organization_id": "org", "practitionerRoleId": "role", "practitioner_role_id": "role",
	"locationId": "location", "roomId": "room", "deviceId": "device", "questionnaireId": "questionnaire",
	"templateId": "template", "dataRequestId": "request", "quoteId": "quote", "receiptId": "receipt",
	"carePlanId": "plan", "encounterId": "encounter", "encounterIds": "encounter",
}

// matrixBodies are the bodies the schema cannot give: a write whose validation needs more than its schema says.
var matrixBodies = map[string]string{
	"POST /clinic/appointments/{id}/diagnostic-reports": `{"title":"Placeholder blood count","results":[]}`,
	"PUT /clinic/diagnostic-reports/{id}":               `{"title":"Placeholder blood count","results":[]}`,
	"PUT /practitioner-roles/{id}/availability":         `{"slotMinutes":30,"availability":[{"day":1,"start":"09:00","end":"12:00"}]}`,
	"PUT /practitioner-roles/{id}/specialty":            `{"code":"394579002"}`,
	"POST /invites":                                     `{"email":"matrix-invite@example.test","role":"practitioner","givenName":"Synthetic","familyName":"Invitee"}`,
	"POST /clinic/services":                             `{"name":"Synthetic exam","priceMinor":5000,"modes":["in-person"],"active":true,"translations":[]}`,
	"PUT /clinic/services/{id}":                         `{"name":"Synthetic exam","priceMinor":5000,"modes":["in-person"],"active":true,"translations":[]}`,
}

// requestBody gives op's JSON request body built from its schema: every required field and every field that names a
// resource, each id from ids (by kind, unknownID for a kind ids lacks), "" when op takes no JSON body.
func requestBody(t *testing.T, op string, ids map[string]string) string {
	t.Helper()
	doc, err := contractDoc()
	if err != nil {
		t.Fatal(err)
	}
	if b, ok := matrixBodies[op]; ok {
		return b
	}
	method, path, _ := strings.Cut(op, " ")
	operation, _ := dig(doc, "paths", path, strings.ToLower(method)).(map[string]any)
	schema := dig(operation, "requestBody", "content", "application/json", "schema")
	if schema == nil {
		return ""
	}
	g := bodyGen{doc: doc, ids: ids, now: clock.System{}.Now().UTC().Truncate(time.Hour)}
	b, err := json.Marshal(g.example(schema, "", 0))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func dig(v any, keys ...string) any {
	for _, k := range keys {
		m, ok := v.(map[string]any)
		if !ok {
			return nil
		}
		v = m[k]
	}
	return v
}

type bodyGen struct {
	now time.Time
	doc map[string]any
	ids map[string]string
}

func (g bodyGen) resolve(s map[string]any) map[string]any {
	for i := 0; i < 8; i++ {
		ref, ok := s["$ref"].(string)
		if !ok {
			return s
		}
		next, _ := dig(g.doc, strings.Split(strings.TrimPrefix(ref, "#/"), "/")...).(map[string]any)
		if next == nil {
			return s
		}
		s = next
	}
	return s
}

// example gives a value of schema for the field name.
func (g bodyGen) example(raw any, name string, depth int) any {
	s, _ := raw.(map[string]any)
	if s == nil || depth > 8 {
		return nil
	}
	s = g.resolve(s)
	if all, ok := s["allOf"].([]any); ok {
		out := map[string]any{}
		for _, part := range all {
			if m, ok := g.example(part, name, depth+1).(map[string]any); ok {
				for k, v := range m {
					out[k] = v
				}
			}
		}
		return out
	}
	for _, k := range []string{"oneOf", "anyOf"} {
		if alts, ok := s[k].([]any); ok && len(alts) > 0 {
			return g.example(alts[0], name, depth+1)
		}
	}
	if enum, ok := s["enum"].([]any); ok && len(enum) > 0 {
		return enum[0]
	}
	typ, _ := s["type"].(string)
	if list, ok := s["type"].([]any); ok && len(list) > 0 {
		typ, _ = list[0].(string)
	}
	switch {
	case typ == "object" || s["properties"] != nil:
		return g.object(s, depth)
	case typ == "array":
		return []any{g.example(s["items"], name, depth+1)}
	case typ == "integer" || typ == "number":
		if m, ok := s["minimum"].(int); ok && m > 1 {
			return m
		}
		return 1
	case typ == "boolean":
		return true
	}
	return g.str(s, name)
}

func (g bodyGen) object(s map[string]any, depth int) map[string]any {
	required := map[string]bool{}
	if req, ok := s["required"].([]any); ok {
		for _, r := range req {
			if k, ok := r.(string); ok {
				required[k] = true
			}
		}
	}
	out := map[string]any{}
	props, _ := s["properties"].(map[string]any)
	for k, p := range props {
		if _, names := bodyIDs[k]; required[k] || names {
			out[k] = g.example(p, k, depth+1)
		}
	}
	return out
}

func (g bodyGen) str(s map[string]any, name string) string {
	format, _ := s["format"].(string)
	switch format {
	case "uuid":
		if id, ok := g.ids[bodyIDs[name]]; ok {
			return id
		}
		return unknownID
	case "date-time":
		return g.now.Add(72 * time.Hour).Format(time.RFC3339)
	case "date":
		if strings.Contains(strings.ToLower(name), "birth") {
			return "1980-05-05"
		}
		return g.now.Add(72 * time.Hour).Format(time.DateOnly)
	case "email":
		return "matrix@example.test"
	case "uri":
		return "https://hooks.example.test/matrix"
	}
	if n, ok := s["minLength"].(int); ok && n > 9 {
		return strings.Repeat("s", n)
	}
	return "Synthetic"
}
