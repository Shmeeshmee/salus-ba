package main

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"slices"
	"strconv"
)

const rulesFile = "internal/retention/rules.go"

// checkRetention enforces the retention hook (docs/accounts/SPEC.md): every domain's table says whether it
// holds personal data, every table that does has a rule in rules.go, and every rule names a live table.
func checkRetention(fsys fs.FS, own ownership, created map[string]bool, libraryOwners []string) ([]violation, error) {
	ruled, err := ruleTables(fsys)
	if err != nil {
		return nil, err
	}
	const ownFile = configDir + "ownership.yaml"
	var vs []violation
	for table, t := range own.Tables {
		switch {
		case slices.Contains(libraryOwners, t.Owner) || t.Dropped:
		case t.PersonalData == nil:
			vs = append(vs, violation{"personal-data-unmarked", ownFile, "table " + table + " has no personal_data: true|false"})
		case *t.PersonalData && !slices.Contains(ruled, table):
			vs = append(vs, violation{"retention-rule-missing", ownFile, "table " + table + " holds personal data and has no rule in " + rulesFile})
		}
	}
	for _, table := range ruled {
		if !created[table] || own.Tables[table].Dropped {
			vs = append(vs, violation{"retention-rule-unknown-table", rulesFile, "a rule names table " + table + ", which no migration creates or which was dropped"})
		}
	}
	return vs, nil
}

// ruleTables returns the Table field of every Rule literal in rules.go, or nothing when the file is missing.
func ruleTables(fsys fs.FS) ([]string, error) {
	src, err := fs.ReadFile(fsys, rulesFile)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	f, err := parser.ParseFile(token.NewFileSet(), rulesFile, src, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	var tables []string
	ast.Inspect(f, func(n ast.Node) bool {
		kv, ok := n.(*ast.KeyValueExpr)
		if !ok {
			return true
		}
		key, ok := kv.Key.(*ast.Ident)
		lit, isLit := kv.Value.(*ast.BasicLit)
		if ok && isLit && key.Name == "Table" && lit.Kind == token.STRING {
			table, _ := strconv.Unquote(lit.Value) // a parsed string literal always unquotes
			tables = append(tables, table)
		}
		return true
	})
	return tables, nil
}
