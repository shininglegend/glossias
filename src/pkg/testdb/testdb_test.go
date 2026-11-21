//go:build integration

package testdb

import (
	"context"
	"os"
	"testing"

	"glossias/src/pkg/models"
)

func TestMain(m *testing.M) {
	if err := models.SetCache(); err != nil {
		panic(err)
	}
	code := m.Run()
	Terminate()
	os.Exit(code)
}

func TestSeededCounts(t *testing.T) {
	f := Shared(t)
	ctx := context.Background()
	for _, tbl := range f.Data.Tables {
		var n int
		if err := f.Pool.QueryRow(ctx, "SELECT count(*) FROM "+tbl.Name).Scan(&n); err != nil {
			t.Fatalf("%s: %v", tbl.Name, err)
		}
		// Migrations insert no rows into seeded tables.
		if n != len(tbl.Rows) {
			t.Errorf("%s has %d rows, want %d", tbl.Name, n, len(tbl.Rows))
		}
	}
}

func TestFreshIsIsolated(t *testing.T) {
	ctx := context.Background()
	var before int
	if err := Shared(t).Pool.QueryRow(ctx, "SELECT count(*) FROM stories").Scan(&before); err != nil {
		t.Fatal(err)
	}
	t.Run("write", func(t *testing.T) {
		f := Fresh(t)
		if _, err := f.Pool.Exec(ctx, "DELETE FROM stories"); err != nil {
			t.Fatal(err)
		}
	})
	var after int
	if err := Shared(t).Pool.QueryRow(ctx, "SELECT count(*) FROM stories").Scan(&after); err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Fatalf("shared stories = %d after a Fresh write, want %d", after, before)
	}
}
