import { useState, useEffect, useRef, useReducer, useCallback } from "react";
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
} from "../lib/produceMachine";

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
        referenceHebrew: submission.reference_hebrew,
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
          referenceHebrew: s.reference_hebrew,
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
  // the student had typed so far.
  const [draft, setDraft] = useState("");
  const draftRef = useRef(draft);
  draftRef.current = draft;
  useEffect(() => {
    if (phase.kind === "idle") setDraft("");
  }, [phase.kind, segmentIndex]);

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
  // page loaded (a reload mid-segment); the draft text was not kept.
  const [resumedCountdown] = useState(
    () =>
      state.phase.kind === "writing" ||
      (state.phase.kind === "submitting" && state.phase.timedOut),
  );
  const currentAttempt = segment ? attempts[segment.id] : undefined;
  const grammarPointNames = pageData.segments
    .map((s) => s.grammar_point_name ?? "")
    .filter(Boolean);

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
              You wrote both segments. This phase is finished and can't be
              repeated — your attempts and the reference sentences are shown
              below.
            </p>
          </div>
        )}

        {resumedMidway && (
          <div
            className="bg-yellow-50 border-l-4 border-yellow-400 p-3 mb-4 rounded-r-lg text-left"
            data-testid="produce-resumed"
          >
            <p className="text-gray-800">
              Welcome back — you already wrote segment {phase.segment} of{" "}
              {totalSegments}. Pick up with segment {phase.segment + 1}.
            </p>
          </div>
        )}

        {resumedCountdown && !isComplete && (
          <div
            className="bg-yellow-50 border-l-4 border-yellow-400 p-3 mb-4 rounded-r-lg text-left"
            data-testid="produce-resumed-countdown"
          >
            <p className="text-gray-800">
              Welcome back — the timer for this segment kept running while you
              were away, so you're picking up with the time that's left.
              Anything you'd typed wasn't saved.
            </p>
          </div>
        )}

        <div className="bg-gray-50 border border-gray-300 p-4 mb-4 rounded-lg text-center">
          <div className="flex items-start justify-center">
            <span className="material-icons text-gray-600 mr-2 mt-1">info</span>
            <div>
              <p className="text-gray-700 mb-2">
                Write each English sentence in Hebrew, in your own words. You
                have {formatCountdown(pageData.time_limit_seconds)} per
                sentence.
              </p>
              <p className="text-gray-700">
                When you submit — or time runs out — the story's version appears
                under yours so you can compare. After both sentences you'll see
                an explanation of the grammar.
              </p>
            </div>
          </div>
        </div>

        {!hasSegments && (
          <div className="bg-yellow-50 border-l-4 border-yellow-400 p-3 mb-4 rounded-r-lg text-left">
            <p className="text-gray-800">
              This story has no writing segments yet, so there is nothing to do
              here. Continue to the next phase.
            </p>
          </div>
        )}

        {hasSegments && !isComplete && (
          <div
            className="text-gray-700 text-base my-3"
            aria-live="polite"
            data-testid="produce-counter"
          >
            Segment <strong>{(segmentIndex ?? 0) + 1}</strong> of{" "}
            {totalSegments}
          </div>
        )}
      </header>

      <div className="max-w-4xl mx-auto px-5 pb-24">
        {segment && (
          <StoryContext
            lines={pageData.lines}
            slot={segment.slot}
            reveal={
              phase.kind === "revealed" ? currentAttempt?.referenceHebrew : null
            }
            isRTL={isRTL}
          />
        )}

        {segment && (
          <section
            className="bg-white shadow-xl rounded-2xl border border-gray-100 p-6 mt-6"
            data-testid={`produce-segment-${segment.segment_order}`}
          >
            <div className="flex flex-wrap items-start justify-between gap-3 mb-4">
              <div>
                <p className="text-xs font-semibold uppercase tracking-wider text-primary-700 mb-1">
                  Write this in Hebrew
                </p>
                <p className="text-xl text-gray-900">{segment.english_text}</p>
                {segment.grammar_point_name && (
                  <p className="text-sm text-gray-500 mt-1">
                    Grammar focus: {segment.grammar_point_name}
                  </p>
                )}
              </div>
              {phase.kind === "writing" && (
                <Countdown seconds={phase.secondsLeft} />
              )}
              {phase.kind === "submitting" && (
                <span className="text-sm text-gray-500">Saving…</span>
              )}
            </div>

            {(phase.kind === "idle" || phase.kind === "starting") && (
              <div className="text-center py-4">
                <p className="text-gray-600 mb-4">
                  The timer starts when you press Start — and keeps running even
                  if you leave the page.
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
                      : "Start next segment"}
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
                  onChange={(e) => setDraft(e.target.value)}
                  disabled={phase.kind === "submitting"}
                  dir={isRTL ? "rtl" : "ltr"}
                  lang={isRTL ? "he" : undefined}
                  rows={3}
                  autoFocus
                  className="text-2xl leading-relaxed py-3"
                  placeholder={isRTL ? "כתבו כאן…" : "Write here…"}
                  aria-label="Your Hebrew translation"
                  data-testid="produce-textarea"
                />
                {submitError && (
                  <p className="mt-2 text-sm text-red-600" role="alert">
                    {submitError}
                  </p>
                )}
                <div className="mt-4 flex justify-end">
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

            {phase.kind === "revealed" && currentAttempt && (
              <>
                {phase.timedOut && (
                  <p
                    className="text-sm text-amber-700 bg-amber-50 border border-amber-200 rounded-md px-3 py-2 mb-4"
                    role="status"
                  >
                    Time's up — here's what you had so far.
                  </p>
                )}
                <AttemptComparison attempt={currentAttempt} isRTL={isRTL} />
                <div className="mt-6 flex justify-end">
                  <Button
                    size="lg"
                    onClick={() => dispatch({ type: "NEXT" })}
                    data-testid="produce-next"
                  >
                    {answered < totalSegments
                      ? "Next segment"
                      : "See the grammar explanation"}
                    <span className="material-icons text-base">
                      arrow_forward
                    </span>
                  </Button>
                </div>
              </>
            )}
          </section>
        )}

        {isComplete && hasSegments && (
          <section className="mt-6 space-y-4" data-testid="produce-review">
            {pageData.segments.map((s) => {
              const a = attempts[s.id];
              return (
                <div
                  key={s.id}
                  className="bg-white shadow rounded-2xl border border-gray-100 p-6"
                >
                  <p className="text-xs font-semibold uppercase tracking-wider text-primary-700 mb-1">
                    Segment {s.segment_order}
                  </p>
                  <p className="text-lg text-gray-900 mb-3">{s.english_text}</p>
                  {a && <AttemptComparison attempt={a} isRTL={isRTL} />}
                </div>
              );
            })}
            <div className="text-center">
              <Button
                variant="outline"
                onClick={() => dispatch({ type: "SHOW_EXPLANATION" })}
                data-testid="produce-show-explanation"
              >
                <span className="material-icons text-base">school</span>
                Show the grammar explanation
              </Button>
            </div>
          </section>
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

function AttemptComparison({
  attempt,
  isRTL,
}: {
  attempt: ProduceAttempt;
  isRTL: boolean;
}) {
  const dir = isRTL ? "rtl" : "ltr";
  const lang = isRTL ? "he" : undefined;
  return (
    <div className="grid gap-3 sm:grid-cols-2">
      <div className="rounded-xl border border-slate-200 bg-slate-50 p-4">
        <p className="text-xs font-semibold uppercase tracking-wider text-slate-500 mb-2">
          Your version
        </p>
        <p
          className="text-2xl leading-relaxed text-gray-900 whitespace-pre-wrap"
          dir={dir}
          lang={lang}
          data-testid="produce-student-text"
        >
          {attempt.studentText || (
            <span className="text-base italic text-gray-400" dir="ltr">
              (nothing written)
            </span>
          )}
        </p>
      </div>
      <div className="rounded-xl border border-green-200 bg-green-50 p-4">
        <p className="text-xs font-semibold uppercase tracking-wider text-green-700 mb-2">
          The story's version
        </p>
        <p
          className="text-2xl leading-relaxed text-gray-900"
          dir={dir}
          lang={lang}
          data-testid="produce-reference"
        >
          {attempt.referenceHebrew}
        </p>
      </div>
    </div>
  );
}

interface StoryContextProps {
  lines: { text: string }[];
  slot: ProduceSlot | undefined;
  /** Once revealed, the reference is shown in the slot instead of a blank. */
  reveal: string | null | undefined;
  isRTL: boolean;
}

/**
 * The surrounding Hebrew story text with the current segment's place marked.
 * An exact slot is a rune range, so the text is split by code point to match
 * the backend, and the reference is blanked out (then shown on reveal). A
 * non-exact slot (the author placed the segment on a line but paraphrased the
 * reference) highlights the whole line instead. With no slot at all the story
 * is shown without a marker.
 */
function StoryContext({ lines, slot, reveal, isRTL }: StoryContextProps) {
  return (
    <div
      className="story-lines text-2xl max-w-3xl mx-auto leading-loose text-gray-800"
      dir={isRTL ? "rtl" : "ltr"}
      lang={isRTL ? "he" : undefined}
      data-testid="produce-context"
    >
      {lines.map((line, i) => {
        if (!slot || slot.line_index !== i) {
          return (
            <span key={i} className="inline">
              {line.text}{" "}
            </span>
          );
        }
        if (!slot.exact) {
          return (
            <span key={i} className="inline">
              <mark
                className={`rounded px-1 ${
                  reveal
                    ? "bg-green-100 text-green-900"
                    : "bg-primary-50 text-gray-900 border-b-2 border-dashed border-primary-500"
                }`}
                aria-label={
                  reveal ? undefined : "your sentence belongs in this line"
                }
                data-testid={
                  reveal ? "produce-slot-revealed" : "produce-slot-line"
                }
              >
                {line.text}
              </mark>{" "}
            </span>
          );
        }
        const runes = Array.from(line.text);
        const before = runes.slice(0, slot.start).join("");
        const after = runes.slice(slot.end).join("");
        return (
          <span key={i} className="inline">
            {before}
            {reveal ? (
              <mark
                className="bg-green-100 text-green-900 rounded px-1"
                data-testid="produce-slot-revealed"
              >
                {reveal}
              </mark>
            ) : (
              <span
                className="inline-block min-w-[8rem] border-b-2 border-dashed border-primary-500 align-baseline mx-1"
                aria-label="your sentence goes here"
                data-testid="produce-slot"
              >
                &nbsp;
              </span>
            )}
            {after}{" "}
          </span>
        );
      })}
    </div>
  );
}

export type { ProduceSegmentView };
