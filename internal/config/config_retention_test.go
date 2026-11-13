package config_test

import (
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/config"
)

func TestLoadRetentionPolicy_KeepsClinicalDataAndOperationalLogs90Days(t *testing.T) {
	t.Setenv("RETENTION_CLINICAL_DATA", "")
	t.Setenv("RETENTION_OPERATIONAL_LOG_DAYS", "")
	p, err := config.LoadRetentionPolicy()
	if err != nil || p.ClinicalData != config.RetentionKeep || p.OperationalLogDays != 90 {
		t.Fatalf("defaults = %+v, %v; want keep and 90 days", p, err)
	}
	t.Setenv("RETENTION_CLINICAL_DATA", "keep")
	t.Setenv("RETENTION_OPERATIONAL_LOG_DAYS", "120")
	if p, err := config.LoadRetentionPolicy(); err != nil || p.ClinicalData != config.RetentionKeep || p.OperationalLogDays != 120 {
		t.Errorf("set = %+v, %v; want keep and 120 days", p, err)
	}
}

// No automatic deletion of clinical data is taken while the lawyer's answers are pending, and a period outside 1 day
// to 10 years is refused; errors name the variable, never the value.
func TestLoadRetentionPolicy_RefusesClinicalDeletionAndPeriodsOutOfRange(t *testing.T) {
	for _, tt := range []struct{ name, value string }{
		{"RETENTION_CLINICAL_DATA", "delete"}, {"RETENTION_CLINICAL_DATA", "restrict"},
		{"RETENTION_OPERATIONAL_LOG_DAYS", "0"}, {"RETENTION_OPERATIONAL_LOG_DAYS", "3654"}, {"RETENTION_OPERATIONAL_LOG_DAYS", "ninety"},
	} {
		t.Setenv("RETENTION_CLINICAL_DATA", "")
		t.Setenv("RETENTION_OPERATIONAL_LOG_DAYS", "")
		t.Setenv(tt.name, tt.value)
		if _, err := config.LoadRetentionPolicy(); err == nil || !strings.Contains(err.Error(), tt.name) || strings.Contains(err.Error(), tt.value) {
			t.Errorf("%s=%s: %v, want refused naming the variable only", tt.name, tt.value, err)
		}
	}
}
