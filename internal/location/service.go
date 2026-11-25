package location

import (
	"context"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// Transactor runs fn in a transaction that travels in the context (postgres.Transactor).
type Transactor interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Auditor records an AuditEvent inside the caller's transaction (auditevent.Service).
type Auditor interface {
	Record(ctx context.Context, e auditevent.AuditEvent) error
}

// Ports are what the Service consumes.
type Ports struct {
	Repo    Repository
	Clock   clock.Clock
	Audit   Auditor    // the writes' AuditEvents (LC1, RM1)
	Tx      Transactor // LC1, RM1
	Clinics Clinics    // LC1: the directory's listed clinics
}

// Service is the locations' business rules.
type Service struct {
	p Ports
}

// NewService builds a Service on p.
func NewService(p Ports) *Service {
	return &Service{p: p}
}
