package config

import (
	"errors"
	"os"
)

// Fiscal is the fiscalization engine of receipts (docs/plans/clinic-cms.md, ADR 0016): the fake (pkg/fiscal/fiscaltest)
// only. The ESET adapter, and the engine and credentials that turn it on, come with package G once the owner has the
// Tax Administration's technical specification and test access (Q6). No binary loads it before package C.
type Fiscal struct {
	Engine string // FISCAL_ENGINE: fake, the default and the only one taken (FiscalESET is named, not taken)
}

// The engines LoadFiscal knows. FiscalFake is the one it takes; FiscalESET is the Tax Administration's system the
// adapter is planned for, refused until package G writes it.
const (
	FiscalFake = "fake"
	FiscalESET = "eset"
)

// LoadFiscal reads the fiscalization engine and refuses any but the fake. Errors name the variable, never a value.
func LoadFiscal() (Fiscal, error) {
	switch os.Getenv("FISCAL_ENGINE") {
	case "", FiscalFake:
		return Fiscal{Engine: FiscalFake}, nil
	case FiscalESET:
		return Fiscal{}, errors.New("config: FISCAL_ENGINE=eset waits for its adapter (docs/plans/clinic-cms.md, package G)")
	}
	return Fiscal{}, errors.New("config: FISCAL_ENGINE is not fake; no fiscalization engine exists yet (docs/plans/clinic-cms.md)")
}
