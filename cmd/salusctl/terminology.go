package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Shmeeshmee/salus/internal/terminology"
)

const terminologyUsage = "usage: salusctl terminology import <seed.jsonl | terms.csv> | terminology export -o <terms.csv> [-status draft|reviewed]"

// terminologyCommand runs `salusctl terminology import|export` (docs/terminology/SPEC.md §3 "Curation").
func (s services) terminologyCommand(ctx context.Context, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return errors.New(terminologyUsage)
	}
	switch args[0] {
	case "import":
		if len(args) != 2 {
			return errors.New(terminologyUsage)
		}
		return s.importTerms(ctx, args[1], stdout)
	case "export":
		return s.exportTerms(ctx, args[1:], stdout)
	default:
		return fmt.Errorf("unknown command %q; %s", args[0], terminologyUsage)
	}
}

// importTerms applies the seed (.jsonl) or the curator's CSV (.csv) and prints what it did, the refused rows and
// the terms per category. Refused rows make it fail once the rest is applied.
func (s services) importTerms(ctx context.Context, path string, stdout io.Writer) error {
	decode, apply := terminology.DecodeSeed, s.terms.ImportSeed
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jsonl":
	case ".csv":
		decode, apply = terminology.DecodeCSV, s.terms.ImportCurated
	default:
		return errors.New("terminology import: the file is neither .jsonl (the seed) nor .csv (the curator's)")
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	rows, refused, err := decode(f)
	if err != nil {
		return err
	}
	report, err := apply(ctx, rows)
	report.Refused = append(refused, report.Refused...)
	if _, werr := io.WriteString(stdout, reportText(report)); err == nil {
		err = werr
	}
	if err != nil {
		return err
	}
	counts, err := s.terms.Counts(ctx)
	if err != nil {
		return err
	}
	parts := make([]string, len(counts))
	for i, c := range counts {
		parts[i] = fmt.Sprintf("%s %d", c.Category, c.Terms)
	}
	if _, err := fmt.Fprintf(stdout, "terms: %s\n", strings.Join(parts, ", ")); err != nil {
		return err
	}
	if len(report.Refused) > 0 {
		return fmt.Errorf("terminology import: rows refused: %d; the others are applied", len(report.Refused))
	}
	return nil
}

// reportText is what an import did, a line per refused row: its line, its id when it has one, and the reason.
func reportText(r terminology.ImportReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "created %d, updated %d, unchanged %d, refused %d\n", r.Created, r.Updated, r.Unchanged, len(r.Refused))
	for _, e := range r.Refused {
		if e.ID != "" {
			fmt.Fprintf(&b, "line %d (id %s): %s\n", e.Line, e.ID, e.Reason)
		} else {
			fmt.Fprintf(&b, "line %d: %s\n", e.Line, e.Reason)
		}
	}
	return b.String()
}

// exportTerms writes the curator's CSV.
func (s services) exportTerms(ctx context.Context, args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("terminology export", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	out := flags.String("o", "", "the CSV file to write")
	status := flags.String("status", "", "draft or reviewed; every term when absent")
	if err := flags.Parse(args); err != nil || *out == "" || flags.NArg() != 0 {
		return errors.New(terminologyUsage)
	}
	terms, err := s.terms.Export(ctx, *status)
	if err != nil {
		return err
	}
	f, err := os.Create(*out)
	if err != nil {
		return err
	}
	if err := terminology.EncodeCSV(f, terms); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "terms exported: %d, to %s\n", len(terms), *out)
	return err
}
