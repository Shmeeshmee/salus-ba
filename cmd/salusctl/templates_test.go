package main

import (
	"bytes"
	"strings"
	"testing"
)

// The dev seed: the two placeholder templates in assets/document-templates go into the clinic as drafts, once,
// audited with salusctl as the agent.
func TestDocumentTemplatesSeed_SeedsThePlaceholderTemplatesOnce(t *testing.T) {
	db, count := ownerDB(t)
	var clinic string
	if err := db.QueryRow(t.Context(), `INSERT INTO organization (name) VALUES ('Synthetic clinic') RETURNING id::text`).Scan(&clinic); err != nil {
		t.Fatal(err)
	}
	files := []string{"../../assets/document-templates/specialist-findings.json", "../../assets/document-templates/referral.json"}
	var out bytes.Buffer
	if err := run(t.Context(), append([]string{"document-templates", "seed", "-clinic", clinic}, files...), &out); err != nil {
		t.Fatalf("seed: %v (%s)", err, out.String())
	}
	if n := count(`SELECT count(*) FROM document_template WHERE organization_id = $1 AND status = 'draft'`, clinic); n != 2 {
		t.Errorf("%d draft templates, want 2", n)
	}
	if n := count(`SELECT count(*) FROM document_template_translation`); n != 2 {
		t.Errorf("%d translations, want 2", n)
	}
	if strings.Count(out.String(), "created template ") != 2 {
		t.Errorf("output %q", out.String())
	}
	out.Reset()
	if err := run(t.Context(), append([]string{"document-templates", "seed", "-clinic", clinic}, files...), &out); err != nil {
		t.Fatal(err)
	}
	if strings.Count(out.String(), "already seeded ") != 2 || count(`SELECT count(*) FROM document_template`) != 2 {
		t.Errorf("seed again: %q; want nothing created", out.String())
	}
	if n := count(`SELECT count(*) FROM audit_event WHERE agent_name = 'salusctl' AND entity_name = 'document_template'`); n != 2 {
		t.Errorf("audited %d seeds, want 2", n)
	}
	for _, args := range [][]string{{"document-templates", "seed"}, {"document-templates", "seed", "-clinic", clinic}, {"document-templates"}} {
		if err := run(t.Context(), args, &out); err == nil {
			t.Errorf("%v = nil, want the usage", args)
		}
	}
}
