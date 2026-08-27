import { describe, it, expect } from "vitest";
import {
  identifyReducer,
  createIdentifyState,
  identifiedWords,
  linesWithUnansweredTargets,
  shuffle,
  type IdentifyState,
  type IdentifyEvent,
  type IdentifyResume,
} from "./identifyMachine";

const run = (state: IdentifyState, ...events: IdentifyEvent[]) =>
  events.reduce(identifyReducer, state);

const started = (lineCount = 5, resume?: IdentifyResume) =>
  run(createIdentifyState(lineCount, resume), { type: "START" });

describe("identifyReducer — resuming an earlier visit", () => {
  it("opens finished and inert when the server says complete", () => {
    const s = createIdentifyState(5, {
      picks: [{ line: 1, targetId: 10 }],
      completed: true,
    });
    expect(s.phase.kind).toBe("complete");
    expect(run(s, { type: "START" })).toBe(s);
    expect(run(s, { type: "LINE_ENDED", line: 1, targets: [10] })).toBe(s);
    expect(s.commandSeq).toBe(0);
  });

  it("starts from the last line with a correct pick", () => {
    const s = started(5, {
      picks: [
        { line: 3, targetId: 11 },
        { line: 1, targetId: 10 },
      ],
      completed: false,
    });
    expect(s.currentLine).toBe(3);
    expect(s.command).toEqual({ type: "playFrom", index: 3 });
  });

  it("ignores the audio player's line while idle, so a resume point survives mount", () => {
    const idle = createIdentifyState(5, {
      picks: [{ line: 3, targetId: 11 }],
      completed: false,
    });
    const afterMount = run(idle, { type: "LINE_CHANGED", index: 0 });
    expect(afterMount).toBe(idle);
    expect(run(afterMount, { type: "START" }).command).toEqual({
      type: "playFrom",
      index: 3,
    });
  });

  it("drops out-of-range picks and starts from 0 when none remain", () => {
    const s = started(5, {
      picks: [{ line: 9, targetId: 10 }],
      completed: false,
    });
    expect(s.picks).toEqual([]);
    expect(s.command).toEqual({ type: "playFrom", index: 0 });
  });

  it("does not re-ask quizzes already answered, but asks the rest", () => {
    const resumed = started(5, {
      picks: [{ line: 1, targetId: 10 }],
      completed: false,
    });
    // Fully answered line: playback continues straight on.
    const skipped = run(resumed, {
      type: "LINE_ENDED",
      line: 1,
      targets: [10],
    });
    expect(skipped.phase.kind).toBe("playing");
    expect(skipped.command).toEqual({ type: "playFrom", index: 2 });

    // Partly answered line: only the unanswered word is quizzed.
    const partial = run(resumed, {
      type: "LINE_ENDED",
      line: 1,
      targets: [10, 11],
    });
    expect(partial.phase).toEqual({
      kind: "quiz",
      line: 1,
      targetId: 11,
      remaining: [],
      wrongPicks: [],
    });
  });

  it("keeps the same word on another line as a separate quiz", () => {
    const s = run(
      started(5, { picks: [{ line: 1, targetId: 10 }], completed: false }),
      { type: "LINE_ENDED", line: 3, targets: [10] },
    );
    expect(s.phase.kind).toBe("quiz");
  });
});

describe("identify helpers", () => {
  it("identifiedWords lists distinct words across lines", () => {
    expect(
      identifiedWords([
        { line: 1, targetId: 10 },
        { line: 3, targetId: 10 },
        { line: 2, targetId: 11 },
      ]),
    ).toEqual([10, 11]);
  });

  it("linesWithUnansweredTargets excludes fully answered lines", () => {
    const lineTargets = [[], [10], [10, 11], [], [12]];
    const picks = [
      { line: 1, targetId: 10 },
      { line: 2, targetId: 10 },
    ];
    expect([...linesWithUnansweredTargets(picks, lineTargets)]).toEqual([2, 4]);
    expect([...linesWithUnansweredTargets([], lineTargets)]).toEqual([1, 2, 4]);
  });
});

describe("identifyReducer — start / pause / resume", () => {
  it("starts playback from line 0", () => {
    const s = started();
    expect(s.phase.kind).toBe("playing");
    expect(s.command).toEqual({ type: "playFrom", index: 0 });
    expect(s.commandSeq).toBe(1);
  });

  it("ignores START when not idle", () => {
    const s = started();
    expect(run(s, { type: "START" })).toBe(s);
  });

  it("pauses while playing and resumes at the current line", () => {
    const s = run(
      started(),
      { type: "LINE_CHANGED", index: 2 },
      { type: "PAUSE" },
    );
    expect(s.phase.kind).toBe("paused");
    expect(s.commandSeq).toBe(1);

    const r = run(s, { type: "RESUME" });
    expect(r.phase.kind).toBe("playing");
    expect(r.command).toEqual({ type: "playFrom", index: 2 });
    expect(r.commandSeq).toBe(2);
  });

  it("cannot pause during a quiz or replay", () => {
    const quiz = run(started(), {
      type: "LINE_ENDED",
      line: 1,
      targets: [10],
    });
    expect(run(quiz, { type: "PAUSE" })).toBe(quiz);

    const replaying = run(quiz, {
      type: "PICK_RESULT",
      selected: 10,
      correct: true,
    });
    expect(replaying.phase.kind).toBe("replaying");
    expect(run(replaying, { type: "PAUSE" })).toBe(replaying);
  });
});

describe("identifyReducer — quiz flow", () => {
  it("opens a quiz for the first target on the ended line", () => {
    const s = run(started(), {
      type: "LINE_ENDED",
      line: 1,
      targets: [10, 11],
    });
    expect(s.phase).toEqual({
      kind: "quiz",
      line: 1,
      targetId: 10,
      remaining: [11],
      wrongPicks: [],
    });
    expect(s.currentLine).toBe(1);
    // No audio command: the hook already paused.
    expect(s.commandSeq).toBe(1);
  });

  it("continues playback when the ended line has no targets", () => {
    const s = run(started(), { type: "LINE_ENDED", line: 1, targets: [] });
    expect(s.phase.kind).toBe("playing");
    expect(s.command).toEqual({ type: "playFrom", index: 2 });
  });

  it("ignores LINE_ENDED unless playing", () => {
    const paused = run(started(), { type: "PAUSE" });
    expect(run(paused, { type: "LINE_ENDED", line: 1, targets: [10] })).toBe(
      paused,
    );
  });

  it("records wrong picks, disables them, and keeps the quiz open", () => {
    const quiz = run(started(), {
      type: "LINE_ENDED",
      line: 1,
      targets: [10],
    });
    const s = run(
      quiz,
      { type: "PICK_RESULT", selected: 12, correct: false },
      { type: "PICK_RESULT", selected: 12, correct: false },
      { type: "PICK_RESULT", selected: 13, correct: false },
    );
    expect(s.phase.kind).toBe("quiz");
    expect(s.incorrect).toBe(3);
    expect(s.correct).toBe(0);
    if (s.phase.kind === "quiz") {
      expect(s.phase.wrongPicks).toEqual([12, 13]);
    }
    expect(s.picks).toEqual([]);
  });

  it("moves to the next target on the same line after a correct pick", () => {
    const s = run(
      started(),
      { type: "LINE_ENDED", line: 1, targets: [10, 11] },
      { type: "PICK_RESULT", selected: 12, correct: false },
      { type: "PICK_RESULT", selected: 10, correct: true },
    );
    expect(s.phase).toEqual({
      kind: "quiz",
      line: 1,
      targetId: 11,
      remaining: [],
      wrongPicks: [],
    });
    expect(s.correct).toBe(1);
    expect(s.incorrect).toBe(1);
    expect(s.picks).toEqual([{ line: 1, targetId: 10 }]);
    expect(s.commandSeq).toBe(1);
  });

  it("replays the line once every target on it is identified", () => {
    const s = run(
      started(),
      { type: "LINE_ENDED", line: 1, targets: [10, 11] },
      { type: "PICK_RESULT", selected: 10, correct: true },
      { type: "PICK_RESULT", selected: 11, correct: true },
    );
    expect(s.phase).toEqual({ kind: "replaying", line: 1 });
    expect(s.command).toEqual({ type: "replay", line: 1 });
    expect(s.commandSeq).toBe(2);
    expect(s.picks).toEqual([
      { line: 1, targetId: 10 },
      { line: 1, targetId: 11 },
    ]);
  });

  it("ignores PICK_RESULT outside a quiz", () => {
    const s = started();
    expect(run(s, { type: "PICK_RESULT", selected: 10, correct: true })).toBe(
      s,
    );
  });
});

describe("identifyReducer — replay and completion", () => {
  const replayingAt = (line: number, lineCount = 5) =>
    run(
      started(lineCount),
      { type: "LINE_ENDED", line, targets: [10] },
      { type: "PICK_RESULT", selected: 10, correct: true },
    );

  it("resumes playback at the following line after the replay", () => {
    const s = run(replayingAt(1), { type: "REPLAY_DONE" });
    expect(s.phase.kind).toBe("playing");
    expect(s.command).toEqual({ type: "playFrom", index: 2 });
  });

  it("completes directly when the replayed line was the last one", () => {
    const s = run(replayingAt(4), { type: "REPLAY_DONE" });
    expect(s.phase.kind).toBe("complete");
    expect(s.command).toEqual({ type: "replay", line: 4 }); // unchanged
    expect(s.commandSeq).toBe(2);
  });

  it("ignores REPLAY_DONE unless replaying", () => {
    const s = started();
    expect(run(s, { type: "REPLAY_DONE" })).toBe(s);
  });

  it("completes when playback runs off the end", () => {
    const s = run(started(), { type: "STORY_ENDED" });
    expect(s.phase.kind).toBe("complete");
  });

  it("ignores STORY_ENDED during a quiz", () => {
    const quiz = run(started(), {
      type: "LINE_ENDED",
      line: 4,
      targets: [10],
    });
    expect(run(quiz, { type: "STORY_ENDED" })).toBe(quiz);
  });

  it("does nothing after completion", () => {
    const done = run(started(), { type: "STORY_ENDED" });
    expect(run(done, { type: "START" })).toBe(done);
    expect(run(done, { type: "RESUME" })).toBe(done);
    expect(run(done, { type: "LINE_ENDED", line: 0, targets: [10] })).toBe(
      done,
    );
  });
});

describe("shuffle", () => {
  it("returns a permutation without mutating the input", () => {
    const input = [1, 2, 3, 4, 5];
    const out = shuffle(input, () => 0.42);
    expect(out).toHaveLength(5);
    expect([...out].sort()).toEqual([1, 2, 3, 4, 5]);
    expect(input).toEqual([1, 2, 3, 4, 5]);
  });

  it("is deterministic for a fixed random source", () => {
    let i = 0;
    const seq = [0.1, 0.9, 0.5, 0.3];
    const rnd = () => seq[i++ % seq.length];
    const a = shuffle([1, 2, 3, 4, 5], rnd);
    i = 0;
    const b = shuffle([1, 2, 3, 4, 5], rnd);
    expect(a).toEqual(b);
  });
});
