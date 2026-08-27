/**
 * State machine for the Translate phase (SUMMER_2026.md, Phase 3).
 *
 * Audio plays continuously; the student clicks the current or previous line
 * to request its translation. The machine is a pure reducer so the transition
 * table can be unit-tested without audio. Side effects the component must
 * perform (resume audio at a line, restart from the top) are returned as a
 * `command` with a monotonically increasing `commandSeq`, so the component
 * runs each exactly once from an effect.
 *
 * Phases:
 *   idle            – before the student presses Start
 *   playing         – continuous playback, requests accepted
 *   paused          – student paused; nothing pending
 *   awaitingLineEnd – a request was made; audio finishes the current line
 *   predicting      – 2s silent "prediction beat" before the reveal
 *   revealing       – translation shown, 5s hold, then resume
 *   complete        – quota satisfied and the story finished
 */

export const MIN_REQUESTS = 4;
export const MAX_REQUESTS = 7;
export const MAX_CONSECUTIVE = 3;
export const PREDICT_MS = 2000;
export const REVEAL_MS = 5000;

export type TranslatePhase =
  | { kind: "idle" }
  | { kind: "playing" }
  | { kind: "paused" }
  | { kind: "awaitingLineEnd"; requestedLine: number }
  | { kind: "predicting"; requestedLine: number; resumeFrom: number }
  | { kind: "revealing"; requestedLine: number; resumeFrom: number }
  | { kind: "complete" };

export type TranslateCommand =
  /** Start (or continue) sequential playback at `index`. */
  | { type: "playFrom"; index: number }
  /** Nothing to do. */
  | null;

export interface TranslateState {
  lineCount: number;
  phase: TranslatePhase;
  /** 0-based index of the line the audio player is on. */
  currentLine: number;
  /** Lines whose translation has been requested, in request order. */
  requested: number[];
  /** Subset of `requested` whose translation has actually been shown. */
  revealed: number[];
  /** 1 on the first listen; incremented on each end-of-story restart. */
  pass: number;
  command: TranslateCommand;
  commandSeq: number;
}

export type TranslateEvent =
  | { type: "START" }
  | { type: "PAUSE" }
  | { type: "RESUME" }
  | { type: "LINE_CHANGED"; index: number }
  | { type: "REQUEST"; line: number }
  /** Audio paused at the end of a line because a request was pending. */
  | { type: "LINE_ENDED" }
  | { type: "PREDICT_DONE" }
  | { type: "REVEAL_DONE" }
  /** Sequential playback ran off the end of the story. */
  | { type: "STORY_ENDED" };

export const createTranslateState = (lineCount: number): TranslateState => ({
  lineCount,
  phase: { kind: "idle" },
  currentLine: 0,
  requested: [],
  revealed: [],
  pass: 1,
  command: null,
  commandSeq: 0,
});

/**
 * The most requests a story of `lineCount` lines can hold under the
 * consecutive cap: every (MAX_CONSECUTIVE + 1)th line must stay untranslated.
 */
export const maxRequestable = (lineCount: number): number =>
  lineCount - Math.floor(lineCount / (MAX_CONSECUTIVE + 1));

/** The minimum this story can actually demand (short stories lower it). */
export const effectiveMinRequests = (lineCount: number): number =>
  Math.max(0, Math.min(MIN_REQUESTS, maxRequestable(lineCount)));

/** Length of the run of consecutive requested lines that would contain `line`. */
const runLengthWith = (requested: number[], line: number): number => {
  const set = new Set(requested);
  set.add(line);
  let length = 1;
  for (let i = line - 1; set.has(i); i--) length++;
  for (let i = line + 1; set.has(i); i++) length++;
  return length;
};

export type RequestBlockReason =
  | "not-playing"
  | "not-eligible-line"
  | "already-requested"
  | "max-reached"
  | "consecutive-cap";

/** Why `line` cannot be requested right now, or null if it can. */
export const requestBlockReason = (
  state: TranslateState,
  line: number,
): RequestBlockReason | null => {
  if (state.phase.kind !== "playing") return "not-playing";
  if (line !== state.currentLine && line !== state.currentLine - 1) {
    return "not-eligible-line";
  }
  if (line < 0 || line >= state.lineCount) return "not-eligible-line";
  if (state.requested.includes(line)) return "already-requested";
  if (state.requested.length >= MAX_REQUESTS) return "max-reached";
  if (runLengthWith(state.requested, line) > MAX_CONSECUTIVE) {
    return "consecutive-cap";
  }
  return null;
};

export const canRequest = (state: TranslateState, line: number): boolean =>
  requestBlockReason(state, line) === null;

/** Fast-forward is only offered on restart passes while audio is playing. */
export const canFastForward = (state: TranslateState): boolean =>
  state.pass > 1 && state.phase.kind === "playing";

const withCommand = (
  state: TranslateState,
  command: Exclude<TranslateCommand, null>,
): TranslateState => ({
  ...state,
  command,
  commandSeq: state.commandSeq + 1,
});

const minimumMet = (state: TranslateState): boolean =>
  state.requested.length >= effectiveMinRequests(state.lineCount);

/** The story ran out of lines: finish if the minimum is met, else restart. */
const endOfStory = (state: TranslateState): TranslateState => {
  if (minimumMet(state)) {
    return { ...state, phase: { kind: "complete" }, command: null };
  }
  return withCommand(
    {
      ...state,
      phase: { kind: "playing" },
      currentLine: 0,
      pass: state.pass + 1,
    },
    { type: "playFrom", index: 0 },
  );
};

export function translateReducer(
  state: TranslateState,
  event: TranslateEvent,
): TranslateState {
  const { phase } = state;

  switch (event.type) {
    case "START":
      if (phase.kind !== "idle") return state;
      return withCommand(
        { ...state, phase: { kind: "playing" }, currentLine: 0 },
        { type: "playFrom", index: 0 },
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
      if (event.index === state.currentLine) return state;
      return { ...state, currentLine: event.index };

    case "REQUEST":
      if (!canRequest(state, event.line)) return state;
      return {
        ...state,
        phase: { kind: "awaitingLineEnd", requestedLine: event.line },
        requested: [...state.requested, event.line],
      };

    case "LINE_ENDED":
      if (phase.kind !== "awaitingLineEnd") return state;
      return {
        ...state,
        phase: {
          kind: "predicting",
          requestedLine: phase.requestedLine,
          resumeFrom: state.currentLine + 1,
        },
      };

    case "PREDICT_DONE":
      if (phase.kind !== "predicting") return state;
      return {
        ...state,
        phase: { ...phase, kind: "revealing" },
        revealed: [...state.revealed, phase.requestedLine],
      };

    case "REVEAL_DONE": {
      if (phase.kind !== "revealing") return state;
      const resumed: TranslateState = { ...state, phase: { kind: "playing" } };
      // On a restart pass the student has already heard the whole story once,
      // so the phase ends as soon as the minimum is satisfied.
      if (state.pass > 1 && minimumMet(state)) {
        return { ...resumed, phase: { kind: "complete" }, command: null };
      }
      if (phase.resumeFrom >= state.lineCount) return endOfStory(resumed);
      return withCommand(resumed, {
        type: "playFrom",
        index: phase.resumeFrom,
      });
    }

    case "STORY_ENDED":
      if (phase.kind === "playing") return endOfStory(state);
      // The pause request landed after the last line's `ended` fired: still
      // reveal the translation, then apply the end-of-story rules.
      if (phase.kind === "awaitingLineEnd") {
        return {
          ...state,
          phase: {
            kind: "predicting",
            requestedLine: phase.requestedLine,
            resumeFrom: state.lineCount,
          },
        };
      }
      return state;

    default:
      return state;
  }
}
