//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"maps"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"
	"text/tabwriter"

	"glossias/src/pkg/models"
	"glossias/src/pkg/testdb"
	"glossias/src/pkg/testutil"
)

var updateBaseline = flag.Bool("update-baseline", false, "rewrite testdata/query_baseline.json with the measured counts")

const baselinePath = "testdata/query_baseline.json"

// Measurement is one baseline entry.
type Measurement struct {
	Queries int `json:"queries"`
	Status  int `json:"status"`
}

// TestQueryBudgets serves every case cold (cache cleared) and, for reads,
// warm (after one unmeasured request), and compares each count with the
// committed baseline: the baseline is the budget. It fails when a count goes
// up or a status changes. -update-baseline records the current numbers;
// lowering them is the point of a query-count fix, so commit the new file.
//
// -v prints the full table with the change against the baseline;
// QUERY_REPORT_MD=path also writes it as Markdown.
func TestQueryBudgets(t *testing.T) {
	fx := testdb.Shared(t)
	for _, tbl := range fx.Data.Tables {
		if tbl.Name != "users" {
			continue
		}
		for _, r := range tbl.Rows {
			seedUsers[r[0].(string)] = [2]string{r[1].(string), r[2].(string)}
		}
	}
	f := newFixtures(t, fx.Data)

	baseline := map[string]Measurement{}
	if raw, err := os.ReadFile(baselinePath); err == nil {
		if err := json.Unmarshal(raw, &baseline); err != nil {
			t.Fatalf("read %s: %v", baselinePath, err)
		}
	} else if !*updateBaseline {
		t.Fatalf("no %s; run with -update-baseline first", baselinePath)
	}

	got := map[string]Measurement{}
	seen := map[string]bool{}
	for _, c := range f.cases() {
		if seen[c.Name] {
			t.Fatalf("duplicate case name %s", c.Name)
		}
		seen[c.Name] = true
		t.Run(c.Name, func(t *testing.T) {
			if c.Mutates {
				testdb.Fresh(t)
			} else {
				testdb.Shared(t)
			}
			if c.Paywall {
				t.Setenv("PAYWALL_ENABLED", "true")
			}
			if models.PaymentsPaused() {
				t.Fatal("payments are paused, which lifts the paywall; the case would measure the wrong path")
			}

			models.ClearAllCache()
			got[c.Name+"/cold"] = serve(t, c)
			if !c.Mutates {
				serve(t, c) // warm the cache
				got[c.Name+"/warm"] = serve(t, c)
			}
		})
	}

	if *updateBaseline {
		raw, err := json.MarshalIndent(got, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(baselinePath, append(raw, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %s (%d entries)", baselinePath, len(got))
	} else {
		for _, k := range slices.Sorted(maps.Keys(got)) {
			b, ok := baseline[k]
			g := got[k]
			switch {
			case !ok:
				t.Errorf("%s: not in the baseline (%d queries); run with -update-baseline", k, g.Queries)
			case g.Queries > b.Queries:
				t.Errorf("%s: %d DB queries, budget %d (possible N+1)", k, g.Queries, b.Queries)
			case g.Status != b.Status:
				t.Errorf("%s: status %d, baseline %d", k, g.Status, b.Status)
			}
		}
	}
	report(t, got, baseline)
}

func serve(t *testing.T, c Case) Measurement {
	t.Helper()
	body := c.Body
	if c.Prepare != nil {
		body = c.Prepare(t)
	}
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(c.Method, c.Path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if c.As != "" {
		req.Header.Set(testUserHeader, c.As)
	}
	rr, n := testutil.ServeCounted(router, req)
	switch {
	case c.KnownBug != "":
		if rr.Code >= 500 {
			t.Logf("known bug (status %d): %s", rr.Code, c.KnownBug)
		}
	case c.Want != 0 && rr.Code != c.Want:
		t.Errorf("%s %s as %s: status %d, want %d: %s", c.Method, c.Path, c.As, rr.Code, c.Want, truncate(rr.Body.String()))
	case c.Want == 0 && rr.Code >= 500:
		t.Errorf("%s %s as %s: status %d: %s", c.Method, c.Path, c.As, rr.Code, truncate(rr.Body.String()))
	}
	return Measurement{Queries: n, Status: rr.Code}
}

func truncate(s string) string {
	if len(s) > 300 {
		return s[:300] + "..."
	}
	return s
}

// report prints the measured table (always with -v) and writes Markdown to
// QUERY_REPORT_MD when set.
func report(t *testing.T, got, baseline map[string]Measurement) {
	var b strings.Builder
	tw := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "CASE\tQUERIES\tBASELINE\tΔ\tSTATUS")
	var md strings.Builder
	md.WriteString("| Case | Queries | Baseline | Δ | Status |\n|---|---:|---:|---:|---:|\n")
	total, totalBase := 0, 0
	for _, k := range slices.Sorted(maps.Keys(got)) {
		g := got[k]
		base, delta := "-", ""
		if m, ok := baseline[k]; ok {
			base = fmt.Sprint(m.Queries)
			if d := g.Queries - m.Queries; d != 0 {
				delta = fmt.Sprintf("%+d", d)
			}
			totalBase += m.Queries
		}
		total += g.Queries
		fmt.Fprintf(tw, "%s\t%d\t%s\t%s\t%d\n", k, g.Queries, base, delta, g.Status)
		fmt.Fprintf(&md, "| %s | %d | %s | %s | %d |\n", k, g.Queries, base, delta, g.Status)
	}
	fmt.Fprintf(tw, "TOTAL\t%d\t%d\t%+d\t\n", total, totalBase, total-totalBase)
	fmt.Fprintf(&md, "| **Total** | %d | %d | %+d | |\n", total, totalBase, total-totalBase)
	tw.Flush()
	t.Log("\n" + b.String())
	if path := os.Getenv("QUERY_REPORT_MD"); path != "" {
		if err := os.WriteFile(path, []byte(md.String()), 0o644); err != nil {
			t.Errorf("write %s: %v", path, err)
		}
	}
}
