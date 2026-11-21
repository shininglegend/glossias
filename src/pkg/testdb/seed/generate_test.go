package seed

import (
	"reflect"
	"strings"
	"testing"
)

func TestGenerateDeterministic(t *testing.T) {
	a, b := Generate(Realistic), Generate(Realistic)
	if !reflect.DeepEqual(a, b) {
		t.Fatal("Generate(Realistic) differs between runs")
	}
}

func TestGenerateInvariants(t *testing.T) {
	for name, spec := range map[string]Spec{"small": Small, "realistic": Realistic} {
		t.Run(name, func(t *testing.T) {
			ds := Generate(spec)

			if got := len(ds.Stories); got != spec.Stories {
				t.Fatalf("stories = %d, want %d", got, spec.Stories)
			}
			for _, kind := range Kinds {
				if len(ds.Students[kind]) != spec.StudentsPerKind {
					t.Errorf("students of kind %s = %d, want %d", kind, len(ds.Students[kind]), spec.StudentsPerKind)
				}
			}
			if _, ok := ds.TrialStory(); !ok {
				t.Error("no trial story")
			}
			if spec.MultiCourseFraction > 0 {
				if _, ok := ds.MultiCourseStory(); !ok {
					t.Error("no multi-course story")
				}
			}
			if len(ds.Progress) == 0 || len(ds.Archived) == 0 {
				t.Error("no progress or archived attempts")
			}

			for _, tbl := range ds.Tables {
				for _, row := range tbl.Rows {
					if len(row) != len(tbl.Columns) {
						t.Fatalf("%s row has %d values for %d columns", tbl.Name, len(row), len(tbl.Columns))
					}
				}
			}

			// Line numbers are contiguous from 1 per story.
			next := map[int32]int32{}
			text := map[[2]int32]string{}
			for _, row := range ds.Table("story_lines").Rows {
				sid, ln := row[0].(int32), row[1].(int32)
				if ln != next[sid]+1 {
					t.Fatalf("story %d line %d follows %d", sid, ln, next[sid])
				}
				next[sid] = ln
				text[[2]int32{sid, ln}] = row[2].(string)
			}

			// Vocabulary positions point at the word in the line text.
			for _, row := range ds.Table("vocabulary_items").Rows {
				line := text[[2]int32{row[1].(int32), row[2].(int32)}]
				start, end := row[5].(int32), row[6].(int32)
				if int(end) > len(line) || line[start:end] != row[3].(string) {
					t.Fatalf("vocab %v does not match line %q", row, line)
				}
			}

			for _, row := range ds.Table("recall_sentences").Rows {
				if o := row[2].(int32); o < 1 || o > 5 {
					t.Fatalf("recall sequence_order %d out of range", o)
				}
			}
			for _, row := range ds.Table("produce_segments").Rows {
				if o := row[2].(int32); o < 1 || o > 2 {
					t.Fatalf("produce segment_order %d out of range", o)
				}
			}

			emails := map[string]bool{}
			for _, row := range ds.Table("users").Rows {
				e := row[1].(string)
				if emails[e] || !strings.Contains(e, "@") {
					t.Fatalf("bad or duplicate email %q", e)
				}
				emails[e] = true
			}
		})
	}
}
