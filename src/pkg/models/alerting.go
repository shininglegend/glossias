package models

import (
	"sync"

	"glossias/src/alerting"
)

// Operator alerting.
//
// One process-wide alerting.Alerter (Better Stack in production, a recording
// fake in tests) receives every incident the models raise: a payments pause
// (payments_state.go) and an AI grading outage (produce_grading.go). It is
// installed once from main and read at the moment an incident is raised, so
// construction order does not matter and a nil alerter simply means
// "logs only".

var alerterState struct {
	mu sync.Mutex
	a  alerting.Alerter
}

// SetAlerter installs the alerter that receives incidents. nil disables
// alerting; transitions are still logged (and, for payments, persisted).
func SetAlerter(a alerting.Alerter) {
	alerterState.mu.Lock()
	defer alerterState.mu.Unlock()
	alerterState.a = a
}

// currentAlerter returns the installed alerter, or nil.
func currentAlerter() alerting.Alerter {
	alerterState.mu.Lock()
	defer alerterState.mu.Unlock()
	return alerterState.a
}

// failureStreak counts consecutive failures of one kind and remembers whether
// the current streak has been reported. It is the shape of every "N in a row"
// trigger: one rejected request from a scanner, one dropped connection or one
// odd input must not raise an alarm, but a misconfiguration or an outage
// repeats, and a success in between means it was noise.
//
// Callers that keep their own notion of "reported" (the payments pause, which
// is persisted and reason-aware) use only fail and reset's side effect; callers
// with nothing else to hang the once-ness on use report and reset's result.
// The zero value is ready to use and safe for concurrent use.
type failureStreak struct {
	mu       sync.Mutex
	n        int
	reported bool
}

// fail records one failure and returns the streak length including it.
func (s *failureStreak) fail() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.n++
	return s.n
}

// report marks the current streak as reported. It returns false if it already
// was, so a caller raising an alert on true alerts once per outage.
func (s *failureStreak) report() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.reported {
		return false
	}
	s.reported = true
	return true
}

// reset clears the streak and returns whether it had been reported, which is
// the caller's cue to resolve the alert it raised.
func (s *failureStreak) reset() (wasReported bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	wasReported = s.reported
	s.n = 0
	s.reported = false
	return wasReported
}

// length returns the current streak, for tests and status views.
func (s *failureStreak) length() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.n
}
