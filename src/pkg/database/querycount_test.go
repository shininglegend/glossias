package database

import (
	"context"
	"testing"
)

func TestQueryCounter(t *testing.T) {
	ctx, c := WithQueryCounter(context.Background())

	if got := QueryCount(ctx); got != 0 {
		t.Fatalf("fresh counter: got %d, want 0", got)
	}

	CountQuery(ctx)
	CountQuery(ctx)
	CountQuery(ctx)

	if got := QueryCount(ctx); got != 3 {
		t.Errorf("after 3 calls: got %d, want 3", got)
	}
	if got := c.Load(); got != 3 {
		t.Errorf("returned counter: got %d, want 3", got)
	}

	// A derived context shares the same counter.
	child := context.WithValue(ctx, struct{}{}, "x")
	CountQuery(child)
	if got := QueryCount(ctx); got != 4 {
		t.Errorf("after child call: got %d, want 4", got)
	}
}

func TestQueryCounterAbsent(t *testing.T) {
	ctx := context.Background()

	// Must not panic.
	CountQuery(ctx)

	if got := QueryCount(ctx); got != -1 {
		t.Errorf("no counter: got %d, want -1", got)
	}
}
