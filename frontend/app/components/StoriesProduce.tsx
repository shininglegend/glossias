import {
  useState,
  useEffect,
  useRef,
  useReducer,
  useCallback,
  type ReactNode,
} from "react";
import { useParams, useNavigate, Link } from "react-router";
import { useApiService } from "../services/api";
import type {
  ProduceData,
  ProduceSegmentView,
  ProduceSlot,
} from "../services/api";
import { useNavigationGuidance } from "../hooks/useNavigationGuidance";
import { CompletionMessage } from "./story-components/CompletionMessage";
import { ProduceExplanationModal } from "./story-components/ProduceExplanationModal";
import Textarea from "./ui/Textarea";
import Button from "./ui/Button";
import {
  produceReducer,
  createProduceState,
  formatCountdown,
  type ProduceAttempt,
  type ProduceEvent,
  type ProducePhase,
} from "../lib/produceMachine";
import {
  loadProduceDraft,
  saveProduceDraft,
  clearProduceDraft,
} from "../lib/produceDraft";

const RTL_LANGUAGES = ["he", "ar", "fa", "ur"];
/** Countdown turns urgent (red) at or below this many seconds. */
const URGENT_SECONDS = 15;

/**
 * Loads the Produce payload and hands it to `ProduceSession`, which owns the
 * writing/reveal state machine (created with the real segments on mount).
 */
export function StoriesProduce() {
  const { id } = useParams<{ id: string }>();
  const api = useApiService();
  const navigate = useNavigate();
  const { getNavigationGuidance } = useNavigationGuidance();

  const [pageData, setPageData] = useState<ProduceData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [nextStepName, setNextStepName] = useState<string>("Next Step");

  useEffect(() => {
    const fetchData = async () => {
      if (!id) {
        setError("Story ID is required");
        setLoading(false);
        return;
      }
      try {
        const response = await api.getStoryProduce(id);
        if (response.success && response.data) {
          setPageData(response.data);
        } else {
          setError(response.error || "Failed to fetch story");
        }
      } catch (err) {
        console.error("Failed to fetch produce data:", err);
        setError("Failed to fetch story");
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, [id, api]);

  useEffect(() => {
    const fetchNextStep = async () => {
      if (!id) return;
      try {
        const guidance = await getNavigationGuidance(id, "produce");
        if (guidance) setNextStepName(guidance.displayName);
      } catch (err) {
        console.error("Failed to get navigation guidance:", err);
      }
    };
    fetchNextStep();
  }, [id, getNavigationGuidance]);

  const start = useCallback(
    async (segmentId: number): Promise<number> => {
      if (!id) throw new Error("Story ID is required");
      const response = await api.startProduce(id, segmentId);
      if (!response.success || !response.data) {
        throw new Error(response.error || "Failed to start");
      }
      return response.data.seconds_left;
    },
    [id, api],
  );

  const submit = useCallback(
    async (segmentId: number, studentText: string): Promise<ProduceAttempt> => {
      if (!id) throw new Error("Story ID is required");
      const response = await api.submitProduce(id, segmentId, studentText);
      if (!response.success || !response.data) {
        throw new Error(response.error || "Failed to submit");
      }
      const { submission } = response.data;
      return {
        segmentId: submission.segment_id,
        studentText: submission.student_text,
        referenceEnglish: submission.reference_english,
      };
    },
    [id, api],
  );

  const handleContinue = async () => {
    if (!id) return;
    try {
      const guidance = await getNavigationGuidance(id, "produce");
      if (guidance) navigate(`/stories/${id}/${guidance.nextPage}`);
    } catch (err) {
      console.error("Failed to navigate to next phase:", err);
    }
  };

  if (loading) {
    return (
      <div className="flex justify-center items-center min-h-[50vh]">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-500"></div>
      </div>
    );
  }

  if (error || !pageData) {
    return (
      <div className="container max-w-xl mx-auto mt-10 p-6 bg-red-50 border border-red-200 rounded-lg text-center">
        <h2 className="text-red-700 font-bold mb-2">Error Loading Phase</h2>
        <p className="text-red-600 mb-4">
          {error || "Could not retrieve story details."}
        </p>
        <Link
          to="/"
          className="text-primary-600 hover:text-primary-700 underline font-medium"
        >
          Back to Stories
        </Link>
      </div>
    );
  }

  return (
    <ProduceSession
      pageData={pageData}
      nextStepName={nextStepName}
      onStart={start}
      onSubmit={submit}
      onContinue={handleContinue}
    />
  );
}

interface ProduceSessionProps {
  pageData: ProduceData;
  nextStepName: string;
  /** Records the attempt's start server-side; resolves with seconds left. */
  onStart: (segmentId: number) => Promise<number>;
  /** Stores an attempt server-side; resolves with the reference revealed. */
  onSubmit: (segmentId: number, studentText: string) => Promise<ProduceAttempt>;
  onContinue: () => void;
}

/**
 * The Produce phase runs Hebrew → English. The whole Hebrew story is shown
 * with the current segment highlighted where it sits, and the answer area —
 * Start button, then the English textarea under a countdown, then the
 * side-by-side reveal — is rendered inline in the story directly beneath that
 * highlighted line, not in a separate card under the text.
 */
export function ProduceSession({
  pageData,
  nextStepName,
  onStart,
  onSubmit,
  onContinue,
}: ProduceSessionProps) {
  const [state, dispatch] = useReducer(produceReducer, pageData, (data) =>
    createProduceState(
      data.segments.map((s) => s.id),
      data.time_limit_seconds,
      {
        attempts: data.submissions.map((s) => ({
          segmentId: s.segment_id,
          studentText: s.student_text,
          referenceEnglish: s.reference_english,
        })),
        starts: (data.starts ?? []).map((s) => ({
          segmentId: s.segment_id,
          secondsLeft: s.seconds_left,
        })),
        completed: data.completed,
      },
    ),
  );
  const { phase, attempts } = state;

  const segmentIndex =
    phase.kind === "idle" ||
    phase.kind === "starting" ||
    phase.kind === "writing" ||
    phase.kind === "submitting" ||
    phase.kind === "revealed"
      ? phase.segment
      : null;
  const segment =
    segmentIndex !== null ? pageData.segments[segmentIndex] : undefined;

  // The textarea is uncontrolled by the machine: its value lives here and is
  // read through a ref when the countdown fires so the timeout submits what
  // the student had typed so far. It is mirrored to localStorage so a reload
  // mid-countdown restores it alongside the server-side clock.
  const storyId = pageData.story_id;
  const [draft, setDraftState] = useState(() =>
    segment ? loadProduceDraft(storyId, segment.id) : "",
  );
  const draftRef = useRef(draft);
  draftRef.current = draft;
  const setDraft = (text: string) => {
    setDraftState(text);
    if (segment) saveProduceDraft(storyId, segment.id, text);
  };
  useEffect(() => {
    if (phase.kind === "idle") setDraftState("");
  }, [phase.kind, segmentIndex]);
  // Once an attempt is stored its draft has served its purpose.
  useEffect(() => {
    for (const id of Object.keys(attempts)) {
      clearProduceDraft(storyId, Number(id));
    }
  }, [attempts, storyId]);

  // Record the start server-side once the machine enters `starting`; the
  // countdown only runs with the server's remaining time, so a reload picks
  // up the same clock.
  const [startError, setStartError] = useState<string | null>(null);
  useEffect(() => {
    if (phase.kind !== "starting" || !segment) return;
    let cancelled = false;
    setStartError(null);
    onStart(segment.id).then(
      (secondsLeft) => {
        if (!cancelled) dispatch({ type: "STARTED", secondsLeft });
      },
      (err) => {
        console.error("Failed to start produce segment:", err);
        if (cancelled) return;
        setStartError("Couldn't start the timer. Please try again.");
        dispatch({ type: "START_FAILED" });
      },
    );
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [phase.kind, segment?.id]);

  // Countdown: one TICK per second while writing.
  useEffect(() => {
    if (phase.kind !== "writing") return;
    const t = setInterval(() => dispatch({ type: "TICK" }), 1000);
    return () => clearInterval(t);
  }, [phase.kind, segmentIndex]);

  // Send the attempt once the machine enters `submitting` (by button or by
  // timeout). Keyed on the phase kind so a retry after SUBMIT_FAILED re-fires.
  const [submitError, setSubmitError] = useState<string | null>(null);
  useEffect(() => {
    if (phase.kind !== "submitting" || !segment) return;
    let cancelled = false;
    setSubmitError(null);
    onSubmit(segment.id, draftRef.current).then(
      (attempt) => {
        if (!cancelled) dispatch({ type: "SUBMITTED", attempt });
      },
      (err) => {
        console.error("Failed to submit produce attempt:", err);
        if (cancelled) return;
        setSubmitError("Couldn't save your answer. Please try again.");
        dispatch({ type: "SUBMIT_FAILED" });
      },
    );
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [phase.kind, segment?.id]);

  const isRTL = RTL_LANGUAGES.includes(pageData.language);
  const isComplete = phase.kind === "complete";
  const hasSegments = pageData.segments.length > 0;
  const totalSegments = pageData.segments.length;
  const answered = Object.keys(attempts).length;
  const resumedMidway =
    phase.kind === "idle" && answered > 0 && phase.segment > 0;
  // The server said this segment's countdown was already running when the
  // page loaded (a reload mid-segment); the draft came back from localStorage.
  const [resumedCountdown] = useState(
    () =>
      state.phase.kind === "writing" ||
      (state.phase.kind === "submitting" && state.phase.timedOut),
  );
  const currentAttempt = segment ? attempts[segment.id] : undefined;
  const grammarPointNames = pageData.segments
    .map((s) => s.grammar_point_name ?? "")
    .filter(Boolean);

  // What goes into the story: while working, only the current segment is
  // highlighted and carries the answer area; once finished, every segment is
  // highlighted with its comparison in place so the review reads in context.
  const entries: StoryEntry[] = isComplete
    ? pageData.segments.map((s) => ({
        key: s.id,
        slot: s.slot,
        revealed: true,
        block: (
          <ReviewCard segment={s} attempt={attempts[s.id]} isRTL={isRTL} />
        ),
      }))
    : segment
      ? [
          {
            key: segment.id,
            slot: segment.slot,
            revealed: phase.kind === "revealed",
            block: (
              <SegmentAnswer
                segment={segment}
                phase={phase}
                attempt={currentAttempt}
                draft={draft}
                onDraftChange={setDraft}
                dispatch={dispatch}
                startError={startError}
                submitError={submitError}
                isLastSegment={answered >= totalSegments}
                isRTL={isRTL}
              />
            ),
          },
        ]
      : [];

  return (
    <>
      <header className="max-w-4xl mx-auto px-5 pt-6 text-center">
        <h1 className="text-3xl font-extrabold text-gray-900 tracking-tight mb-1">
          {pageData.story_title}
        </h1>
        <h2 className="text-lg font-medium text-gray-500 mb-4">Produce</h2>

        {isComplete && (
          <CompletionMessage
            currentStepName="produce"
            nextStepName={nextStepName}
            onContinue={onContinue}
          />
        )}

        {isComplete && hasSegments && (
          <div
            className="bg-green-50 border-l-4 border-green-400 p-3 mb-4 rounded-r-lg text-left"
            data-testid="produce-finished"
          >
            <p className="text-gray-800">
              You translated both passages. This phase is finished and can't be
              repeated — your answers and the story's English are shown in the
              text below.
            </p>
          </div>
        )}

        {resumedMidway && (
          <div
            className="bg-yellow-50 border-l-4 border-yellow-400 p-3 mb-4 rounded-r-lg text-left"
            data-testid="produce-resumed"
          >
            <p className="text-gray-800">
              Welcome back — you already translated passage {phase.segment} of{" "}
              {totalSegments}. Pick up with passage {phase.segment + 1}.
            </p>
          </div>
        )}

        {resumedCountdown && !isComplete && (
          <div
            className="bg-yellow-50 border-l-4 border-yellow-400 p-3 mb-4 rounded-r-lg text-left"
            data-testid="produce-resumed-countdown"
          >
            <p className="text-gray-800">
              Welcome back — the timer for this passage kept running while you
              were away, so you're picking up with the time that's left. What
              you'd typed in this browser has been restored.
            </p>
          </div>
        )}

        <div className="bg-gray-50 border border-gray-300 p-4 mb-4 rounded-lg text-center">
          <div className="flex items-start justify-center">
            <span className="material-icons text-gray-600 mr-2 mt-1">info</span>
            <div>
              <p className="text-gray-700 mb-2">
                A passage of the story is highlighted below. Write what it means
                in English, right where it sits in the text. You have{" "}
                {formatCountdown(pageData.time_limit_seconds)} per passage.
              </p>
              <p className="text-gray-700">
                When you submit — or time runs out — the story's English appears
                beside yours so you can compare. After both passages you'll see
                an explanation of the grammar.
              </p>
            </div>
          </div>
        </div>

        {!hasSegments && (
          <div className="bg-yellow-50 border-l-4 border-yellow-400 p-3 mb-4 rounded-r-lg text-left">
            <p className="text-gray-800">
              This story has no passages to translate yet, so there is nothing
              to do here. Continue to the next phase.
            </p>
          </div>
        )}

        {hasSegments && !isComplete && (
          <div
            className="text-gray-700 text-base my-3"
            aria-live="polite"
            data-testid="produce-counter"
          >
            Passage <strong>{(segmentIndex ?? 0) + 1}</strong> of{" "}
            {totalSegments}
          </div>
        )}
      </header>

      <div className="max-w-4xl mx-auto px-5 pb-24">
        <StoryWithAnswers
          lines={pageData.lines}
          entries={entries}
          isRTL={isRTL}
        />

        {isComplete && hasSegments && (
          <div className="text-center mt-8" data-testid="produce-review">
            <Button
              variant="outline"
              onClick={() => dispatch({ type: "SHOW_EXPLANATION" })}
              data-testid="produce-show-explanation"
            >
              <span className="material-icons text-base">school</span>
              Show the grammar explanation
            </Button>
          </div>
        )}
      </div>

      <ProduceExplanationModal
        isOpen={phase.kind === "explanation"}
        explanation={pageData.explanation}
        grammarPointNames={grammarPointNames}
        onClose={() => dispatch({ type: "CLOSE_EXPLANATION" })}
      />
    </>
  );
}

interface SegmentAnswerProps {
  segment: ProduceSegmentView;
  phase: ProducePhase;
  attempt: ProduceAttempt | undefined;
  draft: string;
  onDraftChange: (text: string) => void;
  dispatch: (event: ProduceEvent) => void;
  startError: string | null;
  submitError: string | null;
  /** After this reveal there is nothing left but the explanation. */
  isLastSegment: boolean;
  isRTL: boolean;
}

/**
 * The answer area for the current segment, rendered inline in the story under
 * the highlighted Hebrew: Start → English textarea with countdown → reveal.
 */
function SegmentAnswer({
  segment,
  phase,
  attempt,
  draft,
  onDraftChange,
  dispatch,
  startError,
  submitError,
  isLastSegment,
  isRTL,
}: SegmentAnswerProps) {
  // When the Hebrew is highlighted verbatim in the line above there is no
  // need to repeat it; when the author placed it on a line range without a
  // verbatim match (or didn't place it at all) the passage is shown here.
  const showHebrew = !segment.slot?.exact;

  return (
    <section
      className="bg-white shadow-xl rounded-2xl border border-primary-100 p-5"
      data-testid={`produce-segment-${segment.segment_order}`}
    >
      <div className="flex flex-wrap items-start justify-between gap-3 mb-3">
        <div>
          <p className="text-xs font-semibold uppercase tracking-wider text-primary-700">
            {showHebrew
              ? "Write this passage in English"
              : "Write the highlighted passage in English"}
          </p>
          {segment.grammar_point_name && (
            <p className="text-sm text-gray-500 mt-1">
              Grammar focus: {segment.grammar_point_name}
            </p>
          )}
        </div>
        {phase.kind === "writing" && <Countdown seconds={phase.secondsLeft} />}
        {phase.kind === "submitting" && (
          <span className="text-sm text-gray-500">Saving…</span>
        )}
      </div>

      {showHebrew && (
        <p
          className="text-2xl leading-relaxed text-gray-900 whitespace-pre-line mb-3"
          dir={isRTL ? "rtl" : "ltr"}
          lang={isRTL ? "he" : undefined}
          data-testid="produce-hebrew"
        >
          {segment.hebrew_text}
        </p>
      )}

      {(phase.kind === "idle" || phase.kind === "starting") && (
        <div className="text-center py-3">
          <p className="text-gray-600 mb-4">
            The timer starts when you press Start — and keeps running even if
            you leave the page.
          </p>
          {startError && (
            <p className="mb-3 text-sm text-red-600" role="alert">
              {startError}
            </p>
          )}
          <button
            type="button"
            onClick={() => dispatch({ type: "START" })}
            disabled={phase.kind === "starting"}
            className={`inline-flex items-center gap-2 px-5 py-3 text-white rounded-lg text-base transition-colors duration-200 ${
              phase.kind === "starting"
                ? "bg-gray-400 cursor-wait"
                : "bg-green-500 hover:bg-green-600 cursor-pointer"
            }`}
            data-testid="produce-start"
          >
            <span className="material-icons">edit</span>
            {phase.kind === "starting"
              ? "Starting…"
              : phase.segment === 0
                ? "Start"
                : "Start next passage"}
          </button>
        </div>
      )}

      {(phase.kind === "writing" || phase.kind === "submitting") && (
        <form
          onSubmit={(e) => {
            e.preventDefault();
            dispatch({ type: "SUBMIT" });
          }}
        >
          <Textarea
            value={draft}
            onChange={(e) => onDraftChange(e.target.value)}
            disabled={phase.kind === "submitting"}
            dir="ltr"
            lang="en"
            rows={2}
            autoFocus
            className="text-xl leading-relaxed py-3"
            placeholder="Write the English here…"
            aria-label="Your English translation"
            data-testid="produce-textarea"
          />
          {submitError && (
            <p className="mt-2 text-sm text-red-600" role="alert">
              {submitError}
            </p>
          )}
          <div className="mt-3 flex justify-end">
            <Button
              type="submit"
              size="lg"
              disabled={phase.kind === "submitting"}
              data-testid="produce-submit"
            >
              {phase.kind === "submitting" ? "Saving…" : "Submit"}
            </Button>
          </div>
        </form>
      )}

      {phase.kind === "revealed" && attempt && (
        <>
          {phase.timedOut && (
            <p
              className="text-sm text-amber-700 bg-amber-50 border border-amber-200 rounded-md px-3 py-2 mb-3"
              role="status"
            >
              Time's up — here's what you had so far.
            </p>
          )}
          <AttemptComparison attempt={attempt} />
          <div className="mt-4 flex justify-end">
            <Button
              size="lg"
              onClick={() => dispatch({ type: "NEXT" })}
              data-testid="produce-next"
            >
              {isLastSegment ? "See the grammar explanation" : "Next passage"}
              <span className="material-icons text-base">arrow_forward</span>
            </Button>
          </div>
        </>
      )}
    </section>
  );
}

/** A finished segment's answer and reference, shown in place in the story. */
function ReviewCard({
  segment,
  attempt,
  isRTL,
}: {
  segment: ProduceSegmentView;
  attempt: ProduceAttempt | undefined;
  isRTL: boolean;
}) {
  return (
    <section
      className="bg-white shadow rounded-2xl border border-gray-100 p-5"
      data-testid={`produce-segment-${segment.segment_order}`}
    >
      <p className="text-xs font-semibold uppercase tracking-wider text-primary-700 mb-1">
        Passage {segment.segment_order}
        {segment.grammar_point_name && ` · ${segment.grammar_point_name}`}
      </p>
      {!segment.slot?.exact && (
        <p
          className="text-2xl leading-relaxed text-gray-900 whitespace-pre-line mb-3"
          dir={isRTL ? "rtl" : "ltr"}
          lang={isRTL ? "he" : undefined}
          data-testid="produce-hebrew"
        >
          {segment.hebrew_text}
        </p>
      )}
      {attempt && <AttemptComparison attempt={attempt} />}
    </section>
  );
}

function Countdown({ seconds }: { seconds: number }) {
  const urgent = seconds <= URGENT_SECONDS;
  return (
    <div
      className={`inline-flex items-center gap-1 rounded-full px-3 py-1 font-mono text-lg tabular-nums ${
        urgent
          ? "bg-red-50 text-red-700 border border-red-200"
          : "bg-slate-50 text-slate-700 border border-slate-200"
      }`}
      role="timer"
      aria-live={urgent ? "assertive" : "off"}
      aria-label={`${seconds} seconds left`}
      data-testid="produce-countdown"
    >
      <span className="material-icons text-base" aria-hidden="true">
        timer
      </span>
      {formatCountdown(seconds)}
    </div>
  );
}

/** The student's English beside the story's English. */
function AttemptComparison({ attempt }: { attempt: ProduceAttempt }) {
  return (
    <div className="grid gap-3 sm:grid-cols-2" dir="ltr" lang="en">
      <div className="rounded-xl border border-slate-200 bg-slate-50 p-4">
        <p className="text-xs font-semibold uppercase tracking-wider text-slate-500 mb-2">
          Your English
        </p>
        <p
          className="text-xl leading-relaxed text-gray-900 whitespace-pre-wrap"
          data-testid="produce-student-text"
        >
          {attempt.studentText || (
            <span className="text-base italic text-gray-400">
              (nothing written)
            </span>
          )}
        </p>
      </div>
      <div className="rounded-xl border border-green-200 bg-green-50 p-4">
        <p className="text-xs font-semibold uppercase tracking-wider text-green-700 mb-2">
          The story's English
        </p>
        <p
          className="text-xl leading-relaxed text-gray-900 whitespace-pre-line"
          data-testid="produce-reference"
        >
          {attempt.referenceEnglish}
        </p>
      </div>
    </div>
  );
}

/** One segment to show in the story: where it is, and what to render there. */
interface StoryEntry {
  key: number;
  slot: ProduceSlot | undefined;
  /** Highlight turns green once the answer has been revealed. */
  revealed: boolean;
  /** The answer area, rendered directly under the segment's last line. */
  block: ReactNode;
}

interface StoryWithAnswersProps {
  lines: { text: string }[];
  entries: StoryEntry[];
  isRTL: boolean;
}

/**
 * The Hebrew story with each entry's passage highlighted and its answer block
 * inserted right after the line the passage ends on. An exact slot highlights
 * the passage's rune range within its line (split by code point to match the
 * backend); a non-exact slot highlights every line in its range. Entries with
 * no slot at all — the author hasn't placed the passage and it isn't in the
 * text verbatim — get their blocks after the story instead.
 */
function StoryWithAnswers({ lines, entries, isRTL }: StoryWithAnswersProps) {
  const unplaced = entries.filter((e) => !e.slot);

  return (
    <div data-testid="produce-context">
      <div
        className="story-lines text-2xl max-w-3xl mx-auto leading-loose text-gray-800 space-y-1"
        dir={isRTL ? "rtl" : "ltr"}
        lang={isRTL ? "he" : undefined}
      >
        {lines.map((line, i) => {
          const covering = entries.filter(
            (e) => e.slot && i >= e.slot.line_index && i <= e.slot.line_end,
          );
          const ending = entries.filter((e) => e.slot?.line_end === i);
          return (
            <div key={i}>
              <StoryLine text={line.text} lineIndex={i} covering={covering} />
              {ending.map((e) => (
                <AnswerSlot key={e.key}>{e.block}</AnswerSlot>
              ))}
            </div>
          );
        })}
      </div>
      {unplaced.map((e) => (
        <AnswerSlot key={e.key}>{e.block}</AnswerSlot>
      ))}
    </div>
  );
}

/**
 * Resets direction and type scale for an answer block sitting inside the
 * RTL, large-type story flow.
 */
function AnswerSlot({ children }: { children: ReactNode }) {
  return (
    <div
      className="my-4 text-base leading-normal text-left"
      dir="ltr"
      data-testid="produce-answer-slot"
    >
      {children}
    </div>
  );
}

function highlightClass(revealed: boolean): string {
  return revealed
    ? "bg-green-100 text-green-900 rounded px-1"
    : "bg-primary-100 text-gray-900 rounded px-1 border-b-2 border-primary-500";
}

/** One story line with any covering passages highlighted. */
function StoryLine({
  text,
  lineIndex,
  covering,
}: {
  text: string;
  lineIndex: number;
  covering: StoryEntry[];
}) {
  if (covering.length === 0) return <>{text}</>;

  // An exact slot pins the passage to a rune range on this very line; a
  // non-exact one (or an exact one whose range is on another line of the
  // segment's span) marks the whole line.
  const exact = covering.find(
    (e) => e.slot?.exact && e.slot.line_index === lineIndex,
  );
  if (!exact?.slot) {
    const revealed = covering.every((e) => e.revealed);
    return (
      <mark
        className={highlightClass(revealed)}
        aria-label={
          revealed ? undefined : "translate this line into English below"
        }
        data-testid={revealed ? "produce-slot-revealed" : "produce-slot-line"}
      >
        {text}
      </mark>
    );
  }

  const runes = Array.from(text);
  const before = runes.slice(0, exact.slot.start).join("");
  const passage = runes.slice(exact.slot.start, exact.slot.end).join("");
  const after = runes.slice(exact.slot.end).join("");
  return (
    <>
      {before}
      <mark
        className={highlightClass(exact.revealed)}
        aria-label={
          exact.revealed
            ? undefined
            : "translate this passage into English below"
        }
        data-testid={exact.revealed ? "produce-slot-revealed" : "produce-slot"}
      >
        {passage}
      </mark>
      {after}
    </>
  );
}

export type { ProduceSegmentView };
