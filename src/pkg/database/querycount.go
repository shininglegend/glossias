package database

import (
	"context"
	"sync/atomic"
)

// Per-request DB query counting.
//
// A counter attached to a request's context is incremented by every SQLC call
// routed through models.ContextTxRouter. The HTTP layer logs the total per
// request and warns when it is high; handler tests assert a budget with
// assertQueryBudget. Together these catch N+1 patterns automatically instead
// of relying on anyone knowing which model functions are expensive.
//
// Work done outside the request context (context.Background in background
// goroutines) is intentionally not counted.

type queryCountKey struct{}

// WithQueryCounter attaches a fresh counter to ctx and returns it alongside
// the derived context.
func WithQueryCounter(ctx context.Context) (context.Context, *atomic.Int32) {
	c := new(atomic.Int32)
	return context.WithValue(ctx, queryCountKey{}, c), c
}

// CountQuery records one DB call against the counter on ctx, if any.
func CountQuery(ctx context.Context) {
	if c, ok := ctx.Value(queryCountKey{}).(*atomic.Int32); ok {
		c.Add(1)
	}
}

// QueryCount returns the number of DB calls recorded on ctx, or -1 when no
// counter is attached (so an absent counter is never mistaken for zero).
func QueryCount(ctx context.Context) int {
	if c, ok := ctx.Value(queryCountKey{}).(*atomic.Int32); ok {
		return int(c.Load())
	}
	return -1
}
