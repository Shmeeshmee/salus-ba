package clinicstats

import (
	"context"
	"time"
)

// BuildRepository writes the aggregates, on salus_stats, in the transaction in the context.
type BuildRepository interface {
	// Lock takes the clinic's refresh lock until the transaction ends, so two refreshes of one clinic never
	// interleave.
	Lock(ctx context.Context, orgID string) error
	// Replace deletes the clinic's rows of the days first to last (dates, midnight UTC) and inserts these.
	Replace(ctx context.Context, orgID string, first, last time.Time, appts []AppointmentCount, diags []DiagnosisCount) error
}

// Repository reads the aggregates, on salus_app, in the clinic's scope set on the transaction in the context.
type Repository interface {
	SetClinic(ctx context.Context, orgID string) error
	Series(ctx context.Context, orgID string, q AppointmentQuery) ([]Period, error)
	Heatmap(ctx context.Context, orgID string, q AppointmentQuery) ([]HeatCell, error)
	Practitioners(ctx context.Context, orgID string, q AppointmentQuery) ([]PractitionerTotal, error)
	Diagnoses(ctx context.Context, orgID string, q DiagnosisQuery) ([]Count, error)
}

// Queue enqueues a clinic-day's refresh in the transaction in the context: the producers' StatsQueue port
// (appointment, composition).
type Queue interface {
	EnqueueStatsRefresh(ctx context.Context, orgID string, day time.Time) error
}
