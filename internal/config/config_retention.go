package config

import (
	"errors"
	"os"
	"strconv"
)

// RetentionKeep is the one RETENTION_CLINICAL_DATA taken: clinical data is never deleted automatically.
const RetentionKeep = "keep"

// The operational logs' period: 90 days by default, at least a day and at most ten years.
const (
	defaultOperationalLogDays = 90
	maxOperationalLogDays     = 3653
)

// RetentionPolicy is what the retention job's rules take from the environment (internal/retention.Policy). Pending
// lawyer L1–L3: both values are defaults until the owner's lawyer answers, and clinical data is kept (no automatic
// deletion) whatever is set.
type RetentionPolicy struct {
	// ClinicalData is what happens to clinical data once its period would run (RETENTION_CLINICAL_DATA): keep, the
	// default and, pending lawyer L1–L3, the only value taken.
	ClinicalData string
	// OperationalLogDays is how long the operational logs (when each reminder was due, which channel a notification
	// went through) are kept (RETENTION_OPERATIONAL_LOG_DAYS): 90 by default, pending lawyer L1–L3.
	OperationalLogDays int
}

// LoadRetentionPolicy reads the retention policy; cmd/worker, which runs the retention job, calls it. Errors name the
// variable, never a value.
func LoadRetentionPolicy() (RetentionPolicy, error) {
	p := RetentionPolicy{ClinicalData: RetentionKeep, OperationalLogDays: defaultOperationalLogDays}
	switch os.Getenv("RETENTION_CLINICAL_DATA") {
	case "", RetentionKeep:
	default:
		return RetentionPolicy{}, errors.New("config: RETENTION_CLINICAL_DATA takes only keep: no automatic deletion of clinical data, pending lawyer L1–L3")
	}
	if v := os.Getenv("RETENTION_OPERATIONAL_LOG_DAYS"); v != "" {
		days, err := strconv.Atoi(v)
		if err != nil || days < 1 || days > maxOperationalLogDays {
			return RetentionPolicy{}, errors.New("config: RETENTION_OPERATIONAL_LOG_DAYS is not a number of days from 1 to 3653")
		}
		p.OperationalLogDays = days
	}
	return p, nil
}
