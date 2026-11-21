package seed

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Load copies ds into the database behind pool, which must have the goose
// migrations applied and no seeded rows yet. It bypasses models (and so the
// query counters and cache) entirely.
func Load(ctx context.Context, pool *pgxpool.Pool, ds *Dataset) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, t := range ds.Tables {
		if len(t.Rows) == 0 {
			continue
		}
		n, err := tx.CopyFrom(ctx, pgx.Identifier{t.Name}, t.Columns, pgx.CopyFromRows(t.Rows))
		if err != nil {
			return fmt.Errorf("seed %s: %w", t.Name, err)
		}
		if int(n) != len(t.Rows) {
			return fmt.Errorf("seed %s: copied %d of %d rows", t.Name, n, len(t.Rows))
		}
		if t.Serial != "" {
			q := fmt.Sprintf("SELECT setval(pg_get_serial_sequence('%s', '%s'), (SELECT max(%s) FROM %s))",
				t.Name, t.Serial, t.Serial, t.Name)
			if _, err := tx.Exec(ctx, q); err != nil {
				return fmt.Errorf("seed %s sequence: %w", t.Name, err)
			}
		}
	}

	// Snapshots reference story_attempts by identity values the database
	// assigned; they line up only if the table started empty.
	if t := ds.Table("story_attempts"); t != nil && len(t.Rows) > 0 {
		var maxID int64
		if err := tx.QueryRow(ctx, "SELECT max(attempt_id) FROM story_attempts").Scan(&maxID); err != nil {
			return err
		}
		if maxID != int64(len(t.Rows)) {
			return fmt.Errorf("seed: story_attempts ids end at %d, want %d (table was not empty)", maxID, len(t.Rows))
		}
	}
	return tx.Commit(ctx)
}
