/**
 * State machine for the Produce phase (SUMMER_2026.md, Phase 4).
 *
 * The student renders two English segments into Hebrew, one at a time, each
 * under a countdown. Submitting — or the timer running out — reveals the
 * reference for self-comparison; a button then advances to the next segment.
 * After the last segment the authored grammar explanation pops up, and the
 * phase is complete.
 *
 * Like `identifyMachine`, this is a pure reducer so the transition table can
 * be unit-tested without timers or network. The component owns the interval
 * and the requests; it reports back with STARTED / TICK / SUBMITTED.
 *
 * Progress is server-side. Pressing Start records the attempt's start time,
 * and every attempt is stored as it is made; the page loads with both. A
 * reload therefore resumes at the first unanswered segment — mid-countdown if
 * it was already started, with the remaining time the server computed — and a
 * phase the server reports as complete opens finished and cannot be redone.
 * The draft text itself is not persisted: a reload mid-segment keeps the
 * clock but clears the textarea.
 *
 * Phases:
 *   idle        – before the student presses Start (or Next segment)
 *   starting    – Start pressed, waiting for the server to record the start
 *   writing     – textarea open, countdown running for `segment`
 *   submitting  – attempt sent, waiting for the reference
 *   revealed    – reference shown under the attempt
 *   explanation – both segments done, explanation popup open
 *   complete    – finished (now or on an earlier visit)
 */

export interface ProduceAttempt {
  segmentId: number;
  studentText: string;
  referenceHebrew: string;
}

/** A segment already started on the server, with the countdown remaining. */
export interface ProduceStart {
  segmentId: number;
  secondsLeft: number;
}

export type ProducePhase =
  | { kind: "idle"; segment: number }
  | { kind: "starting"; segment: number }
  | { kind: "writing"; segment: number; secondsLeft: number }
  | { kind: "submitting"; segment: number; timedOut: boolean }
  | { kind: "revealed"; segment: number; timedOut: boolean }
  | { kind: "explanation" }
  | { kind: "complete" };

export interface ProduceState {
  /** Segment IDs in presentation order. */
  segmentIds: number[];
  timeLimit: number;
  phase: ProducePhase;
  /** Stored attempts keyed by segment ID. */
  attempts: Record<number, ProduceAttempt>;
  /** Whether the explanation popup has been dismissed this session. */
  explanationSeen: boolean;
}

export type ProduceEvent =
  /** Student pressed Start on the current segment (from idle). */
  | { type: "START" }
  /** The server recorded (or already had) the start; countdown to run. */
  | { type: "STARTED"; secondsLeft: number }
  /** Recording the start failed; back to idle so the student can retry. */
  | { type: "START_FAILED" }
  /** One second elapsed on the countdown. */
  | { type: "TICK" }
  /** Student pressed Submit. */
  | { type: "SUBMIT" }
  /** The server stored the attempt and returned the reference. */
  | { type: "SUBMITTED"; attempt: ProduceAttempt }
  /** The request failed; go back to writing so the student can retry. */
  | { type: "SUBMIT_FAILED" }
  /** Advance from the reveal to the next segment / the explanation. */
  | { type: "NEXT" }
  /** Dismiss the explanation popup. */
  | { type: "CLOSE_EXPLANATION" }
  /** Re-open the explanation after finishing. */
  | { type: "SHOW_EXPLANATION" };

export interface ProduceResume {
  attempts: ProduceAttempt[];
  /** Segments started but not submitted, per the server. */
  starts?: ProduceStart[];
  completed: boolean;
}

/**
 * Builds the initial state. With no segments there is nothing to do, so the
 * phase opens complete. Otherwise it opens at the first segment that has no
 * attempt yet — idle, or straight into the countdown if the server says that
 * segment was already started (out of time → submit at once) — or complete
 * when every segment has an attempt.
 */
export function createProduceState(
  segmentIds: number[],
  timeLimit: number,
  resume?: ProduceResume,
): ProduceState {
  const attempts: Record<number, ProduceAttempt> = {};
  for (const a of resume?.attempts ?? []) {
    if (segmentIds.includes(a.segmentId)) attempts[a.segmentId] = a;
  }
  const next = firstUnanswered(segmentIds, attempts);
  const completed =
    segmentIds.length === 0 || resume?.completed === true || next === -1;

  let phase: ProducePhase;
  if (completed) {
    phase = { kind: "complete" };
  } else {
    const start = resume?.starts?.find((s) => s.segmentId === segmentIds[next]);
    phase = start
      ? countdownPhase(next, start.secondsLeft)
      : { kind: "idle", segment: next };
  }

  return {
    segmentIds,
    timeLimit,
    attempts,
    // A finished phase never shows the popup again on load — the student can
    // re-open it with the button.
    explanationSeen: completed,
    phase,
  };
}

/** Writing with time left, or submitting as timed out when there is none. */
function countdownPhase(segment: number, secondsLeft: number): ProducePhase {
  return secondsLeft > 0
    ? { kind: "writing", segment, secondsLeft }
    : { kind: "submitting", segment, timedOut: true };
}

function firstUnanswered(
  segmentIds: number[],
  attempts: Record<number, ProduceAttempt>,
): number {
  return segmentIds.findIndex((id) => !(id in attempts));
}

export function produceReducer(
  state: ProduceState,
  event: ProduceEvent,
): ProduceState {
  const { phase } = state;

  switch (event.type) {
    case "START": {
      if (phase.kind !== "idle") return state;
      return { ...state, phase: { kind: "starting", segment: phase.segment } };
    }

    case "STARTED": {
      if (phase.kind !== "starting") return state;
      return {
        ...state,
        phase: countdownPhase(
          phase.segment,
          Math.min(event.secondsLeft, state.timeLimit),
        ),
      };
    }

    case "START_FAILED": {
      if (phase.kind !== "starting") return state;
      return { ...state, phase: { kind: "idle", segment: phase.segment } };
    }

    case "TICK": {
      if (phase.kind !== "writing") return state;
      // Time's up at zero: the component submits whatever is in the textarea.
      return {
        ...state,
        phase: countdownPhase(phase.segment, phase.secondsLeft - 1),
      };
    }

    case "SUBMIT": {
      if (phase.kind !== "writing") return state;
      return {
        ...state,
        phase: { kind: "submitting", segment: phase.segment, timedOut: false },
      };
    }

    case "SUBMITTED": {
      if (phase.kind !== "submitting") return state;
      if (state.segmentIds[phase.segment] !== event.attempt.segmentId) {
        return state;
      }
      return {
        ...state,
        attempts: {
          ...state.attempts,
          [event.attempt.segmentId]: event.attempt,
        },
        phase: {
          kind: "revealed",
          segment: phase.segment,
          timedOut: phase.timedOut,
        },
      };
    }

    case "SUBMIT_FAILED": {
      if (phase.kind !== "submitting") return state;
      // Give a little time back so a failed request cannot strand the student
      // on a zero-second timer.
      return {
        ...state,
        phase: {
          kind: "writing",
          segment: phase.segment,
          secondsLeft: phase.timedOut ? RETRY_GRACE_SECONDS : state.timeLimit,
        },
      };
    }

    case "NEXT": {
      if (phase.kind !== "revealed") return state;
      const next = firstUnanswered(state.segmentIds, state.attempts);
      if (next !== -1) {
        return { ...state, phase: { kind: "idle", segment: next } };
      }
      return { ...state, phase: { kind: "explanation" } };
    }

    case "CLOSE_EXPLANATION": {
      if (phase.kind !== "explanation") return state;
      return { ...state, explanationSeen: true, phase: { kind: "complete" } };
    }

    case "SHOW_EXPLANATION": {
      if (phase.kind !== "complete") return state;
      return { ...state, phase: { kind: "explanation" } };
    }

    default:
      return state;
  }
}

/** Seconds handed back after a failed submit on a timed-out segment. */
export const RETRY_GRACE_SECONDS = 15;

/** Formats seconds as m:ss for the countdown. */
export function formatCountdown(seconds: number): string {
  const s = Math.max(0, seconds);
  const m = Math.floor(s / 60);
  return `${m}:${String(s % 60).padStart(2, "0")}`;
}
