import { describe, it, expect } from "vitest";
import {
  translateReducer,
  createTranslateState,
  canRequest,
  requestBlockReason,
  canFastForward,
  effectiveMinRequests,
  maxRequestable,
  MAX_REQUESTS,
  type TranslateState,
  type TranslateEvent,
} from "./translateMachine";

const run = (state: TranslateState, ...events: TranslateEvent[]) =>
  events.reduce(translateReducer, state);

const started = (lineCount = 10) =>
  run(createTranslateState(lineCount), { type: "START" });

const at = (state: TranslateState, index: number) =>
  run(state, { type: "LINE_CHANGED", index });

/** Request `line` while `playingLine` plays, then run the full reveal cycle. */
const requestCycle = (
  state: TranslateState,
  line: number,
  playingLine = line,
) =>
  run(
    at(state, playingLine),
    { type: "REQUEST", line },
    { type: "LINE_ENDED" },
    { type: "PREDICT_DONE" },
    { type: "REVEAL_DONE" },
  );

describe("translateReducer — start / pause", () => {
  it("starts idle and begins playing from line 0 on START", () => {
    const s0 = createTranslateState(10);
    expect(s0.phase.kind).toBe("idle");
    const s1 = translateReducer(s0, { type: "START" });
    expect(s1.phase.kind).toBe("playing");
    expect(s1.command).toEqual({ type: "playFrom", index: 0 });
    expect(s1.commandSeq).toBe(1);
  });

  it("ignores START once running", () => {
    const s = started();
    expect(translateReducer(s, { type: "START" })).toBe(s);
  });

  it("pauses while playing and resumes at the current line", () => {
    const s = at(started(), 3);
    const paused = translateReducer(s, { type: "PAUSE" });
    expect(paused.phase).toEqual({ kind: "paused" });
    const resumed = translateReducer(paused, { type: "RESUME" });
    expect(resumed.phase.kind).toBe("playing");
    expect(resumed.command).toEqual({ type: "playFrom", index: 3 });
  });

  it("pausing while a request awaits its line end keeps the request", () => {
    const pending = translateReducer(at(started(), 3), {
      type: "REQUEST",
      line: 2,
    });
    const paused = translateReducer(pending, { type: "PAUSE" });
    expect(paused.phase).toEqual({ kind: "paused", pendingRequest: 2 });
    expect(paused.requested).toEqual([2]);
    // Line-end and timer events are ignored while paused.
    expect(translateReducer(paused, { type: "LINE_ENDED" })).toBe(paused);
    expect(requestBlockReason(paused, 3)).toBe("not-playing");

    const resumed = translateReducer(paused, { type: "RESUME" });
    expect(resumed.phase).toEqual({
      kind: "awaitingLineEnd",
      requestedLine: 2,
    });
    expect(resumed.command).toEqual({ type: "playFrom", index: 3 });
    const predicting = translateReducer(resumed, { type: "LINE_ENDED" });
    expect(predicting.phase).toEqual({
      kind: "predicting",
      requestedLine: 2,
      resumeFrom: 4,
    });
  });

  it("cannot pause during the prediction beat or the reveal", () => {
    const pending = translateReducer(at(started(), 3), {
      type: "REQUEST",
      line: 3,
    });
    const predicting = translateReducer(pending, { type: "LINE_ENDED" });
    expect(translateReducer(predicting, { type: "PAUSE" })).toBe(predicting);
    const revealing = translateReducer(predicting, { type: "PREDICT_DONE" });
    expect(translateReducer(revealing, { type: "PAUSE" })).toBe(revealing);
  });

  it("tracks the current line", () => {
    expect(at(started(), 4).currentLine).toBe(4);
  });
});

describe("translateReducer — request eligibility", () => {
  it("accepts the current line and the previous line only", () => {
    const s = at(started(), 5);
    expect(canRequest(s, 5)).toBe(true);
    expect(canRequest(s, 4)).toBe(true);
    expect(requestBlockReason(s, 3)).toBe("not-eligible-line");
    expect(requestBlockReason(s, 6)).toBe("not-eligible-line");
  });

  it("rejects requests unless playing", () => {
    expect(requestBlockReason(createTranslateState(10), 0)).toBe("not-playing");
    const paused = translateReducer(started(), { type: "PAUSE" });
    expect(requestBlockReason(paused, 0)).toBe("not-playing");
  });

  it("ignores clicks while a request is in flight", () => {
    const pending = translateReducer(at(started(), 2), {
      type: "REQUEST",
      line: 2,
    });
    expect(pending.phase.kind).toBe("awaitingLineEnd");
    expect(translateReducer(pending, { type: "REQUEST", line: 1 })).toBe(
      pending,
    );
    const predicting = translateReducer(pending, { type: "LINE_ENDED" });
    expect(translateReducer(predicting, { type: "REQUEST", line: 1 })).toBe(
      predicting,
    );
    const revealing = translateReducer(predicting, { type: "PREDICT_DONE" });
    expect(translateReducer(revealing, { type: "REQUEST", line: 1 })).toBe(
      revealing,
    );
  });

  it("rejects an already-requested line", () => {
    const s = at(requestCycle(started(), 2), 3);
    expect(requestBlockReason(s, 2)).toBe("already-requested");
  });

  it("disables requests at the maximum", () => {
    // Lines 0,1,2 · 4,5,6 · 8 = 7 requests, no run longer than 3.
    let s = started(20);
    for (const line of [0, 1, 2, 4, 5, 6, 8]) s = requestCycle(s, line);
    expect(s.requested).toHaveLength(MAX_REQUESTS);
    expect(requestBlockReason(at(s, 10), 10)).toBe("max-reached");
  });

  it("caps consecutive translated lines at three", () => {
    const s = requestCycle(requestCycle(requestCycle(started(), 0), 1), 2);
    expect(requestBlockReason(at(s, 3), 3)).toBe("consecutive-cap");
    // After line 3 plays untranslated, line 4 is allowed again.
    expect(canRequest(at(s, 4), 4)).toBe(true);
    // Line 3 as the "previous line" is still capped while 4 plays.
    expect(requestBlockReason(at(s, 4), 3)).toBe("consecutive-cap");
  });

  it("applies the consecutive cap to runs joined from either side", () => {
    // Pass 1 translated 5,6,7. On pass 2, requesting 4 would join a run of 4.
    let s = started(10);
    for (const line of [5, 6, 7]) s = requestCycle(s, line);
    s = translateReducer(s, { type: "STORY_ENDED" });
    expect(s.pass).toBe(2);
    expect(requestBlockReason(at(s, 4), 4)).toBe("consecutive-cap");
    expect(canRequest(at(s, 3), 3)).toBe(true);
  });
});

describe("translateReducer — request cycle", () => {
  it("waits for the line to end, predicts, reveals, then resumes after it", () => {
    const s0 = at(started(), 2);
    const s1 = translateReducer(s0, { type: "REQUEST", line: 2 });
    expect(s1.phase).toEqual({ kind: "awaitingLineEnd", requestedLine: 2 });
    expect(s1.requested).toEqual([2]);
    expect(s1.revealed).toEqual([]);

    const s2 = translateReducer(s1, { type: "LINE_ENDED" });
    expect(s2.phase).toEqual({
      kind: "predicting",
      requestedLine: 2,
      resumeFrom: 3,
    });

    const s3 = translateReducer(s2, { type: "PREDICT_DONE" });
    expect(s3.phase.kind).toBe("revealing");
    expect(s3.revealed).toEqual([2]);

    const s4 = translateReducer(s3, { type: "REVEAL_DONE" });
    expect(s4.phase.kind).toBe("playing");
    expect(s4.command).toEqual({ type: "playFrom", index: 3 });
  });

  it("requesting the previous line resumes after the line that was playing", () => {
    const s = requestCycle(started(), 4, 5);
    expect(s.requested).toEqual([4]);
    expect(s.revealed).toEqual([4]);
    expect(s.command).toEqual({ type: "playFrom", index: 6 });
  });

  it("increments commandSeq for every command so repeats are distinguishable", () => {
    const s = requestCycle(requestCycle(started(), 1), 3);
    expect(s.commandSeq).toBe(3); // START + two resumes
  });

  it("ignores out-of-phase timer events", () => {
    const s = started();
    expect(translateReducer(s, { type: "LINE_ENDED" })).toBe(s);
    expect(translateReducer(s, { type: "PREDICT_DONE" })).toBe(s);
    expect(translateReducer(s, { type: "REVEAL_DONE" })).toBe(s);
  });
});

describe("translateReducer — end of story", () => {
  it("completes when the minimum is met", () => {
    let s = started(10);
    for (const line of [1, 3, 5, 7]) s = requestCycle(s, line);
    const done = translateReducer(s, { type: "STORY_ENDED" });
    expect(done.phase.kind).toBe("complete");
    expect(done.command).toBeNull();
  });

  it("restarts from line 0 on a new pass when under the minimum", () => {
    const s = requestCycle(started(10), 2);
    const restarted = translateReducer(s, { type: "STORY_ENDED" });
    expect(restarted.phase.kind).toBe("playing");
    expect(restarted.pass).toBe(2);
    expect(restarted.currentLine).toBe(0);
    expect(restarted.requested).toEqual([2]);
    expect(restarted.command).toEqual({ type: "playFrom", index: 0 });
  });

  it("keeps looping until the minimum is met", () => {
    let s = translateReducer(started(10), { type: "STORY_ENDED" });
    s = translateReducer(s, { type: "STORY_ENDED" });
    expect(s.pass).toBe(3);
    expect(s.phase.kind).toBe("playing");
  });

  it("offers fast-forward only on restart passes while playing", () => {
    const s = started(10);
    expect(canFastForward(s)).toBe(false);
    const restarted = translateReducer(s, { type: "STORY_ENDED" });
    expect(canFastForward(restarted)).toBe(true);
    const pending = translateReducer(at(restarted, 0), {
      type: "REQUEST",
      line: 0,
    });
    expect(canFastForward(pending)).toBe(false);
  });

  it("on a restart pass, ends right after the reveal that meets the minimum", () => {
    let s = started(10);
    for (const line of [1, 3, 5]) s = requestCycle(s, line);
    s = translateReducer(s, { type: "STORY_ENDED" });
    expect(s.pass).toBe(2);
    const done = requestCycle(s, 0);
    expect(done.phase.kind).toBe("complete");
    expect(done.requested).toEqual([1, 3, 5, 0]);
    expect(done.revealed).toEqual([1, 3, 5, 0]);
  });

  it("on the first pass, keeps playing after the minimum is met", () => {
    let s = started(10);
    for (const line of [1, 3, 5, 7]) s = requestCycle(s, line);
    expect(s.phase.kind).toBe("playing");
    expect(s.command).toEqual({ type: "playFrom", index: 8 });
  });

  it("a reveal on the last line applies end-of-story rules instead of resuming", () => {
    // Under minimum: restart.
    const restarted = requestCycle(started(5), 4);
    expect(restarted.pass).toBe(2);
    expect(restarted.command).toEqual({ type: "playFrom", index: 0 });

    // Minimum met: complete.
    let s = started(10);
    for (const line of [1, 3, 5]) s = requestCycle(s, line);
    const done = requestCycle(s, 9);
    expect(done.phase.kind).toBe("complete");
  });

  it("still reveals when STORY_ENDED arrives while awaiting the line end", () => {
    const pending = translateReducer(at(started(5), 4), {
      type: "REQUEST",
      line: 4,
    });
    const s = translateReducer(pending, { type: "STORY_ENDED" });
    expect(s.phase).toEqual({
      kind: "predicting",
      requestedLine: 4,
      resumeFrom: 5,
    });
    const after = run(s, { type: "PREDICT_DONE" }, { type: "REVEAL_DONE" });
    expect(after.pass).toBe(2);
  });

  it("ignores STORY_ENDED once complete", () => {
    let s = started(10);
    for (const line of [1, 3, 5, 7]) s = requestCycle(s, line);
    const done = translateReducer(s, { type: "STORY_ENDED" });
    expect(translateReducer(done, { type: "STORY_ENDED" })).toBe(done);
  });
});

describe("effective minimum for short stories", () => {
  it("leaves every fourth line untranslated when computing capacity", () => {
    expect(maxRequestable(3)).toBe(3);
    expect(maxRequestable(4)).toBe(3);
    expect(maxRequestable(7)).toBe(6);
    expect(maxRequestable(8)).toBe(6);
  });

  it("lowers the minimum to what the story can hold", () => {
    expect(effectiveMinRequests(10)).toBe(4);
    expect(effectiveMinRequests(4)).toBe(3);
    expect(effectiveMinRequests(2)).toBe(2);
    expect(effectiveMinRequests(0)).toBe(0);
  });

  it("lets a 4-line story complete with 3 requests", () => {
    let s = started(4);
    for (const line of [0, 1, 2]) s = requestCycle(s, line);
    expect(requestBlockReason(at(s, 3), 3)).toBe("consecutive-cap");
    const done = translateReducer(s, { type: "STORY_ENDED" });
    expect(done.phase.kind).toBe("complete");
  });
});
