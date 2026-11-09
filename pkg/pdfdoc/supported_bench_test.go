package pdfdoc_test

import (
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/pkg/pdfdoc"
)

// What Supported costs per call, parsing both fonts each time (PR #39 review): a field's value of about 500
// characters. Run with go test -bench Supported ./pkg/pdfdoc/.
func BenchmarkSupported(b *testing.B) {
	s := strings.Repeat("Placeholder čćđšž ћирилица ", 20)
	for b.Loop() {
		if !pdfdoc.Supported(s) {
			b.Fatal("unsupported")
		}
	}
}
