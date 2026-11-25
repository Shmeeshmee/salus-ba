package api_test

import (
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// TestEveryTestRunsInParallel keeps this package's tests parallel. Each builds its own App on its own database, so
// nothing is shared; run one after another they took 90 of the package's 104 s alone and more than go test's
// 10-minute package limit under the whole suite's load (docs/clinic-cms/PROGRESS.md, "Wave 6, second half").
func TestEveryTestRunsInParallel(t *testing.T) {
	t.Parallel()
	files, err := filepath.Glob("*_test.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	for _, name := range files {
		// Files behind a build tag (the loadtest readings) are not in this build and run alone with -run.
		if ok, err := build.Default.MatchFile(".", name); err != nil || !ok {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || !strings.HasPrefix(fn.Name.Name, "Test") || fn.Name.Name == "TestMain" {
				continue
			}
			if !startsParallel(fn) {
				t.Errorf("%s: %s does not start with t.Parallel()", name, fn.Name.Name)
			}
		}
	}
}

// startsParallel reports whether fn's first statement is <param>.Parallel().
func startsParallel(fn *ast.FuncDecl) bool {
	if fn.Body == nil || len(fn.Body.List) == 0 || len(fn.Type.Params.List) != 1 || len(fn.Type.Params.List[0].Names) != 1 {
		return false
	}
	stmt, ok := fn.Body.List[0].(*ast.ExprStmt)
	if !ok {
		return false
	}
	call, ok := stmt.X.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Parallel" {
		return false
	}
	id, ok := sel.X.(*ast.Ident)
	return ok && id.Name == fn.Type.Params.List[0].Names[0].Name
}
