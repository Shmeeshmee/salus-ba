// Package production_test checks the production wiring (docs/deploy/BEFORE-PRODUCTION.md §3): every environment
// variable a binary reads through internal/config is listed for that binary's service in docker-compose.prod.yml, and
// every variable the compose file interpolates is named in .env.production.example. Compose passes a container only
// the variables its service lists, so a setting missing there never reaches the binary, whatever the env file holds.
//
// What each binary reads is derived from the code, not kept by hand: the os.Getenv and os.LookupEnv calls of each
// function in internal/config (and the functions it calls), and the internal/config functions each cmd/<binary>
// references. The hand-kept part is small: which service runs which binary when its command does not say so
// (extraService), which variables are never set in production (neverInProduction), and which of a loader's variables
// only some of the binaries that call it use (onlyFor). Each entry carries its reason, and a stale entry fails.
package production_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const (
	root        = "../.."
	composeFile = "docker-compose.prod.yml"
	exampleFile = ".env.production.example"
)

// extraService is the service each binary without a service of its own runs in.
var extraService = map[string]string{
	// The operator command runs in the migrate service, as the database owner (docs/deploy/README.md "The first clinic").
	"salusctl": "migrate",
}

// neverInProduction are variables internal/config reads that production never sets, with the reason.
var neverInProduction = map[string]string{
	"FAKE_ENGINE_SCRIPT": "the fake engines' scripts, set only by make e2e (deployments/docker-compose.e2e.yml); refused beside a real engine",
}

// onlyFor lists, for a variable that a loader reads for every binary calling it (config.Load above all), the binaries
// that use it; the others leave it unset and get the loader's default, which they never look at.
var onlyFor = map[string]struct {
	why      string
	binaries []string
}{
	"HTTP_ADDR":                   {"the only binary that listens", []string{"api"}},
	"REDIS_URL":                   {"the scheduler, the notifier, migrate and salusctl open no Redis connection", []string{"api", "worker", "agent"}},
	"APP_DATABASE_PASSWORD":       {"migrate sets it on salus_app; the others log in with DATABASE_URL", []string{"migrate"}},
	"RETENTION_DATABASE_PASSWORD": {"migrate sets it on salus_retention; the worker logs in with RETENTION_DATABASE_URL", []string{"migrate"}},
	"STATS_DATABASE_PASSWORD":     {"migrate sets it on salus_stats; the worker logs in with STATS_DATABASE_URL", []string{"migrate"}},
	"RETENTION_DATABASE_URL":      {"the retention job's pool; migrate reads only the password", []string{"worker"}},
	"STATS_DATABASE_URL":          {"the aggregate builds' pool; migrate reads only the password", []string{"worker"}},
	"WEB_BASE_URL":                {"links in e-mails", []string{"notifier"}},
	"PATIENT_APP_URL":             {"links in appointment e-mails", []string{"notifier"}},
	"APPOINTMENT_LINK_URL":        {"the reminders' one-time links", []string{"notifier"}},
	"FHIR_BASE_URL":               {"the Bundle links and the attachments' download base", []string{"api"}},
	"TOTP_KEY":                    {"seals the TOTP secrets at sign-in", []string{"api"}},
	"RATE_LIMIT_KEY":              {"the login and link rate limits", []string{"api"}},
	"TRUSTED_PROXY_CIDRS":         {"the client address behind Caddy", []string{"api"}},
	"ANALYTICS_PLATFORM_ENABLED":  {"the platform view, its builds and their schedule", []string{"api", "worker", "scheduler", "salusctl"}},
	"ANALYTICS_MIN_COUNT":         {"k of the platform figures", []string{"api", "worker", "scheduler", "salusctl"}},
}

// notFromExample are variables the compose file interpolates that the env file does not hold, with the reason.
var notFromExample = map[string]string{
	"SALUS_TAG": "the images' commit SHA, which the deploy writes to release.env",
}

func TestEveryVariableABinaryReadsIsPassedToItsService(t *testing.T) {
	loaders := configLoaders(t)
	services := composeServices(t)
	reads := binaryReads(t, loaders)
	if len(reads) == 0 {
		t.Fatal("no binary under cmd/ calls internal/config")
	}
	for _, bin := range sortedKeys(reads) {
		service, ok := extraService[bin]
		if !ok {
			service = serviceOf(services, bin)
		}
		env, ok := services[service]
		if service == "" || !ok {
			t.Errorf("cmd/%s reads its configuration but no service of %s runs it", bin, composeFile)
			continue
		}
		for _, v := range sortedVars(reads[bin]) {
			if _, never := neverInProduction[v.name]; never {
				continue
			}
			if o, narrowed := onlyFor[v.name]; narrowed && !slices.Contains(o.binaries, bin) {
				continue
			}
			if !slices.ContainsFunc(v.names(), func(n string) bool { _, ok := env.Environment[n]; return ok }) {
				t.Errorf("cmd/%s reads %s, which the %s service of %s does not pass", bin, strings.Join(v.names(), " or "), service, composeFile)
			}
		}
	}
}

// The hand-kept tables stay true: each entry is read by the code, for the binaries it names.
func TestTheExceptionsAreNotStale(t *testing.T) {
	reads := binaryReads(t, configLoaders(t))
	readBy := map[string][]string{}
	for bin, vars := range reads {
		for _, v := range vars {
			readBy[v.name] = append(readBy[v.name], bin)
		}
	}
	for name := range neverInProduction {
		if len(readBy[name]) == 0 {
			t.Errorf("neverInProduction names %s, which no binary reads", name)
		}
	}
	for name, o := range onlyFor {
		for _, bin := range o.binaries {
			if !slices.Contains(readBy[name], bin) {
				t.Errorf("onlyFor gives %s to cmd/%s, which does not read it", name, bin)
			}
		}
	}
	services := composeServices(t)
	for bin, service := range extraService {
		if _, ok := services[service]; !ok || reads[bin] == nil {
			t.Errorf("extraService puts cmd/%s in %s: no such service, or the binary reads nothing", bin, service)
		}
	}
}

var interpolation = regexp.MustCompile(`\$\{([A-Z0-9_]+)(:?[?-])?`)

// Every ${VAR} of the compose file is named in .env.production.example (the deploy warns about a key it does not
// name); a required one (${VAR:?}) is set there, not commented out, so the deploy refuses an env file without it.
// And the example names nothing the compose file does not read.
func TestEveryComposeVariableIsInTheExample(t *testing.T) {
	compose := read(t, composeFile)
	example := map[string]bool{} // name → set (true) or commented out (false)
	for _, line := range strings.Split(read(t, exampleFile), "\n") {
		if m := regexp.MustCompile(`^(# ?)?([A-Z0-9_]+)=`).FindStringSubmatch(line); m != nil {
			example[m[2]] = example[m[2]] || m[1] == ""
		}
	}
	used := map[string]bool{}
	for _, m := range interpolation.FindAllStringSubmatch(compose, -1) {
		name, required := m[1], strings.HasSuffix(m[2], "?")
		used[name] = true
		if _, ok := notFromExample[name]; ok {
			continue
		}
		set, named := example[name]
		switch {
		case !named:
			t.Errorf("%s uses ${%s}, which %s does not name", composeFile, name, exampleFile)
		case required && !set:
			t.Errorf("%s requires ${%s:?}, which %s has commented out", composeFile, name, exampleFile)
		}
	}
	for name := range example {
		if !used[name] {
			t.Errorf("%s names %s, which %s does not use", exampleFile, name, composeFile)
		}
	}
	for name := range notFromExample {
		if !used[name] {
			t.Errorf("notFromExample names %s, which %s does not use", name, composeFile)
		}
	}
}

// envVar is a variable a loader reads; a key setting may be given as NAME or as NAME_FILE.
type envVar struct {
	name string
	file bool
}

func (v envVar) names() []string {
	if v.file {
		return []string{v.name, v.name + "_FILE"}
	}
	return []string{v.name}
}

// configLoaders maps each function of internal/config to the variables it reads, its callees' included.
func configLoaders(t *testing.T) map[string][]envVar {
	t.Helper()
	direct, calls := map[string][]envVar{}, map[string][]string{}
	for _, f := range parseDir(t, filepath.Join(root, "internal/config")) {
		for _, d := range f.Decls {
			if fn, ok := d.(*ast.FuncDecl); ok && fn.Recv == nil {
				direct[fn.Name.Name], calls[fn.Name.Name] = funcReads(t, fn)
			}
		}
	}
	out := map[string][]envVar{}
	for name := range direct {
		seen := map[string]bool{}
		var walk func(string)
		walk = func(fn string) {
			if seen[fn] {
				return
			}
			seen[fn] = true
			out[name] = append(out[name], direct[fn]...)
			for _, c := range calls[fn] {
				walk(c)
			}
		}
		walk(name)
	}
	return out
}

// funcReads are the variables fn reads itself (os.Getenv, os.LookupEnv, keySetting) and the package functions it calls.
func funcReads(t *testing.T, fn *ast.FuncDecl) (reads []envVar, calls []string) {
	t.Helper()
	name := fn.Name.Name
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch fun := call.Fun.(type) {
		case *ast.SelectorExpr:
			x, ok := fun.X.(*ast.Ident)
			// keySetting reads its argument and argument_FILE: taken at each of its calls instead.
			if ok && x.Name == "os" && (fun.Sel.Name == "Getenv" || fun.Sel.Name == "LookupEnv") && name != "keySetting" {
				reads = append(reads, envVar{name: literal(t, name, call.Args[0])})
			}
		case *ast.Ident:
			if fun.Name == "keySetting" {
				reads = append(reads, envVar{name: literal(t, name, call.Args[0]), file: true})
			} else {
				calls = append(calls, fun.Name)
			}
		}
		return true
	})
	return reads, calls
}

// literal is the variable name a call passes; a name the test cannot read from the source fails it, so no read
// escapes the check.
func literal(t *testing.T, fn string, arg ast.Expr) string {
	t.Helper()
	if lit, ok := arg.(*ast.BasicLit); ok && lit.Kind == token.STRING {
		if s, err := strconv.Unquote(lit.Value); err == nil {
			return s
		}
	}
	t.Fatalf("internal/config.%s reads a variable whose name is not a string literal: write it as one, so this check sees it", fn)
	return ""
}

// binaryReads maps each binary under cmd/ to the variables its internal/config calls read. Only cmd/ calls the
// loaders (checked here too), so that is all a binary reads.
func binaryReads(t *testing.T, loaders map[string][]envVar) map[string][]envVar {
	t.Helper()
	dirs, err := os.ReadDir(filepath.Join(root, "cmd"))
	if err != nil {
		t.Fatal(err)
	}
	out := map[string][]envVar{}
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		for _, fn := range configCalls(t, filepath.Join(root, "cmd", d.Name()), loaders) {
			out[d.Name()] = append(out[d.Name()], loaders[fn]...)
		}
	}
	for _, dir := range []string{"internal", "pkg"} {
		_ = filepath.WalkDir(filepath.Join(root, dir), func(path string, e os.DirEntry, err error) error {
			if err != nil || !e.IsDir() || path == filepath.Join(root, "internal/config") {
				return err
			}
			if fns := configCalls(t, path, loaders); len(fns) > 0 {
				t.Errorf("%s calls internal/config's %v: only cmd/ may, so this check sees what each binary reads", path, fns)
			}
			return nil
		})
	}
	return out
}

// configCalls are the internal/config functions the non-test Go files of dir refer to.
func configCalls(t *testing.T, dir string, loaders map[string][]envVar) []string {
	t.Helper()
	var out []string
	for _, f := range parseDir(t, dir) {
		ast.Inspect(f, func(n ast.Node) bool {
			if sel, ok := n.(*ast.SelectorExpr); ok {
				if x, ok := sel.X.(*ast.Ident); ok && x.Name == "config" {
					if _, ok := loaders[sel.Sel.Name]; ok && !slices.Contains(out, sel.Sel.Name) {
						out = append(out, sel.Sel.Name)
					}
				}
			}
			return true
		})
	}
	return out
}

func parseDir(t *testing.T, dir string) []*ast.File {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var files []*ast.File
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), filepath.Join(dir, e.Name()), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, f)
	}
	return files
}

type service struct {
	Environment map[string]any `yaml:"environment"`
	Command     []string       `yaml:"command"`
}

// composeServices are the services of the production compose file, their anchors and merge keys resolved.
func composeServices(t *testing.T) map[string]service {
	t.Helper()
	var f struct {
		Services map[string]service `yaml:"services"`
	}
	if err := yaml.Unmarshal([]byte(read(t, composeFile)), &f); err != nil {
		t.Fatal(err)
	}
	return f.Services
}

// serviceOf is the service whose command runs /usr/local/bin/<bin>.
func serviceOf(services map[string]service, bin string) string {
	for name, s := range services {
		if len(s.Command) > 0 && s.Command[0] == "/usr/local/bin/"+bin {
			return name
		}
	}
	return ""
}

func read(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedVars(vars []envVar) []envVar {
	seen := map[string]envVar{}
	for _, v := range vars {
		seen[v.name] = v
	}
	out := make([]envVar, 0, len(seen))
	for _, k := range sortedKeys(seen) {
		out = append(out, seen[k])
	}
	return out
}
