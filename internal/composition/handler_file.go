package composition

import (
	"encoding/json"
	"errors"
	"io"
)

// ErrInvalidSeed is returned for a seed file that is not one.
var ErrInvalidSeed = errors.New("composition: not a seed template file")

// DecodeSeed reads a seed template file (assets/document-templates, salusctl document-templates seed): a note
// saying what the template is and who is to review it, and the template as SaveDocumentTemplateRequest has it.
// Its content is checked by the service.
func DecodeSeed(r io.Reader) (TemplateContent, error) {
	var file struct {
		Template *struct {
			Title        string            `json:"title"`
			Language     string            `json:"language"`
			Status       TemplateStatus    `json:"status"`
			TypeCode     string            `json:"typeCode"`
			Fields       []TemplateField   `json:"fields"`
			Translations []translationJSON `json:"translations"`
		} `json:"template"`
		Note string `json:"note"`
	}
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&file); err != nil || file.Template == nil || blank(file.Note) {
		return TemplateContent{}, ErrInvalidSeed
	}
	t := file.Template
	in := TemplateContent{Title: t.Title, Language: t.Language, Status: t.Status, TypeCode: t.TypeCode, Fields: t.Fields}
	for _, tr := range t.Translations {
		in.Translations = append(in.Translations, Translation(tr))
	}
	return in, nil
}
