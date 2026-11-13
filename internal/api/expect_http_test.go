package api_test

import (
	"slices"
	"testing"
)

// expectHTTP fails on every missing string: two missing ones must not cancel out (the status was once negated
// per missing string, so an even number passed).
func TestMissing_GivesEveryStringTheBodyLacks(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		body string
		want []string
		gone []string
	}{
		{body: `{"a":1,"b":2}`, want: []string{`"a":1`, `"b":2`}},
		{body: `{"a":1}`, want: []string{`"x"`}, gone: []string{`"x"`}},
		{body: `{"a":1}`, want: []string{`"x"`, `"y"`}, gone: []string{`"x"`, `"y"`}},
		{body: `{"a":1}`, want: []string{`"x"`, `"a":1`, `"y"`}, gone: []string{`"x"`, `"y"`}},
	} {
		if got := missing(tt.body, tt.want); !slices.Equal(got, tt.gone) {
			t.Errorf("missing(%s, %q) = %q, want %q", tt.body, tt.want, got, tt.gone)
		}
	}
}
