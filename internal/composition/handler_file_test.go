package composition_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/composition"
)

// A seed file is a note for its reader and the template as SaveDocumentTemplateRequest has it.
func TestDecodeSeed_ReadsTheTemplateOfASeedFile(t *testing.T) {
	t.Parallel()
	got, err := composition.DecodeSeed(strings.NewReader(`{"note":"Placeholder wording.","template":` + templateJSON + `}`))
	if err != nil || got.Title != "T" || got.TypeCode != "57133-1" || got.Status != composition.TemplateDraft || len(got.Fields) != 1 ||
		len(got.Translations) != 1 || got.Translations[0].Language != "en" || got.Translations[0].Fields["a"].Label != "L" {
		t.Errorf("DecodeSeed() = %+v, %v", got, err)
	}
	for _, bad := range []string{`{"template":` + templateJSON + `}`, `{"note":"x","template":{"other":1}}`, `{"note":"x"}`, `[`} {
		if _, err := composition.DecodeSeed(strings.NewReader(bad)); !errors.Is(err, composition.ErrInvalidSeed) {
			t.Errorf("DecodeSeed(%s) = %v, want ErrInvalidSeed", bad, err)
		}
	}
}

const templateJSON = `{"title":"T","language":"bs","status":"draft","typeCode":"57133-1",
	"fields":[{"key":"a","type":"text","label":"L","required":true}],
	"translations":[{"language":"en","title":"T","fields":{"a":{"label":"L"}}}]}`
