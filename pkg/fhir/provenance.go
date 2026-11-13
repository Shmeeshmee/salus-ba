package fhir

import (
	"strconv"
	"time"
)

// The code systems of a write's Provenance.
const (
	SystemDataOperation   = "http://terminology.hl7.org/CodeSystem/v3-DataOperation"
	SystemParticipantType = "http://terminology.hl7.org/CodeSystem/provenance-participant-type"
)

// The activities of a write (v3-DataOperation): NULLIFY marks the resource as entered in error, kept but treated as
// though it never existed.
const (
	ActivityCreate  = "CREATE"
	ActivityUpdate  = "UPDATE"
	ActivityNullify = "NULLIFY"
)

// The agent types of a write (provenance-participant-type).
const (
	AgentAuthor   = "author"
	AgentVerifier = "verifier"
)

// Change is one write of a resource version by an agent: what a Provenance of it says.
type Change struct {
	Recorded   time.Time
	ID         string // the Provenance's id
	Resource   string // the resource written, ResourceType/id
	Activity   string // a v3-DataOperation code
	AgentType  string // a provenance-participant-type code
	Who        string // the agent, ResourceType/id
	OnBehalfOf string // the organization the agent acted for, Organization/id
	Version    int32  // the version the write produced
}

// Provenance is an R4 Provenance of one write.
type Provenance struct {
	ResourceType string             `json:"resourceType"`
	ID           string             `json:"id"`
	Meta         Meta               `json:"meta"`
	Target       []Reference        `json:"target"`
	Recorded     string             `json:"recorded"`
	Activity     CodeableConcept    `json:"activity"`
	Agent        []ProvenanceAgent  `json:"agent"`
	Entity       []ProvenanceEntity `json:"entity,omitempty"`
}

// ProvenanceAgent is a Provenance's agent: who, in which part, on behalf of which organization.
type ProvenanceAgent struct {
	OnBehalfOf *Reference      `json:"onBehalfOf,omitempty"`
	Who        Reference       `json:"who"`
	Type       CodeableConcept `json:"type"`
}

// ProvenanceEntity is a resource a Provenance's activity used, in its role.
type ProvenanceEntity struct {
	Role string    `json:"role"`
	What Reference `json:"what"`
}

// NewWriteProvenance is the Provenance of w, written once (version 1): its target the version written, and for any
// write but a creation the version it revised as an entity.
func NewWriteProvenance(w Change) Provenance {
	version := func(v int32) Reference {
		return Reference{Reference: w.Resource + "/_history/" + strconv.FormatInt(int64(v), 10)}
	}
	p := Provenance{
		ResourceType: "Provenance", ID: w.ID, Meta: NewMeta(1, w.Recorded),
		Target: []Reference{version(w.Version)}, Recorded: w.Recorded.UTC().Format(time.RFC3339),
		Activity: CodeableConcept{Coding: []Coding{{System: SystemDataOperation, Code: w.Activity}}},
		Agent: []ProvenanceAgent{{
			Type: CodeableConcept{Coding: []Coding{{System: SystemParticipantType, Code: w.AgentType}}},
			Who:  Reference{Reference: w.Who}, OnBehalfOf: &Reference{Reference: w.OnBehalfOf},
		}},
	}
	if w.Activity != ActivityCreate {
		p.Entity = []ProvenanceEntity{{Role: "revision", What: version(w.Version - 1)}}
	}
	return p
}
