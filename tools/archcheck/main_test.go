package main

import (
	"slices"
	"strings"
	"testing"
	"testing/fstest"
)

// cleanRepo is a small repository that breaks no rule. Each test adds or changes files.
func cleanRepo() fstest.MapFS {
	file := func(s string) *fstest.MapFile { return &fstest.MapFile{Data: []byte(s)} }
	return fstest.MapFS{
		"go.mod": file("module example.com/app\n\ngo 1.27\n"),
		"docs/architecture/archcheck/config.yaml": file(`
contexts:
  scheduling: [appointment, slot]
infrastructure: [api, config]
library_owners: [river]
`),
		"docs/architecture/archcheck/ownership.yaml": file(`
tables:
  appointment: {owner: appointment}
  river_job: {owner: river}
  river_client: {owner: river, dropped: true}
`),
		"docs/architecture/archcheck/allowlist.yaml": file("violations: []\n"),
		"migrations/00001_init.sql": file(`-- +goose Up
CREATE TABLE appointment (id uuid);
CREATE TABLE IF NOT EXISTS river_job (id bigint);
CREATE UNLOGGED TABLE river_client (id text);
`),
		"pkg/clock/clock.go":                              file("package clock\n"),
		"internal/config/config.go":                       file("package config\n"),
		"internal/appointment/service.go":                 file("package appointment\n\nimport _ \"example.com/app/internal/slot\"\n"),
		"internal/appointment/repository_postgres.go":     file("package appointment\n\nimport _ \"example.com/app/internal/appointment/appointmentdb\"\n"),
		"internal/appointment/appointmentdb/query.sql.go": file("package appointmentdb\n"),
		"internal/slot/service.go":                        file("package slot\n\nfunc NewPostgresRepository() {}\n"),
		"internal/api/app.go":                             file("package api\n\nimport \"example.com/app/internal/slot\"\n\nvar _ = slot.NewPostgresRepository\n"),
		// Ignored: another module, and test fixtures.
		"tools/archcheck/main.go":                 file("package main\n\nimport _ \"example.com/app/internal/slot\"\n"),
		"internal/appointment/testdata/broken.go": file("not go at all"),
	}
}

// rules returns "rule file" for each violation.
func rules(t *testing.T, fsys fstest.MapFS) []string {
	t.Helper()
	vs, err := check(fsys)
	if err != nil {
		t.Fatalf("check() error = %v", err)
	}
	var out []string
	for _, v := range vs {
		out = append(out, v.Rule+" "+v.File)
	}
	slices.Sort(out)
	return out
}

func assertRules(t *testing.T, fsys fstest.MapFS, want ...string) {
	t.Helper()
	got := rules(t, fsys)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("violations:\n got %q\nwant %q", got, want)
	}
}

func TestCheck_CleanRepositoryPasses(t *testing.T) {
	assertRules(t, cleanRepo())
}

func TestCheck_PkgMustNotImportInternal(t *testing.T) {
	repo := cleanRepo()
	repo["pkg/httpx/httpx.go"] = &fstest.MapFile{Data: []byte("package httpx\n\nimport _ \"example.com/app/internal/config\"\n")}

	assertRules(t, repo, "pkg-imports-internal pkg/httpx/httpx.go")
}

func TestCheck_ForbiddenPackageNames(t *testing.T) {
	repo := cleanRepo()
	for _, dir := range []string{"internal/utils", "pkg/common", "internal/appointment/helpers", "internal/services", "src/app"} {
		repo[dir+"/x.go"] = &fstest.MapFile{Data: []byte("package x\n")}
	}

	assertRules(t, repo,
		"forbidden-package-name internal/appointment/helpers/x.go",
		"forbidden-package-name internal/services/x.go",
		"forbidden-package-name internal/utils/x.go",
		"forbidden-package-name pkg/common/x.go",
		"forbidden-package-name src/app/x.go",
		"unlisted-domain internal/services/x.go",
		"unlisted-domain internal/utils/x.go",
	)
}

func TestCheck_EveryInternalPackageIsListed(t *testing.T) {
	repo := cleanRepo()
	repo["internal/patient/service.go"] = &fstest.MapFile{Data: []byte("package patient\n")}

	assertRules(t, repo, "unlisted-domain internal/patient/service.go")
}

func TestCheck_SqlcPackageOnlyImportedByItsRepository(t *testing.T) {
	repo := cleanRepo()
	repo["internal/appointment/service.go"] = &fstest.MapFile{Data: []byte("package appointment\n\nimport _ \"example.com/app/internal/appointment/appointmentdb\"\n")}
	repo["internal/slot/repository_postgres.go"] = &fstest.MapFile{Data: []byte("package slot\n\nimport _ \"example.com/app/internal/appointment/appointmentdb\"\n")}

	assertRules(t, repo,
		"sqlc-package-import internal/appointment/service.go",
		"sqlc-package-import internal/slot/repository_postgres.go",
	)
}

func TestCheck_DomainMustNotBuildAnotherDomainsRepository(t *testing.T) {
	repo := cleanRepo()
	repo["internal/appointment/service.go"] = &fstest.MapFile{Data: []byte(
		"package appointment\n\nimport \"example.com/app/internal/slot\"\n\nfunc f() { slot.NewPostgresRepository() }\n")}

	assertRules(t, repo, "foreign-repository internal/appointment/service.go")
}

func TestCheck_TableOwnership(t *testing.T) {
	repo := cleanRepo()
	repo["migrations/00002_more.sql"] = &fstest.MapFile{Data: []byte("-- +goose Up\ncreate table unowned (id int);\nCREATE TABLE \"slot\" (id int);\n")}
	repo["docs/architecture/archcheck/ownership.yaml"] = &fstest.MapFile{Data: []byte(`
tables:
  appointment: {owner: appointment}
  river_job: {owner: river}
  river_client: {owner: river, dropped: true}
  slot: {owner: nobody}
  ghost: {owner: appointment}
`)}

	assertRules(t, repo,
		"table-without-owner migrations/00002_more.sql",
		"unknown-table-owner docs/architecture/archcheck/ownership.yaml",
		"owned-table-not-created docs/architecture/archcheck/ownership.yaml",
	)
}

func TestCheck_AllowlistSuppressesListedViolations(t *testing.T) {
	repo := cleanRepo()
	repo["internal/patient/service.go"] = &fstest.MapFile{Data: []byte("package patient\n")}
	repo["docs/architecture/archcheck/allowlist.yaml"] = &fstest.MapFile{Data: []byte(`
violations:
  - {rule: unlisted-domain, file: internal/patient/service.go}
`)}

	assertRules(t, repo)
}

func TestCheck_ViolationsExplainThemselves(t *testing.T) {
	repo := cleanRepo()
	repo["pkg/httpx/httpx.go"] = &fstest.MapFile{Data: []byte("package httpx\n\nimport _ \"example.com/app/internal/config\"\n")}

	vs, err := check(repo)
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 1 || !strings.Contains(vs[0].Detail, "example.com/app/internal/config") {
		t.Errorf("violations = %+v, want one naming the imported package", vs)
	}
}
