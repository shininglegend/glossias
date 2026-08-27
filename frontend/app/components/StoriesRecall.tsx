import { useState, useEffect, useCallback, useMemo } from "react";
import { useParams, useNavigate, Link } from "react-router";
import {
  DndContext,
  KeyboardSensor,
  PointerSensor,
  closestCenter,
  useSensor,
  useSensors,
  type DragEndEvent,
} from "@dnd-kit/core";
import {
  SortableContext,
  arrayMove,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { useApiService } from "../services/api";
import type {
  CheckRecallResult,
  RecallCard,
  RecallData,
  VocabLine,
} from "../services/api";
import { useNavigationGuidance } from "../hooks/useNavigationGuidance";
import { useAudioPlayer } from "./story-components/AudioPlayer";
import { CompletionMessage } from "./story-components/CompletionMessage";

const RTL_LANGUAGES = ["he", "ar", "fa", "ur"];
const EMPTY_SET = new Set<number>();

/**
 * Loads the Recall payload and hands it to `RecallSession`, which owns the
 * listen → arrange → check flow.
 */
export function StoriesRecall() {
  const { id } = useParams<{ id: string }>();
  const api = useApiService();
  const navigate = useNavigate();
  const { getNavigationGuidance } = useNavigationGuidance();

  const [pageData, setPageData] = useState<RecallData | null>(null);
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
        const response = await api.getStoryRecall(id);
        if (response.success && response.data) {
          setPageData(response.data);
        } else {
          setError(response.error || "Failed to fetch story");
        }
      } catch (err) {
        console.error("Failed to fetch recall data:", err);
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
        const guidance = await getNavigationGuidance(id, "recall");
        if (guidance) setNextStepName(guidance.displayName);
      } catch (err) {
        console.error("Failed to get navigation guidance:", err);
      }
    };
    fetchNextStep();
  }, [id, getNavigationGuidance]);

  const checkOrder = useCallback(
    async (orderedIds: number[]) => {
      if (!id) throw new Error("Story ID is required");
      const response = await api.checkRecall(id, orderedIds);
      if (!response.success || !response.data) {
        throw new Error(response.error || "Failed to check order");
      }
      return response.data;
    },
    [id, api],
  );

  const handleContinue = async () => {
    if (!id) return;
    try {
      const guidance = await getNavigationGuidance(id, "recall");
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
    <RecallSession
      pageData={pageData}
      nextStepName={nextStepName}
      onCheckOrder={checkOrder}
      onContinue={handleContinue}
    />
  );
}

/**
 * Phases of one visit:
 *   idle       – before the student presses Start
 *   listening  – the story plays audio-only; no seeking, no skipping
 *   paused     – student paused (only possible while listening)
 *   arranging  – playback finished; the cards can be ordered and submitted
 *   complete   – every card is in the right place (now or on an earlier visit)
 *
 * A story with no narration skips straight to arranging, and a story with no
 * recall sentences finishes right after the narration.
 */
type RecallPhase = "idle" | "listening" | "paused" | "arranging" | "complete";

interface RecallSessionProps {
  pageData: RecallData;
  nextStepName: string;
  /** Grades an ordering server-side (sentence IDs in submitted order). */
  onCheckOrder: (orderedIds: number[]) => Promise<CheckRecallResult>;
  onContinue: () => void;
}

export function RecallSession({
  pageData,
  nextStepName,
  onCheckOrder,
  onContinue,
}: RecallSessionProps) {
  const hasNarration = Object.keys(pageData.audio_urls).length > 0;
  const hasSentences = pageData.sentences.length > 0;

  const [phase, setPhase] = useState<RecallPhase>(() => {
    if (pageData.completed) return "complete";
    if (!hasNarration) return hasSentences ? "arranging" : "complete";
    return "idle";
  });
  const [order, setOrder] = useState<number[]>(() =>
    pageData.sentences.map((s) => s.id),
  );
  const [lastResults, setLastResults] = useState<boolean[] | null>(null);
  const [attempts, setAttempts] = useState(pageData.attempts);
  const [checking, setChecking] = useState(false);
  const [checkError, setCheckError] = useState<string | null>(null);

  const cardsById = useMemo(() => {
    const map = new Map<number, RecallCard>();
    for (const s of pageData.sentences) map.set(s.id, s);
    return map;
  }, [pageData.sentences]);

  // ---- Audio-only narration ------------------------------------------------

  // The audio hook speaks VocabData; the Recall phase shows no text, so hand
  // it empty lines just to establish the line count.
  const [vocabLines] = useState<VocabLine[]>(() =>
    Array.from({ length: pageData.line_count }, () => ({
      text: [],
      audio_files: [],
      signed_audio_urls: {},
    })),
  );
  const audioURLs = useMemo<Record<string, string>>(
    () => ({ ...pageData.audio_urls }),
    [pageData.audio_urls],
  );
  const [audioLineIndex, setAudioLineIndex] = useState(0);
  const [isPlaying, setIsPlaying] = useState(false);
  const [playedLines, setPlayedLines] = useState<Set<number>>(new Set());

  // How many lines the student has heard so far, persisted per story in this
  // browser so a reload can pick up where they left off rather than
  // restarting a two-minute narration. Cleared once the story finishes.
  const progressKey = listeningProgressKey(pageData.story_id);
  const [furthestHeard, setFurthestHeard] = useState(() =>
    readListeningProgress(progressKey, pageData.line_count),
  );
  const resumeLine = furthestHeard > 0 && furthestHeard < pageData.line_count;
  useEffect(() => {
    if (playedLines.size === 0) return;
    const heard = Math.max(...playedLines) + 1;
    setFurthestHeard((current) => {
      const next = Math.max(current, heard);
      // Nothing to resume once every line is heard; onPlaybackEnd clears the
      // key, and this effect can run after it for the final line.
      if (next !== current && next < pageData.line_count) {
        writeListeningProgress(progressKey, next);
      }
      return next;
    });
  }, [playedLines, progressKey, pageData.line_count]);

  const onPlaybackEnd = useCallback(() => {
    clearListeningProgress(progressKey);
    setPhase((current) => {
      if (current !== "listening") return current;
      return hasSentences ? "arranging" : "complete";
    });
  }, [hasSentences, progressKey]);

  const audioPlayer = useAudioPlayer({
    audioURLs,
    pageData: {
      story_id: pageData.story_id,
      story_title: pageData.story_title,
      language: pageData.language,
      lines: vocabLines,
      vocab_bank: [],
    },
    onPlayedLinesChange: setPlayedLines,
    onCurrentLineChange: setAudioLineIndex,
    onPlayingStateChange: setIsPlaying,
    completedLines: EMPTY_SET,
    // Never pause between lines: the student listens straight through.
    pauseOnLines: EMPTY_SET,
    onPlaybackEnd,
  });

  const handlePlayPause = () => {
    if (phase === "idle" && resumeLine) {
      // Continue from the furthest line heard on an earlier visit. The
      // continuation API starts at index + 1.
      setPhase("listening");
      audioPlayer.playNextLineFromIndex(furthestHeard - 1);
    } else if (phase === "idle" || phase === "paused") {
      setPhase("listening");
      audioPlayer.playStoryAudio();
    } else if (phase === "listening") {
      audioPlayer.pauseAudio();
      setPhase("paused");
    }
  };

  /** Replay the previous line (or the current one, on line 1) and carry on. */
  const handleBackOneLine = () => {
    if (phase !== "listening" && phase !== "paused") return;
    const target = Math.max(audioLineIndex - 1, 0);
    setPhase("listening");
    audioPlayer.playNextLineFromIndex(target - 1);
  };

  // ---- Ordering -------------------------------------------------------------

  const sensors = useSensors(
    useSensor(PointerSensor),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    }),
  );

  const moveCard = (from: number, to: number) => {
    if (to < 0 || to >= order.length || from === to) return;
    setOrder((current) => arrayMove(current, from, to));
    // The result markers describe the previous arrangement; drop them once
    // the student changes it.
    setLastResults(null);
  };

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    if (!over || active.id === over.id) return;
    const from = order.indexOf(Number(active.id));
    const to = order.indexOf(Number(over.id));
    moveCard(from, to);
  };

  const handleSubmit = async () => {
    if (phase !== "arranging" || checking) return;
    setChecking(true);
    setCheckError(null);
    try {
      const result = await onCheckOrder(order);
      setAttempts((n) => n + 1);
      setLastResults(result.results);
      if (result.all_correct) setPhase("complete");
    } catch (err) {
      console.error("Failed to check recall order:", err);
      setCheckError("Couldn't check your order. Please try again.");
    } finally {
      setChecking(false);
    }
  };

  // ---- Render ---------------------------------------------------------------

  const isRTL = RTL_LANGUAGES.includes(pageData.language);
  const lineCount = pageData.line_count;
  // The bar shows where playback *is* (lines before the current one), so
  // stepping back moves it back. Before Start it shows the saved resume point.
  const linesBehind = phase === "idle" ? furthestHeard : audioLineIndex;
  const progressPercent =
    lineCount > 0 ? Math.round((linesBehind / lineCount) * 100) : 0;
  const isListeningPhase =
    phase === "idle" || phase === "listening" || phase === "paused";
  const correctCount = lastResults?.filter(Boolean).length ?? 0;

  const playButtonLabel =
    phase === "idle"
      ? "Start"
      : phase === "listening"
        ? "Pause Audio"
        : "Resume Audio";

  return (
    <div className="max-w-4xl mx-auto px-5 py-8">
      <header className="mb-8 text-center">
        <span className="inline-block px-3 py-1 bg-primary-50 text-primary-700 rounded-full text-xs font-semibold uppercase tracking-wider mb-3">
          Phase 5 of 5
        </span>
        <h1 className="text-3xl font-extrabold text-gray-900 tracking-tight sm:text-4xl mb-2">
          {pageData.story_title}
        </h1>
        <h2 className="text-lg font-medium text-gray-500">Recall Phase</h2>
      </header>

      {isListeningPhase && (
        <section
          className="bg-white shadow-xl rounded-2xl border border-gray-100 max-w-2xl mx-auto p-8"
          data-testid="recall-listening"
        >
          <div className="flex items-center justify-center w-16 h-16 bg-orange-50 text-orange-600 rounded-2xl mx-auto mb-6">
            <span className="material-icons text-3xl">headphones</span>
          </div>
          <div className="text-center mb-6">
            <h3 className="text-xl font-bold text-gray-900 mb-3">
              Listen to the whole story
            </h3>
            <p className="text-gray-600 leading-relaxed max-w-md mx-auto">
              Play the story audio here. When it ends, you'll put five key
              sentences back into story order.
            </p>
          </div>

          {phase === "idle" && resumeLine && (
            <div
              className="bg-yellow-50 border-l-4 border-yellow-400 p-3 mb-4 rounded-r-lg text-left"
              data-testid="recall-resume"
            >
              <p className="text-gray-800">
                Welcome back — you'll pick up from line {furthestHeard + 1},
                where you left off.
              </p>
            </div>
          )}

          <div className="mb-6">
            <div
              className="flex justify-between text-sm text-gray-600 mb-1"
              aria-live="polite"
              data-testid="recall-progress"
            >
              <span>
                {phase === "idle"
                  ? "Ready to listen"
                  : `Line ${Math.min(audioLineIndex + 1, lineCount)} of ${lineCount}`}
              </span>
              <span>{progressPercent}%</span>
            </div>
            <div
              className="h-2 w-full bg-gray-200 rounded-full overflow-hidden"
              role="progressbar"
              aria-valuemin={0}
              aria-valuemax={100}
              aria-valuenow={progressPercent}
              aria-label="Story playback progress"
            >
              <div
                className="h-full bg-orange-500 transition-all duration-300"
                style={{ width: `${progressPercent}%` }}
              />
            </div>
          </div>

          <div className="flex flex-wrap justify-center gap-3">
            {phase !== "idle" && (
              <button
                onClick={handleBackOneLine}
                className="inline-flex items-center gap-2 px-4 py-3 bg-gray-100 text-gray-800 border border-gray-300 rounded-lg text-base transition-colors duration-200 cursor-pointer hover:bg-gray-200"
                type="button"
              >
                <span className="material-icons">replay</span>
                Back 1 line
              </button>
            )}
            <button
              onClick={handlePlayPause}
              className={`inline-flex items-center gap-2 px-5 py-3 text-white border-none rounded-lg text-base transition-colors duration-200 cursor-pointer ${
                phase === "listening"
                  ? "bg-red-500 hover:bg-red-600"
                  : "bg-green-500 hover:bg-green-600"
              }`}
              type="button"
            >
              <span className="material-icons">
                {phase === "listening" && isPlaying ? "pause" : "play_arrow"}
              </span>
              {playButtonLabel}
            </button>
          </div>
        </section>
      )}

      {!isListeningPhase && !hasSentences && (
        <div
          className="bg-yellow-50 border-l-4 border-yellow-400 p-3 mb-4 rounded-r-lg text-left max-w-2xl mx-auto"
          data-testid="recall-no-sentences"
        >
          <p className="text-gray-800">
            This story has no recall sentences yet, so there is nothing to put
            in order. Continue to the next step.
          </p>
        </div>
      )}

      {!isListeningPhase && hasSentences && (
        <section className="max-w-2xl mx-auto" data-testid="recall-arranging">
          <div className="bg-gray-50 border border-gray-300 p-4 mb-4 rounded-lg text-center">
            <div className="flex items-start justify-center">
              <span className="material-icons text-gray-600 mr-2 mt-1">
                info
              </span>
              <p className="text-gray-700">
                {phase === "complete"
                  ? "Every sentence is in its place. This is the story's order."
                  : "Drag the sentences into the order they happened in the story, first at the top, then check your answer."}
              </p>
            </div>
          </div>

          {phase === "complete" && pageData.completed && (
            <div
              className="bg-green-50 border-l-4 border-green-500 p-3 mb-4 rounded-r-lg text-left"
              data-testid="recall-already-complete"
            >
              <p className="text-gray-800">
                You already completed this phase on an earlier visit.
              </p>
            </div>
          )}

          {lastResults && phase !== "complete" && (
            <div
              className="bg-yellow-50 border-l-4 border-yellow-400 p-3 mb-4 rounded-r-lg text-left"
              role="status"
              data-testid="recall-feedback"
            >
              <p className="text-gray-800">
                {correctCount} of {lastResults.length} in the right place. Move
                the highlighted cards and check again.
              </p>
            </div>
          )}

          <DndContext
            sensors={sensors}
            collisionDetection={closestCenter}
            onDragEnd={handleDragEnd}
          >
            <SortableContext
              items={order}
              strategy={verticalListSortingStrategy}
              disabled={phase === "complete"}
            >
              <ol className="flex flex-col gap-3" data-testid="recall-cards">
                {order.map((id, index) => {
                  const card = cardsById.get(id);
                  if (!card) return null;
                  return (
                    <SortableRecallCard
                      key={id}
                      card={card}
                      position={index + 1}
                      total={order.length}
                      result={
                        phase === "complete" ? true : lastResults?.[index]
                      }
                      locked={phase === "complete"}
                      isRTL={isRTL}
                      onMoveUp={() => moveCard(index, index - 1)}
                      onMoveDown={() => moveCard(index, index + 1)}
                    />
                  );
                })}
              </ol>
            </SortableContext>
          </DndContext>

          {phase === "arranging" && (
            <div className="flex flex-wrap items-center justify-center gap-4 mt-6">
              <button
                onClick={handleSubmit}
                disabled={checking}
                className={`inline-flex items-center gap-2 px-6 py-3 text-white rounded-lg text-base font-semibold transition-colors duration-200 ${
                  checking
                    ? "bg-gray-400 cursor-not-allowed"
                    : "bg-primary-600 hover:bg-primary-700 cursor-pointer"
                }`}
                type="button"
              >
                <span className="material-icons">check</span>
                {checking ? "Checking…" : "Check Order"}
              </button>
              {attempts > 0 && (
                <span
                  className="text-gray-600 text-sm"
                  data-testid="recall-attempts"
                >
                  Attempts: <strong>{attempts}</strong>
                </span>
              )}
            </div>
          )}

          {checkError && (
            <p className="text-red-600 text-center mt-3" role="alert">
              {checkError}
            </p>
          )}

          {phase === "complete" && (
            <CompletionMessage
              currentStepName="recall"
              nextStepName={nextStepName}
              onContinue={onContinue}
            />
          )}
        </section>
      )}

      {!isListeningPhase && !hasSentences && (
        <CompletionMessage
          currentStepName="recall"
          nextStepName={nextStepName}
          onContinue={onContinue}
        />
      )}
    </div>
  );
}

// ---- Listening progress (per browser) --------------------------------------
//
// Only the listening position is kept client-side; answers are server-side.
// localStorage can be missing or throw (private mode, blocked storage), so
// every access is guarded and falls back to "start from the beginning".

const listeningProgressKey = (storyId: string) => `recall-listened:${storyId}`;

function readListeningProgress(key: string, lineCount: number): number {
  try {
    const raw = window.localStorage.getItem(key);
    const heard = raw === null ? 0 : Number.parseInt(raw, 10);
    if (!Number.isInteger(heard) || heard < 0) return 0;
    return Math.min(heard, lineCount);
  } catch {
    return 0;
  }
}

function writeListeningProgress(key: string, heard: number) {
  try {
    window.localStorage.setItem(key, String(heard));
  } catch {
    // Best effort only.
  }
}

function clearListeningProgress(key: string) {
  try {
    window.localStorage.removeItem(key);
  } catch {
    // Best effort only.
  }
}

interface SortableRecallCardProps {
  card: RecallCard;
  position: number;
  total: number;
  /** Result of the last check for this position; undefined before a check. */
  result: boolean | undefined;
  locked: boolean;
  isRTL: boolean;
  onMoveUp: () => void;
  onMoveDown: () => void;
}

function SortableRecallCard({
  card,
  position,
  total,
  result,
  locked,
  isRTL,
  onMoveUp,
  onMoveDown,
}: SortableRecallCardProps) {
  const {
    attributes,
    listeners,
    setNodeRef,
    setActivatorNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: card.id, disabled: locked });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
  };

  const tone =
    result === true
      ? "border-green-500 bg-green-50"
      : result === false
        ? "border-red-400 bg-red-50"
        : "border-gray-200 bg-white";

  return (
    <li
      ref={setNodeRef}
      style={style}
      className={`flex items-stretch gap-3 rounded-xl border-2 p-3 shadow-sm ${tone} ${
        isDragging ? "opacity-70 shadow-lg" : ""
      }`}
      data-testid={`recall-card-${card.id}`}
      data-result={
        result === undefined ? "pending" : result ? "correct" : "wrong"
      }
    >
      <div className="flex flex-col items-center justify-center w-8 text-gray-500">
        <span className="font-bold text-lg" aria-label={`Position ${position}`}>
          {position}
        </span>
        {result === true && (
          <span className="material-icons text-green-600 text-lg">check</span>
        )}
        {result === false && (
          <span className="material-icons text-red-500 text-lg">close</span>
        )}
      </div>

      {card.image_url && (
        <img
          src={card.image_url}
          alt=""
          className="w-20 h-20 object-cover rounded-lg flex-shrink-0"
          draggable={false}
        />
      )}

      <p
        className="flex-1 self-center text-2xl text-gray-900"
        dir={isRTL ? "rtl" : "ltr"}
      >
        {card.hebrew_text}
      </p>

      {!locked && (
        <div className="flex flex-col items-center justify-between">
          <button
            type="button"
            onClick={onMoveUp}
            disabled={position === 1}
            className="text-gray-500 hover:text-gray-800 disabled:opacity-30 disabled:cursor-not-allowed"
            aria-label={`Move sentence ${position} up`}
          >
            <span className="material-icons">expand_less</span>
          </button>
          <button
            type="button"
            ref={setActivatorNodeRef}
            {...attributes}
            {...listeners}
            className="text-gray-400 hover:text-gray-700 cursor-grab active:cursor-grabbing touch-none"
            aria-label={`Drag sentence ${position}`}
          >
            <span className="material-icons">drag_indicator</span>
          </button>
          <button
            type="button"
            onClick={onMoveDown}
            disabled={position === total}
            className="text-gray-500 hover:text-gray-800 disabled:opacity-30 disabled:cursor-not-allowed"
            aria-label={`Move sentence ${position} down`}
          >
            <span className="material-icons">expand_more</span>
          </button>
        </div>
      )}
    </li>
  );
}
