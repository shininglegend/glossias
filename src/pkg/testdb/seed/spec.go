// Package seed generates a deterministic, realistic dataset for integration
// tests: courses (paid and trial), many stories with full line content,
// students in every paywall state, admins, progress and archived attempts.
//
// Generate is pure and in-memory; Load bulk-copies a Dataset into a database
// that already has the goose migrations applied.
package seed

import "time"

// Spec controls the size and shape of a generated dataset.
type Spec struct {
	// Seed fixes the random stream: the same Spec always yields the same rows.
	Seed uint64
	// Now anchors every timestamp. Entitlement expiry is relative to it, so
	// set it to time.Now() when the data is loaded for a test run (the
	// paywall compares against the database clock).
	Now time.Time

	PaidCourses  int
	TrialCourses int

	Stories int
	// Every TrialEvery-th story belongs to a trial course (0: none).
	TrialEvery int
	// Fraction of paid stories also linked to a second paid course.
	MultiCourseFraction float64

	LinesMin, LinesMax int
	VocabPerLine       [2]int  // min, max vocabulary items per line
	GrammarPoints      [2]int  // min, max grammar points per story
	GrammarItemChance  float64 // chance a line has a grammar item
	FootnoteEvery      int     // one footnote per N lines (0: none)
	TargetVocab        int     // target vocabulary words per story (identify/recall)
	RecallSentences    int     // 0..5
	ProduceSegments    int     // 0..2

	// Students per kind (see Kind).
	StudentsPerKind int
	// Fraction of (paid or unpaid student, story) pairs with exercise progress.
	ProgressFraction float64
	// Fraction of progress pairs that also have an archived (completed) attempt.
	ArchivedFraction float64
}

// Small is a tiny dataset for quick iteration.
var Small = Spec{
	Seed:                20261121,
	PaidCourses:         1,
	TrialCourses:        1,
	Stories:             4,
	TrialEvery:          4,
	MultiCourseFraction: 0,
	LinesMin:            5,
	LinesMax:            8,
	VocabPerLine:        [2]int{1, 3},
	GrammarPoints:       [2]int{1, 2},
	GrammarItemChance:   0.5,
	FootnoteEvery:       3,
	TargetVocab:         4,
	RecallSentences:     3,
	ProduceSegments:     2,
	StudentsPerKind:     1,
	ProgressFraction:    0.5,
	ArchivedFraction:    0.5,
}

// Realistic approximates a full term: enough rows that any per-row query
// pattern shows up as dozens of queries.
var Realistic = Spec{
	Seed:                20261121,
	PaidCourses:         2,
	TrialCourses:        1,
	Stories:             40,
	TrialEvery:          8,
	MultiCourseFraction: 0.25,
	LinesMin:            20,
	LinesMax:            60,
	VocabPerLine:        [2]int{2, 5},
	GrammarPoints:       [2]int{2, 4},
	GrammarItemChance:   0.6,
	FootnoteEvery:       4,
	TargetVocab:         8,
	RecallSentences:     5,
	ProduceSegments:     2,
	StudentsPerKind:     6,
	ProgressFraction:    0.4,
	ArchivedFraction:    0.25,
}

// Kind is a student's paywall situation.
type Kind string

const (
	// KindPaid is enrolled in every paid course with an active entitlement.
	KindPaid Kind = "paid"
	// KindExpired is enrolled in the first paid course; its entitlement expired.
	KindExpired Kind = "expired"
	// KindUnpaid is enrolled in the first paid course with no entitlement.
	KindUnpaid Kind = "unpaid"
	// KindTrial is enrolled only in the trial course(s).
	KindTrial Kind = "trial"
	// KindUnenrolled has an account but no enrollments.
	KindUnenrolled Kind = "unenrolled"
)

// Kinds lists every Kind in a stable order.
var Kinds = []Kind{KindPaid, KindExpired, KindUnpaid, KindTrial, KindUnenrolled}
