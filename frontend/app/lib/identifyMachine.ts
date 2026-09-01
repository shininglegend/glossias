/**
 * State machine for the Identify phase (SUMMER_2026.md, Phase 2).
 *
 * Narration plays line by line. After a line containing a target word ends,
 * playback pauses and a picture quiz opens for each target word on that line
 * in turn. Once every quiz on the line is answered correctly, the line is
 * replayed and playback continues with the next line.
 *
 * Like `translateMachine`, this is a pure reducer so the transition table can
 * be unit-tested without audio. Side effects the component must perform are
 * returned as a `command` with a monotonically increasing `commandSeq`, so
 * the component runs each exactly once from an effect.
 *
 * Progress is server-side: every correct pick is saved as it happens, and the
 * page loads with the picks so far. A reload resumes from the last line that
 * had a correct pick, quizzes already answered are skipped, and a phase the
 * server reports as complete opens in the finished state and cannot be redone.
 *
 * Phases:
 *   idle       – before the student presses Start
 *   playing    – continuous playback
 *   paused     – student paused (only possible while playing)
 *   quiz       – picture quiz open for `targetId` on `line`
 *   replaying  – the quiz line is being replayed before moving on
 *   complete   – the story finished (or was finished on an earlier visit)
 */

export type IdentifyPhase =
  | { kind: "idle" }
  | { kind: "playing" }
  | { kind: "paused" }
  | {
      kind: "quiz";
      /** 0-based line the quiz belongs to. */
      line: number;
      /** Target word being asked about. */
      targetId: number;
      /** Further target words on the same line, quizzed after this one. */
      remaining: number[];
      /** Wrong pictures already clicked for this quiz (disabled in the UI). */
      wrongPicks: number[];
    }
  | { kind: "replaying"; line: number }
  | { kind: "complete" };

export type IdentifyCommand =
  /** Start (or continue) sequential playback at `index`. */
  | { type: "playFrom"; index: number }
  /** Replay a single line, then send REPLAY_DONE. */
  | { type: "replay"; line: number }
  | null;

/** One correctly answered quiz: a target word on a specific line. */
export interface IdentifyPick {
  line: number;
  targetId: number;
}

export interface IdentifyState {
  lineCount: number;
  phase: IdentifyPhase;
  /** 0-based index of the line the audio player is on. */
  currentLine: number;
  /** Server-confirmed correct picks this session. */
  correct: number;
  /** Server-confirmed incorrect picks this session. */
  incorrect: number;
  /** Quizzes answered correctly, this session or on earlier visits. */
  picks: IdentifyPick[];
  command: IdentifyCommand;
  commandSeq: number;
}

export type IdentifyEvent =
  | { type: "START" }
  | { type: "PAUSE" }
  | { type: "RESUME" }
  | { type: "LINE_CHANGED"; index: number }
  /** Audio paused after `line` because it holds target words `targets`. */
  | { type: "LINE_ENDED"; line: number; targets: number[] }
  /** The server graded a picture pick. */
  | { type: "PICK_RESULT"; selected: number; correct: boolean }
  /** The quiz line finished replaying. */
  | { type: "REPLAY_DONE" }
  /** Sequential playback ran off the end of the story. */
  | { type: "STORY_ENDED" };

/** Progress saved on the server from an earlier visit. */
export interface IdentifyResume {
  picks: IdentifyPick[];
  /** The phase was finished on an earlier visit. */
  completed: boolean;
}

export const createIdentifyState = (
  lineCount: number,
  resume?: IdentifyResume,
): IdentifyState => {
  const picks = (resume?.picks ?? []).filter(
    (p) => Number.isInteger(p.line) && p.line >= 0 && p.line < lineCount,
  );
  const completed = resume?.completed ?? false;
  // A reload resumes from the last line with a correct pick, so the student
  // re-hears it in context rather than restarting from the top.
  const lastPickedLine = picks.reduce((max, p) => Math.max(max, p.line), -1);
  return {
    lineCount,
    phase: completed ? { kind: "complete" } : { kind: "idle" },
    currentLine: lastPickedLine >= 0 ? lastPickedLine : 0,
    correct: 0,
    incorrect: 0,
    picks,
    command: null,
    commandSeq: 0,
  };
};

export const isAnswered = (
  picks: readonly IdentifyPick[],
  line: number,
  targetId: number,
): boolean => picks.some((p) => p.line === line && p.targetId === targetId);

/** Distinct target words identified correctly at least once. */
export const identifiedWords = (picks: readonly IdentifyPick[]): number[] => [
  ...new Set(picks.map((p) => p.targetId)),
];

/**
 * Lines that still hold an unanswered target word — the lines playback must
 * pause after. `lineTargets[i]` lists the target words on line `i`.
 */
export const linesWithUnansweredTargets = (
  picks: readonly IdentifyPick[],
  lineTargets: readonly (readonly number[])[],
): Set<number> => {
  const lines = new Set<number>();
  lineTargets.forEach((targets, line) => {
    if (targets.some((t) => !isAnswered(picks, line, t))) lines.add(line);
  });
  return lines;
};

const withCommand = (
  state: IdentifyState,
  command: Exclude<IdentifyCommand, null>,
): IdentifyState => ({
  ...state,
  command,
  commandSeq: state.commandSeq + 1,
});

export function identifyReducer(
  state: IdentifyState,
  event: IdentifyEvent,
): IdentifyState {
  const { phase } = state;

  switch (event.type) {
    case "START":
      if (phase.kind !== "idle") return state;
      return withCommand(
        { ...state, phase: { kind: "playing" } },
        { type: "playFrom", index: state.currentLine },
      );

    case "PAUSE":
      if (phase.kind !== "playing") return state;
      return { ...state, phase: { kind: "paused" } };

    case "RESUME":
      if (phase.kind !== "paused") return state;
      return withCommand(
        { ...state, phase: { kind: "playing" } },
        { type: "playFrom", index: state.currentLine },
      );

    case "LINE_CHANGED":
      // The audio player's index is only meaningful while it is playing; on
      // mount it reports 0, which must not clobber a resumed position.
      if (phase.kind !== "playing" || event.index === state.currentLine) {
        return state;
      }
      return { ...state, currentLine: event.index };

    case "LINE_ENDED": {
      if (phase.kind !== "playing") return state;
      // Quizzes answered on an earlier visit are not asked again.
      const [first, ...rest] = event.targets.filter(
        (t) => !isAnswered(state.picks, event.line, t),
      );
      if (first === undefined) {
        // Nothing to ask about; keep going.
        return withCommand(state, {
          type: "playFrom",
          index: event.line + 1,
        });
      }
      return {
        ...state,
        currentLine: event.line,
        phase: {
          kind: "quiz",
          line: event.line,
          targetId: first,
          remaining: rest,
          wrongPicks: [],
        },
      };
    }

    case "PICK_RESULT": {
      if (phase.kind !== "quiz") return state;
      if (!event.correct) {
        return {
          ...state,
          incorrect: state.incorrect + 1,
          phase: {
            ...phase,
            wrongPicks: phase.wrongPicks.includes(event.selected)
              ? phase.wrongPicks
              : [...phase.wrongPicks, event.selected],
          },
        };
      }
      const picks = isAnswered(state.picks, phase.line, phase.targetId)
        ? state.picks
        : [...state.picks, { line: phase.line, targetId: phase.targetId }];
      const graded = { ...state, correct: state.correct + 1, picks };
      const [next, ...rest] = phase.remaining;
      if (next !== undefined) {
        return {
          ...graded,
          phase: {
            kind: "quiz",
            line: phase.line,
            targetId: next,
            remaining: rest,
            wrongPicks: [],
          },
        };
      }
      return withCommand(
        { ...graded, phase: { kind: "replaying", line: phase.line } },
        { type: "replay", line: phase.line },
      );
    }

    case "REPLAY_DONE": {
      if (phase.kind !== "replaying") return state;
      const next = phase.line + 1;
      if (next >= state.lineCount) {
        return { ...state, phase: { kind: "complete" } };
      }
      return withCommand(
        { ...state, phase: { kind: "playing" } },
        { type: "playFrom", index: next },
      );
    }

    case "STORY_ENDED":
      if (phase.kind !== "playing") return state;
      return { ...state, phase: { kind: "complete" } };
  }
}

/** Fisher–Yates shuffle; returns a new array. */
export function shuffle<T>(items: readonly T[], random = Math.random): T[] {
  const out = [...items];
  for (let i = out.length - 1; i > 0; i--) {
    const j = Math.floor(random() * (i + 1));
    [out[i], out[j]] = [out[j], out[i]];
  }
  return out;
}
