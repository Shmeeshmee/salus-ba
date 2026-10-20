// Package fhir embeds the FHIR contract files the API serves.
package fhir

import _ "embed"

//go:embed CapabilityStatement.json
var capabilityStatement string

// CapabilityStatement returns the CapabilityStatement served at /fhir/R4/metadata.
func CapabilityStatement() string { return capabilityStatement }
