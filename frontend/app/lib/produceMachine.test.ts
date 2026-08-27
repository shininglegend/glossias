import { describe, it, expect } from "vitest";
import {
  produceReducer,
  createProduceState,
  formatCountdown,
  RETRY_GRACE_SECONDS,
  type ProduceState,
  type ProduceEvent,
} from "./produceMachine";

const run = (state: ProduceState, ...events: ProduceEvent[]) =>
  events.reduce(produceReducer, state);

const attempt = (segmentId: number, studentText = "x") => ({
  segmentId,
  studentText,
  referenceHebrew: `ref-${segmentId}`,
});

describe("createProduceState", () => {
  it("opens idle at the first segment with no attempts", () => {
    const s = createProduceState([10, 20], 90);
    expect(s.phase).toEqual({ kind: "idle", segment: 0 });
    expect(s.explanationSeen).toBe(false);
  });

  it("resumes at the first unanswered segment", () => {
    const s = createProduceState([10, 20], 90, {
      attempts: [attempt(10)],
      completed: false,
    });
    expect(s.phase).toEqual({ kind: "idle", segment: 1 });
    expect(s.attempts[10]).toEqual(attempt(10));
  });

  it("opens complete when every segment has an attempt", () => {
    const s = createProduceState([10, 20], 90, {
      attempts: [attempt(20), attempt(10)],
      completed: true,
    });
    expect(s.phase.kind).toBe("complete");
    expect(s.explanationSeen).toBe(true);
    expect(run(s, { type: "START" })).toBe(s);
  });

  it("opens complete with nothing to do when there are no segments", () => {
    expect(createProduceState([], 90).phase.kind).toBe("complete");
  });

  it("ignores attempts for segments that no longer exist", () => {
    const s = createProduceState([10], 90, {
      attempts: [attempt(99)],
      completed: false,
    });
    expect(s.phase).toEqual({ kind: "idle", segment: 0 });
    expect(Object.keys(s.attempts)).toHaveLength(0);
  });
});

describe("produceReducer — one segment end to end", () => {
  const start = () => run(createProduceState([10, 20], 3), { type: "START" });

  it("starts the countdown at the time limit", () => {
    expect(start().phase).toEqual({
      kind: "writing",
      segment: 0,
      secondsLeft: 3,
    });
  });

  it("counts down and submits as timed out at zero", () => {
    const s = run(start(), { type: "TICK" }, { type: "TICK" });
    expect(s.phase).toEqual({ kind: "writing", segment: 0, secondsLeft: 1 });
    const t = run(s, { type: "TICK" });
    expect(t.phase).toEqual({ kind: "submitting", segment: 0, timedOut: true });
    // Further ticks are ignored while submitting.
    expect(run(t, { type: "TICK" })).toBe(t);
  });

  it("submits early on SUBMIT", () => {
    const s = run(start(), { type: "TICK" }, { type: "SUBMIT" });
    expect(s.phase).toEqual({
      kind: "submitting",
      segment: 0,
      timedOut: false,
    });
  });

  it("reveals the reference once the server confirms", () => {
    const s = run(
      start(),
      { type: "SUBMIT" },
      { type: "SUBMITTED", attempt: attempt(10, "שלום") },
    );
    expect(s.phase).toEqual({ kind: "revealed", segment: 0, timedOut: false });
    expect(s.attempts[10]).toEqual(attempt(10, "שלום"));
  });

  it("ignores a SUBMITTED for a different segment", () => {
    const s = run(start(), { type: "SUBMIT" });
    expect(run(s, { type: "SUBMITTED", attempt: attempt(20) })).toBe(s);
  });

  it("returns to writing with a grace period when a timed-out submit fails", () => {
    const s = run(
      start(),
      { type: "TICK" },
      { type: "TICK" },
      { type: "TICK" },
      { type: "SUBMIT_FAILED" },
    );
    expect(s.phase).toEqual({
      kind: "writing",
      segment: 0,
      secondsLeft: RETRY_GRACE_SECONDS,
    });
  });

  it("returns to writing with the full limit when an early submit fails", () => {
    const s = run(start(), { type: "SUBMIT" }, { type: "SUBMIT_FAILED" });
    expect(s.phase).toEqual({ kind: "writing", segment: 0, secondsLeft: 3 });
  });

  it("NEXT after the first reveal moves to the second segment, idle", () => {
    const s = run(
      start(),
      { type: "SUBMIT" },
      { type: "SUBMITTED", attempt: attempt(10) },
      { type: "NEXT" },
    );
    expect(s.phase).toEqual({ kind: "idle", segment: 1 });
  });
});

describe("produceReducer — finishing", () => {
  const afterSecondReveal = () =>
    run(
      createProduceState([10, 20], 90, {
        attempts: [attempt(10)],
        completed: false,
      }),
      { type: "START" },
      { type: "SUBMIT" },
      { type: "SUBMITTED", attempt: attempt(20) },
    );

  it("NEXT after the last reveal opens the explanation", () => {
    const s = run(afterSecondReveal(), { type: "NEXT" });
    expect(s.phase).toEqual({ kind: "explanation" });
    expect(s.explanationSeen).toBe(false);
  });

  it("closing the explanation completes the phase", () => {
    const s = run(
      afterSecondReveal(),
      { type: "NEXT" },
      { type: "CLOSE_EXPLANATION" },
    );
    expect(s.phase).toEqual({ kind: "complete" });
    expect(s.explanationSeen).toBe(true);
  });

  it("the explanation can be re-opened once complete", () => {
    const s = run(
      afterSecondReveal(),
      { type: "NEXT" },
      { type: "CLOSE_EXPLANATION" },
      { type: "SHOW_EXPLANATION" },
    );
    expect(s.phase).toEqual({ kind: "explanation" });
  });

  it("a complete phase ignores writing events", () => {
    const s = run(
      afterSecondReveal(),
      { type: "NEXT" },
      { type: "CLOSE_EXPLANATION" },
    );
    expect(run(s, { type: "START" })).toBe(s);
    expect(run(s, { type: "TICK" })).toBe(s);
    expect(run(s, { type: "SUBMIT" })).toBe(s);
    expect(run(s, { type: "NEXT" })).toBe(s);
  });
});

describe("formatCountdown", () => {
  it("formats m:ss", () => {
    expect(formatCountdown(90)).toBe("1:30");
    expect(formatCountdown(5)).toBe("0:05");
    expect(formatCountdown(0)).toBe("0:00");
    expect(formatCountdown(-3)).toBe("0:00");
  });
});
