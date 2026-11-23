package chargeitemdefinition

import (
	"context"
	"errors"
)

// ImportFile is a price list to import (package O2 of docs/clinic-cms): a CSV or XLSX file of services (name, code,
// price, duration, VAT rate, visit types), told apart by its bytes.
type ImportFile struct {
	Name string
	Data []byte
}

// ImportRow is one row of an import as read: the service it would store and the codes of what is wrong with it
// (empty when it is valid).
type ImportRow struct {
	Errors  []string
	Service Definition
	Line    int // the row's line in the file, from 1
}

// Import reads f into the admin caller's clinic's price list. With save false it only previews every row and its
// errors; with save true it stores all rows or none. The wave 3 contract: refused until package O2.
func (s *Service) Import(ctx context.Context, c Caller, f ImportFile, save bool) ([]ImportRow, error) {
	return nil, errors.ErrUnsupported
}
