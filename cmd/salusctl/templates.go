package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/composition"
)

const templatesUsage = "usage: salusctl document-templates seed -clinic <organization-id> <template.json>..."

// templatesCommand runs `salusctl document-templates seed`: the dev seed of placeholder document templates
// (assets/document-templates) into a clinic, each once, as drafts for the clinic's doctor to review.
func (s services) templatesCommand(ctx context.Context, args []string, stdout io.Writer) error {
	if len(args) == 0 || args[0] != "seed" {
		return errors.New(templatesUsage)
	}
	flags := flag.NewFlagSet("document-templates seed", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	clinic := flags.String("clinic", "", "the clinic's organization id")
	if err := flags.Parse(args[1:]); err != nil || *clinic == "" || flags.NArg() == 0 {
		return errors.New(templatesUsage)
	}
	for _, path := range flags.Args() {
		in, err := decodeTemplateFile(path)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		t, created, err := s.templates.SeedTemplate(ctx, auditevent.AgentOperator, *clinic, in)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		verb := "already seeded"
		if created {
			verb = "created template"
		}
		if _, err := fmt.Fprintln(stdout, verb, t.ID, "from", path); err != nil {
			return err
		}
	}
	return nil
}

func decodeTemplateFile(path string) (composition.TemplateContent, error) {
	f, err := os.Open(path)
	if err != nil {
		return composition.TemplateContent{}, err
	}
	defer func() { _ = f.Close() }()
	return composition.DecodeSeed(f)
}
