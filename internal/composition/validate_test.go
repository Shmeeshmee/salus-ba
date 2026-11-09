package composition_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/composition"
)

func content() composition.TemplateContent {
	return composition.TemplateContent{
		Title: "Placeholder findings report", Language: "bs", Status: composition.TemplateDraft, TypeCode: "11488-4",
		Fields: []composition.TemplateField{
			{Key: "findings", Type: composition.FieldText, Label: "Placeholder findings", Instruction: "Placeholder instruction", Required: true},
			{Key: "tests", Type: composition.FieldList, Label: "Placeholder tests"},
			{Key: "control", Type: composition.FieldDate, Label: "Placeholder control date"},
			{Key: "diagnosis", Type: composition.FieldCoded, Label: "Placeholder diagnosis", Instruction: "Placeholder ICD-10"},
		},
		Translations: []composition.Translation{{
			Language: "en", Title: "Placeholder report",
			Fields: map[string]composition.FieldTexts{
				"findings":  {Label: "Placeholder findings", Instruction: "Placeholder instruction"},
				"tests":     {Label: "Placeholder tests"},
				"control":   {Label: "Placeholder control date"},
				"diagnosis": {Label: "Placeholder diagnosis", Instruction: "Placeholder ICD-10"},
			},
		}},
	}
}

func TestValidateTemplate_AcceptsAValidTemplate(t *testing.T) {
	t.Parallel()
	for name, change := range map[string]func(*composition.TemplateContent){
		"as it is":          func(*composition.TemplateContent) {},
		"active":            func(c *composition.TemplateContent) { c.Status = composition.TemplateActive },
		"no translation":    func(c *composition.TemplateContent) { c.Translations = nil },
		"one field":         func(c *composition.TemplateContent) { c.Fields, c.Translations = c.Fields[:1], nil },
		"a referral's type": func(c *composition.TemplateContent) { c.TypeCode = "57133-1" },
	} {
		c := content()
		change(&c)
		if err := composition.ValidateTemplate(c); err != nil {
			t.Errorf("%s: ValidateTemplate() = %v, want nil", name, err)
		}
	}
}

// A refusal names the offending field keys (none for the template as a whole), never a text.
func TestValidateTemplate_RefusesAnInvalidTemplateNamingTheFields(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("x", 201)
	for name, tt := range map[string]struct {
		change func(*composition.TemplateContent)
		keys   []string
	}{
		"a blank title":                    {func(c *composition.TemplateContent) { c.Title = " " }, nil},
		"a long title":                     {func(c *composition.TemplateContent) { c.Title = long }, nil},
		"an unknown language":              {func(c *composition.TemplateContent) { c.Language = "xx" }, nil},
		"Arabic, which the PDF cannot set": {func(c *composition.TemplateContent) { c.Language = "ar" }, nil},
		"an Arabic translation":            {func(c *composition.TemplateContent) { c.Translations[0].Language = "ar" }, nil},
		"an unknown status":                {func(c *composition.TemplateContent) { c.Status = "retired" }, nil},
		"a type no LOINC code":             {func(c *composition.TemplateContent) { c.TypeCode = "referral" }, nil},
		"no fields":                        {func(c *composition.TemplateContent) { c.Fields, c.Translations = nil, nil }, nil},
		"a key no identifier":              {func(c *composition.TemplateContent) { c.Fields[1].Key = "Tests!" }, []string{"Tests!"}},
		"a key twice":                      {func(c *composition.TemplateContent) { c.Fields[1].Key = "findings" }, []string{"findings"}},
		"an unknown type":                  {func(c *composition.TemplateContent) { c.Fields[2].Type = "number" }, []string{"control"}},
		"a blank label":                    {func(c *composition.TemplateContent) { c.Fields[0].Label = "" }, []string{"findings"}},
		"a long label":                     {func(c *composition.TemplateContent) { c.Fields[3].Label = long }, []string{"diagnosis"}},
		"a long instruction":               {func(c *composition.TemplateContent) { c.Fields[0].Instruction = strings.Repeat("x", 1001) }, []string{"findings"}},
		"a translation's label":            {func(c *composition.TemplateContent) { c.Translations[0].Fields["tests"] = composition.FieldTexts{} }, []string{"tests"}},
		"a translation missing":            {func(c *composition.TemplateContent) { delete(c.Translations[0].Fields, "control") }, []string{"control"}},
		"an instruction extra": {func(c *composition.TemplateContent) {
			c.Translations[0].Fields["tests"] = composition.FieldTexts{Label: "x", Instruction: "y"}
		}, []string{"tests"}},
		"an instruction lacking": {func(c *composition.TemplateContent) {
			c.Translations[0].Fields["findings"] = composition.FieldTexts{Label: "x"}
		}, []string{"findings"}},
		"an unknown key": {func(c *composition.TemplateContent) {
			c.Translations[0].Fields["other"] = composition.FieldTexts{Label: "x"}
		}, nil},
		"a blank title translated": {func(c *composition.TemplateContent) { c.Translations[0].Title = "" }, nil},
		"the original translated":  {func(c *composition.TemplateContent) { c.Translations[0].Language = "bs" }, nil},
		"a language twice": {func(c *composition.TemplateContent) {
			c.Translations = append(c.Translations, c.Translations[0])
		}, nil},
	} {
		c := content()
		tt.change(&c)
		err := composition.ValidateTemplate(c)
		var invalid *composition.InvalidTemplateError
		if !errors.As(err, &invalid) || !errors.Is(err, composition.ErrInvalidTemplate) {
			t.Errorf("%s: ValidateTemplate() = %v, want an InvalidTemplateError", name, err)
			continue
		}
		if !slices.Equal(invalid.Keys, tt.keys) {
			t.Errorf("%s: keys = %q, want %q", name, invalid.Keys, tt.keys)
		}
		if strings.Contains(err.Error(), "Placeholder") {
			t.Errorf("%s: the error %q holds a text", name, err)
		}
	}
}

func TestValidateTemplate_AtMost50Fields(t *testing.T) {
	t.Parallel()
	c := content()
	c.Translations = nil
	for i := len(c.Fields); i < 51; i++ {
		c.Fields = append(c.Fields, composition.TemplateField{Key: "f" + strings.Repeat("x", i), Type: composition.FieldText, Label: "x"})
	}
	if err := composition.ValidateTemplate(c); !errors.Is(err, composition.ErrInvalidTemplate) {
		t.Errorf("51 fields: %v, want ErrInvalidTemplate", err)
	}
}

// A template in a language reads its title and its fields' texts in that language: the original's, or a
// translation's; another language is not one it has.
func TestTemplate_InReadsTheTextsOfALanguageItHas(t *testing.T) {
	t.Parallel()
	c := content()
	tpl := composition.Template{Title: c.Title, Language: c.Language, Fields: c.Fields, Translations: c.Translations}
	title, fields, ok := tpl.In("en")
	if !ok || title != "Placeholder report" || fields[0].Label != "Placeholder findings" || fields[0].Type != composition.FieldText || !fields[0].Required {
		t.Errorf("In(en) = %q, %+v, %v", title, fields, ok)
	}
	if title, fields, ok := tpl.In("bs"); !ok || title != c.Title || !slices.Equal(fields, c.Fields) {
		t.Errorf("In(bs) = %q, %+v, %v; want the original", title, fields, ok)
	}
	if _, _, ok := tpl.In("de"); ok {
		t.Error("In(de) = ok, want a language the template does not have")
	}
}
