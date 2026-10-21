package main

import (
	"io/fs"
	"regexp"
	"slices"
	"strings"
)

var (
	createTable = regexp.MustCompile(`(?i)\bcreate\s+(?:unlogged\s+|temp(?:orary)?\s+)?table\s+(?:if\s+not\s+exists\s+)?"?([a-z_][a-z0-9_]*)"?`)
	sqlComment  = regexp.MustCompile(`--[^\n]*`)
)

// checkTables matches every CREATE TABLE in migrations/ against ownership.yaml.
func checkTables(fsys fs.FS, own ownership, domains map[string]bool, libraryOwners []string) ([]violation, error) {
	files, err := fs.Glob(fsys, "migrations/*.sql")
	if err != nil {
		return nil, err
	}
	created := map[string]bool{}
	var vs []violation
	for _, file := range files {
		src, err := fs.ReadFile(fsys, file)
		if err != nil {
			return nil, err
		}
		for _, m := range createTable.FindAllStringSubmatch(sqlComment.ReplaceAllString(string(src), ""), -1) {
			table := strings.ToLower(m[1])
			created[table] = true
			if _, ok := own.Tables[table]; !ok {
				vs = append(vs, violation{"table-without-owner", file, "table " + table + " is not in ownership.yaml"})
			}
		}
	}
	const ownFile = configDir + "ownership.yaml"
	for table, t := range own.Tables {
		if !created[table] {
			vs = append(vs, violation{"owned-table-not-created", ownFile, "table " + table + " is created by no migration"})
		}
		if !domains[t.Owner] && !slices.Contains(libraryOwners, t.Owner) {
			vs = append(vs, violation{"unknown-table-owner", ownFile, "table " + table + " is owned by " + t.Owner + ", which is no listed domain or library"})
		}
	}
	return vs, nil
}
