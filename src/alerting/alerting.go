// Package alerting raises operator incidents for conditions that need a
// human, independent of log shipping: a payments pause, an AI grading
// outage. The transport (Better Stack) is behind the Alerter interface so
// business code can be tested with a recording fake and so a new destination
// is one new implementation.
package alerting

import "context"

// Alerter raises and resolves operator incidents. Incidents are keyed: at
// most one incident per key is open at a time, so a long outage that keeps
// re-raising the same key pages once, and Resolve for that key closes it.
//
// Both methods are called from request and background paths, so
// implementations must return quickly (do their network work in the
// background) and never panic.
type Alerter interface {
	Raise(ctx context.Context, incident Incident)
	Resolve(ctx context.Context, key, proof string)
}

// Incident describes one condition for the on-call reader.
type Incident struct {
	// Key identifies the condition across raises and resolves, for example
	// "payments" or "ai_grading". Stable, lower-case, no spaces.
	Key string
	// Title is the incident name as it appears in the pager: short, and
	// specific enough to act on from a phone notification.
	Title string
	// Summary is one sentence on what is broken and what it means for users.
	Summary string
	// Detail is free text for the incident body: the last error, counts,
	// timestamps.
	Detail string
}
