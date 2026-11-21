// Package fiscaltest is the in-memory fiscal.Reporter: the only engine of every binary until package G adds the ESET
// adapter (internal/config LoadFiscal), and the fake of the tests. It reports nothing anywhere; a receipt it
// fiscalizes is not a legal receipt. Safe for concurrent use.
package fiscaltest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync"

	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/fiscal"
)

// Reporter fiscalizes every receipt that passes fiscal.Check with deterministic test values derived from its
// Reference: the identifier "TEST-<hash>", a code and a QR payload, at Clock's time. A reference reported again gets
// its first answer and is not fiscalized twice, as an authority that deduplicates on the reference does. A reversal
// must name the fiscal id of a receipt this Reporter fiscalized and not reversed yet (fiscal.ErrRejected otherwise).
// Err, when set, fails every call that passes the checks. Every call that passes the checks is counted per reference
// (Reported).
type Reporter struct {
	Clock clock.Clock
	Err   error
	byRef map[string]fiscal.Fiscalized
	byID  map[string]bool
	calls map[string]int
	mu    sync.Mutex
}

// Report fiscalizes a receipt.
func (r *Reporter) Report(_ context.Context, rc fiscal.Receipt) (fiscal.Fiscalized, error) {
	if rc.Reverses != "" {
		return fiscal.Fiscalized{}, fiscal.ErrInvalidReceipt
	}
	return r.fiscalize(rc)
}

// Reverse fiscalizes a reversal of a receipt this Reporter fiscalized.
func (r *Reporter) Reverse(_ context.Context, rc fiscal.Receipt) (fiscal.Fiscalized, error) {
	if rc.Reverses == "" {
		return fiscal.Fiscalized{}, fiscal.ErrInvalidReceipt
	}
	return r.fiscalize(rc)
}

func (r *Reporter) fiscalize(rc fiscal.Receipt) (fiscal.Fiscalized, error) {
	if err := fiscal.Check(rc); err != nil {
		return fiscal.Fiscalized{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.calls == nil {
		r.calls, r.byRef, r.byID = map[string]int{}, map[string]fiscal.Fiscalized{}, map[string]bool{}
	}
	r.calls[rc.Reference]++
	if r.Err != nil {
		return fiscal.Fiscalized{}, r.Err
	}
	if f, ok := r.byRef[rc.Reference]; ok {
		return f, nil
	}
	if rc.Reverses != "" && !r.byID[rc.Reverses] {
		return fiscal.Fiscalized{}, fiscal.ErrRejected // unknown, or reversed already
	}
	if rc.Reverses != "" {
		r.byID[rc.Reverses] = false
	}
	sum := sha256.Sum256([]byte(rc.Reference))
	h := hex.EncodeToString(sum[:])
	f := fiscal.Fiscalized{At: r.Clock.Now(), ID: "TEST-" + h[:16], Code: h[16:32], QR: "TEST:" + h[:32]}
	r.byRef[rc.Reference], r.byID[f.ID] = f, true
	return f, nil
}

// Reported is how many calls reported the reference, the failed ones included.
func (r *Reporter) Reported(reference string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls[reference]
}

// Fiscalized is how many receipts were fiscalized.
func (r *Reporter) Fiscalized() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.byRef)
}
