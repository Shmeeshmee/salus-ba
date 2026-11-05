package main

import (
	"go/parser"
	"go/token"
	"strconv"
	"testing"
)

// The clinic time zones are validated and shown with time.LoadLocation, so the binary carries the zone data
// itself rather than depend on the runtime image (docs/booking/SPEC.md "Time zones").
func TestMain_EmbedsTheTimeZoneData(t *testing.T) {
	f, err := parser.ParseFile(token.NewFileSet(), "main.go", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	for _, imp := range f.Imports {
		if path, _ := strconv.Unquote(imp.Path.Value); path == "time/tzdata" && imp.Name != nil && imp.Name.Name == "_" {
			return
		}
	}
	t.Error(`main.go does not import _ "time/tzdata"`)
}
