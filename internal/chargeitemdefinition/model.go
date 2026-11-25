// Package chargeitemdefinition is the clinic's price list: FHIR R4 ChargeItemDefinition (docs/plans/clinic-cms.md,
// package B; ADR 0016). A service has a name in the clinic's language with optional translations for patients, an
// optional code, a price in the currency's minor unit, a duration, a VAT rate (none: exempt, Q9), the visit modes it is
// offered in, and whether it is active. Prices are shown at booking and paid at the clinic; a receipt copies its lines
// (internal/invoice), so a price change never changes an issued receipt. Nothing here is personal data.
//
// Step 0 holds the contract: the model, the ports and the service's methods, each refused with errors.ErrUnsupported
// until package B builds it.
package chargeitemdefinition

import (
	"time"

	"github.com/Shmeeshmee/salus/internal/schedule"
)

// Definition is one service of a clinic's price list.
type Definition struct {
	LastUpdated     time.Time // meta.lastUpdated
	VATRateBP       *int32    // basis points (1700 is 17 %); nil for exempt
	ID              string
	OrganizationID  string
	Name            string // in the clinic's language
	Code            string // the clinic's own code; empty for none
	DeviceID        string // the device a visit for it needs, of the clinic; empty for none (WAVE6.md U4, RM1)
	Currency        string // ISO 4217, BAM
	Translations    []Translation
	Modes           []schedule.VisitMode // the modes it is offered in, at least one
	PriceMinor      int64
	DurationMinutes int32 // zero: the slot's length
	VersionID       int32 // meta.versionId
	Active          bool
}

// Translation is a service's name in a patient's language.
type Translation struct {
	Language string // a BCP 47 tag of the language enum
	Name     string
}

// Pick is a service chosen for a receipt: how many and the discount on the line, in the minor unit.
type Pick struct {
	DefinitionID  string
	DiscountMinor int64
	Quantity      int32
}

// Line is a picked service as a receipt copies it at issue: its name, unit price and VAT rate then, and the line's
// total (quantity times the unit price, less the discount).
type Line struct {
	VATRateBP      *int32
	DefinitionID   string
	Name           string
	Currency       string
	UnitPriceMinor int64
	DiscountMinor  int64
	TotalMinor     int64
	Quantity       int32
}

// Caller is who asks: the signed-in account, its clinic and whether it is the clinic's admin.
type Caller struct {
	AccountID      string
	OrganizationID string
	Admin          bool
}
