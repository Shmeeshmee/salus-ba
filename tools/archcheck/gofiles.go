package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// skipDirs are not part of the app module (tools/ is its own module) or hold no app Go code.
var skipDirs = map[string]bool{".git": true, "tools": true, "web": true, "mobile": true, "docs": true, "node_modules": true, "testdata": true, "vendor": true}

var forbiddenNames = map[string]bool{"utils": true, "common": true, "shared": true, "helpers": true}

var newRepository = regexp.MustCompile(`^New\w*Repository$`)

func checkGoFiles(fsys fs.FS, module string, domains map[string]bool, infrastructure []string) ([]violation, error) {
	var vs []violation
	err := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if skipDirs[d.Name()] && p != "." {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".go") {
			return nil
		}
		src, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}
		f, err := parser.ParseFile(token.NewFileSet(), p, src, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		vs = append(vs, checkLocation(p, domains, infrastructure)...)
		vs = append(vs, checkImports(p, f, module)...)
		vs = append(vs, checkForeignRepositories(p, f, module, domains)...)
		return nil
	})
	return vs, err
}

// checkLocation applies the package-name rules to the file's directory.
func checkLocation(p string, domains map[string]bool, infrastructure []string) []violation {
	var vs []violation
	parts := strings.Split(path.Dir(p), "/")
	if forbiddenLocation(parts) {
		vs = append(vs, violation{"forbidden-package-name", p, "no utils, common, shared, helpers, /src or technical-layer packages"})
	}
	if len(parts) >= 2 && parts[0] == "internal" && !domains[parts[1]] && !slices.Contains(infrastructure, parts[1]) {
		vs = append(vs, violation{"unlisted-domain", p, "internal/" + parts[1] + " is in no context or infrastructure list of config.yaml"})
	}
	return vs
}

func forbiddenLocation(parts []string) bool {
	if parts[0] == "src" {
		return true
	}
	if parts[0] != "internal" && parts[0] != "pkg" {
		return false
	}
	if parts[0] == "internal" && len(parts) >= 2 && slices.Contains([]string{"services", "repositories", "handlers"}, parts[1]) {
		return true
	}
	return slices.ContainsFunc(parts[1:], func(s string) bool { return forbiddenNames[s] })
}

func checkImports(p string, f *ast.File, module string) []violation {
	var vs []violation
	internal := module + "/internal"
	for _, imp := range f.Imports {
		ip, _ := strconv.Unquote(imp.Path.Value)
		if strings.HasPrefix(p, "pkg/") && (ip == internal || strings.HasPrefix(ip, internal+"/")) {
			vs = append(vs, violation{"pkg-imports-internal", p, "imports " + ip})
		}
		rel, ok := strings.CutPrefix(ip, internal+"/")
		if !ok {
			continue
		}
		parts := strings.Split(rel, "/")
		if len(parts) == 2 && parts[1] == parts[0]+"db" && p != "internal/"+parts[0]+"/repository_postgres.go" {
			vs = append(vs, violation{"sqlc-package-import", p, "imports " + ip + "; only internal/" + parts[0] + "/repository_postgres.go may"})
		}
	}
	return vs
}

// checkForeignRepositories finds a domain referring to another domain's New...Repository constructor.
// Composition roots (infrastructure packages, cmd/) build repositories; domains call each other's services.
func checkForeignRepositories(p string, f *ast.File, module string, domains map[string]bool) []violation {
	parts := strings.Split(p, "/")
	if len(parts) < 3 || parts[0] != "internal" || !domains[parts[1]] {
		return nil
	}
	own := parts[1]
	others := map[string]string{} // local import name -> domain
	for _, imp := range f.Imports {
		ip, _ := strconv.Unquote(imp.Path.Value)
		dom, ok := strings.CutPrefix(ip, module+"/internal/")
		if !ok || strings.Contains(dom, "/") || dom == own || !domains[dom] {
			continue
		}
		name := dom
		if imp.Name != nil {
			name = imp.Name.Name
		}
		others[name] = dom
	}
	var vs []violation
	ast.Inspect(f, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if x, ok := sel.X.(*ast.Ident); ok && others[x.Name] != "" && newRepository.MatchString(sel.Sel.Name) {
			vs = append(vs, violation{"foreign-repository", p, "uses " + x.Name + "." + sel.Sel.Name + "; call the " + others[x.Name] + " service instead"})
		}
		return true
	})
	return vs
}
