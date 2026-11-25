// Package location is a clinic's sites with their rooms and devices (FHIR Location, a Location partOf its site, and
// Device; docs/clinic-cms/WAVE6.md U1, U3, U4). Not personal data.
package location

// Location is one site of a clinic: where its in-person visits take place (U1). A video visit has none.
type Location struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organizationId"`
	Name           string `json:"name"`
	AddressLine    string `json:"addressLine,omitempty"`
	PostalCode     string `json:"postalCode,omitempty"`
	CityID         string `json:"cityId,omitempty"` // a place (Wikidata item), as the clinic's city
	Phone          string `json:"phone,omitempty"`  // E.164
	TimeZone       string `json:"timeZone"`         // IANA, the clinic's by default
	VersionID      int    `json:"versionId"`
	Active         bool   `json:"active"`
}

// Room is a room of a location an in-person rule may name (U3). Two practitioners' in-person hours never overlap in
// one room.
type Room struct {
	ID         string `json:"id"`
	LocationID string `json:"locationId"`
	Name       string `json:"name"`
	VersionID  int    `json:"versionId"`
	Active     bool   `json:"active"`
}

// Device is a device of a location a price-list service may require (U4). Two visits that need one device never
// overlap.
type Device struct {
	ID         string `json:"id"`
	LocationID string `json:"locationId"`
	Name       string `json:"name"`
	VersionID  int    `json:"versionId"`
	Active     bool   `json:"active"`
}

// Staff is the caller of the clinic's routes: only its admin writes.
type Staff struct {
	AccountID      string
	OrganizationID string
	Admin          bool
}
