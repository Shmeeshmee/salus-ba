package datarequest

import (
	"context"

	"github.com/Shmeeshmee/salus/pkg/postgres"
)

// erasureScope sets the erasure scope of 00075 on the transaction in ctx.
type erasureScope struct{}

// NewErasureScope returns the Scope of the closure and the erasure on Postgres.
func NewErasureScope() Scope { return erasureScope{} }

func (erasureScope) SetErasure(ctx context.Context) error {
	return postgres.SetLocal(ctx, "app.erasure", "on")
}
