package models

import (
	"context"
	"strings"
	"sync"
	"testing"

	"glossias/src/alerting"
)

// recordingAlerter captures raises and resolves for assertions.
type recordingAlerter struct {
	mu       sync.Mutex
	raised   []alerting.Incident
	resolved []string // keys
	proofs   []string
}

func (a *recordingAlerter) Raise(_ context.Context, inc alerting.Incident) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.raised = append(a.raised, inc)
}

func (a *recordingAlerter) Resolve(_ context.Context, key, proof string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.resolved = append(a.resolved, key)
	a.proofs = append(a.proofs, proof)
}

// raisedFor returns the raised incidents with the given key.
func (a *recordingAlerter) raisedFor(key string) []alerting.Incident {
	a.mu.Lock()
	defer a.mu.Unlock()
	var out []alerting.Incident
	for _, inc := range a.raised {
		if inc.Key == key {
			out = append(out, inc)
		}
	}
	return out
}

// resolvedFor counts resolves of the given key.
func (a *recordingAlerter) resolvedFor(key string) int {
	a.mu.Lock()
	defer a.mu.Unlock()
	n := 0
	for _, k := range a.resolved {
		if k == key {
			n++
		}
	}
	return n
}

// installAlerter installs a recording alerter for the test and resets the
// payments pause state before and after.
func installAlerter(t *testing.T) *recordingAlerter {
	t.Helper()
	resetPaymentsStateForTest()
	a := &recordingAlerter{}
	SetAlerter(a)
	t.Cleanup(func() {
		SetAlerter(nil)
		resetPaymentsStateForTest()
	})
	return a
}

func TestFailureStreak(t *testing.T) {
	var s failureStreak
	if s.length() != 0 || s.reset() {
		t.Fatal("zero value must be empty and unreported")
	}
	if s.fail() != 1 || s.fail() != 2 || s.length() != 2 {
		t.Fatal("fail should count consecutively")
	}
	if !s.report() {
		t.Fatal("first report of a streak returns true")
	}
	if s.report() {
		t.Fatal("second report of the same streak returns false")
	}
	if !s.reset() {
		t.Fatal("reset after a report says so")
	}
	if s.length() != 0 || s.report() != true {
		t.Fatal("reset must clear both the count and the reported flag")
	}
	s.reset()
	s.fail()
	if s.reset() {
		t.Fatal("reset of an unreported streak returns false")
	}
}

func TestSetAlerter_NilMeansLogsOnly(t *testing.T) {
	installAlerter(t)
	SetAlerter(nil)
	if currentAlerter() != nil {
		t.Fatal("nil should uninstall")
	}
	// Transitions must not panic without an alerter.
	PausePayments(context.Background(), PauseReasonManual, "no alerter")
	ResumePayments(context.Background(), "no alerter")
}

// containsAll is a small helper for incident wording assertions.
func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !strings.Contains(s, p) {
			return false
		}
	}
	return true
}
