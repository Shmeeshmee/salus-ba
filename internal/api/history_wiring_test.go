package api_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The summary's allergies and medications stand-ins (history.go) fail closed and are not to reach main: once
// SAL-41's services are on this branch (their Seen method, the history.Allergies and history.Medications ports),
// this fails until they are wired in api.New and the stand-ins deleted, with this test.
func TestHistory_TheProfileStandInsGoOnceSAL41IsHere(t *testing.T) {
	t.Parallel()
	for _, dir := range []string{"../allergyintolerance", "../medicationstatement"} {
		files, err := filepath.Glob(filepath.Join(dir, "*.go"))
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range files {
			if strings.HasSuffix(f, "_test.go") {
				continue
			}
			b, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(b), ") Seen(ctx context.Context") {
				t.Errorf("%s has the Seen of SAL-41: wire it in api.New in place of the stand-in, and delete the stand-ins", f)
			}
		}
	}
}
