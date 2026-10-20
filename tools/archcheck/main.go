// Command archcheck enforces the architecture rules in CLAUDE.md that the compiler and linters cannot:
// pkg/ never imports internal/, forbidden package names, every internal package listed as a domain or as
// infrastructure, sqlc packages imported only by their domain's repository, no domain building another
// domain's repository, and every table owned (docs/architecture/archcheck/).
//
// Usage: archcheck [-root dir]. It exits 1 when a violation is not in allowlist.yaml, which only shrinks.
package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

const configDir = "docs/architecture/archcheck/"

type violation struct {
	Rule   string
	File   string
	Detail string
}

type config struct {
	Contexts       map[string][]string `yaml:"contexts"`
	Infrastructure []string            `yaml:"infrastructure"`
	LibraryOwners  []string            `yaml:"library_owners"`
}

type ownership struct {
	Tables map[string]struct {
		Owner   string `yaml:"owner"`
		Dropped bool   `yaml:"dropped"`
	} `yaml:"tables"`
}

type allowlist struct {
	Violations []allowed `yaml:"violations"`
}

type allowed struct {
	Rule string `yaml:"rule"`
	File string `yaml:"file"`
}

func main() {
	root := flag.String("root", ".", "repository root")
	flag.Parse()
	vs, err := check(os.DirFS(*root))
	if err != nil {
		fmt.Fprintln(os.Stderr, "archcheck:", err)
		os.Exit(2)
	}
	for _, v := range vs {
		fmt.Printf("%s: %s: %s\n", v.File, v.Rule, v.Detail)
	}
	if len(vs) > 0 {
		fmt.Printf("archcheck: %d violation(s); see CLAUDE.md \"Go architecture\" (allowlist.yaml only shrinks)\n", len(vs))
		os.Exit(1)
	}
}

// check returns the violations in the repository at fsys that allowlist.yaml does not list.
func check(fsys fs.FS) ([]violation, error) {
	module, err := modulePath(fsys)
	if err != nil {
		return nil, err
	}
	var cfg config
	var own ownership
	var allow allowlist
	for name, dst := range map[string]any{"config.yaml": &cfg, "ownership.yaml": &own, "allowlist.yaml": &allow} {
		if err := readYAML(fsys, configDir+name, dst); err != nil {
			return nil, err
		}
	}
	domains := map[string]bool{}
	for _, pkgs := range cfg.Contexts {
		for _, p := range pkgs {
			domains[p] = true
		}
	}

	vs, err := checkGoFiles(fsys, module, domains, cfg.Infrastructure)
	if err != nil {
		return nil, err
	}
	tvs, err := checkTables(fsys, own, domains, cfg.LibraryOwners)
	if err != nil {
		return nil, err
	}
	vs = append(vs, tvs...)

	vs = slices.DeleteFunc(vs, func(v violation) bool {
		return slices.Contains(allow.Violations, allowed{Rule: v.Rule, File: v.File})
	})
	slices.SortFunc(vs, func(a, b violation) int { return strings.Compare(a.File+a.Rule, b.File+b.Rule) })
	return vs, nil
}

func modulePath(fsys fs.FS) (string, error) {
	data, err := fs.ReadFile(fsys, "go.mod")
	if err != nil {
		return "", err
	}
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		if m, ok := strings.CutPrefix(strings.TrimSpace(sc.Text()), "module "); ok {
			return strings.TrimSpace(m), nil
		}
	}
	return "", errors.New("go.mod has no module line")
}

func readYAML(fsys fs.FS, name string, dst any) error {
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(data, dst); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}
