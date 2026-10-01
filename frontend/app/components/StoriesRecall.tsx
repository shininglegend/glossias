import { useState, useEffect, useCallback, useMemo, useRef } from "react";
import { useParams, useNavigate } from "react-router";
import { StoryShell } from "./story-components/StoryShell";
import { useApiService } from "../services/api";
import type {
  CheckRecallPickResult,
  RecallCard,
  RecallData,
  VocabLine,
} from "../services/api";
import { useNavigationGuidance } from "../hooks/useNavigationGuidance";
import { useAudioPlayer } from "./story-components/AudioPlayer";
import { CompletionMessage } from "./story-components/CompletionMessage";

const RTL_LANGUAGES = ["he", "ar", "fa", "ur"];
const EMPTY_SET = new Set<number>();
const ORDINALS = ["first", "second", "third", "fourth", "fifth"];

/**
 * Loads the Recall payload and hands it to `RecallSession`, which owns the
 * listen → select-in-order → check flow.
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
  const [navError, setNavError] = useState<string | null>(null);

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

  const checkPick = useCallback(
    async (sentenceId: number, position: number) => {
      if (!id) throw new Error("Story ID is required");
      const response = await api.checkRecallPick(id, sentenceId, position);
      if (!response.success || !response.data) {
        throw new Error(response.error || "Failed to check pick");
      }
      return response.data;
    },
    [id, api],
  );

  const handleContinue = async () => {
    if (!id) return;
    try {
      const guidance = await getNavigationGuidance(id, "recall");
      if (guidance) {
        navigate(`/stories/${id}/${guidance.nextPage}`);
        return;
      }
      setNavError("Couldn't open the next phase.");
    } catch (err) {
      console.error("Failed to navigate to next phase:", err);
      setNavError("Couldn't open the next phase.");
    }
  };

  return (
    <StoryShell
      phasePath="recall"
      storyTitle={pageData?.story_title}
      loading={loading}
      error={
        error ||
        (!loading && !pageData ? "Could not retrieve story details." : null)
      }
      navError={navError}
    >
      {pageData ? (
        <RecallSession
          pageData={pageData}
          nextStepName={nextStepName}
          onCheckPick={checkPick}
          onContinue={handleContinue}
        />
      ) : null}
    </StoryShell>
  );
}

/**
 * Phases of one visit:
 *   idle       – before the student presses Start
 *   listening  – the story plays audio-only; no seeking, no skipping
 *   paused     – student paused (only possible while listening)
 *   selecting  – playback finished; pick sentences in story order
 *   complete   – every sentence has been picked in order (now or earlier)
 *
 * A story with no narration skips straight to selecting, and a story with no
 * recall sentences finishes right after the narration.
 */
type RecallPhase = "idle" | "listening" | "paused" | "selecting" | "complete";

interface RecallSessionProps {
  pageData: RecallData;
  nextStepName: string;
  onCheckPick: (
    sentenceId: number,
    position: number,
  ) => Promise<CheckRecallPickResult>;
  onContinue: () => void;
}

export function RecallSession({
  pageData,
  nextStepName,
  onCheckPick,
  onContinue,
}: RecallSessionProps) {
  const hasNarration = Object.keys(pageData.audio_urls).length > 0;
  const hasSentences = pageData.sentences.length > 0;

  // How many lines the student has heard so far, persisted per story in this
  // browser so a reload can pick up where they left off rather than
  // restarting a two-minute narration. Reaching the last line is kept too,
  // so a reload after the narration goes straight to the cards. Cleared once
  // the phase is complete.
  const progressKey = listeningProgressKey(pageData.story_id);
  const [furthestHeard, setFurthestHeard] = useState(() =>
    readListeningProgress(progressKey, pageData.line_count),
  );

  // Sentences already placed, in story order. A finished phase uses the
  // server's order when it sent one; otherwise every card counts as placed.
  const [placedIds, setPlacedIds] = useState<number[]>(() => {
    const placed = pageData.placed_sentence_ids ?? [];
    if (pageData.completed && placed.length !== pageData.sentences.length) {
      return pageData.sentences.map((s) => s.id);
    }
    return placed;
  });

  const [nextPosition, setNextPosition] = useState(placedIds.length + 1);
  const [correctIds, setCorrectIds] = useState<Set<number>>(
    () => new Set(placedIds),
  );
  const [wrongIds, setWrongIds] = useState<Set<number>>(new Set());
  const [attempts, setAttempts] = useState(pageData.attempts);

  // Any recorded pick means the narration was already heard; so does a saved
  // listening position at the last line. Once true the student may move
  // freely between the audio and the cards.
  const heardAll =
    !hasNarration ||
    attempts > 0 ||
    correctIds.size > 0 ||
    (pageData.line_count > 0 && furthestHeard >= pageData.line_count);

  const [phase, setPhase] = useState<RecallPhase>(() => {
    if (pageData.completed) return "complete";
    if (heardAll) return hasSentences ? "selecting" : "complete";
    return "idle";
  });
  const [checking, setChecking] = useState(false);
  const [checkError, setCheckError] = useState<string | null>(null);
  const sentenceAudioRef = useRef<HTMLAudioElement | null>(null);
  // Sentence clips are bare Audio elements, so stop them when the page goes.
  useEffect(() => () => sentenceAudioRef.current?.pause(), []);

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

  const resumeLine = furthestHeard > 0 && furthestHeard < pageData.line_count;
  useEffect(() => {
    if (playedLines.size === 0) return;
    const heard = Math.max(...playedLines) + 1;
    setFurthestHeard((current) => {
      const next = Math.max(current, heard);
      if (next !== current) writeListeningProgress(progressKey, next);
      return next;
    });
  }, [playedLines, progressKey]);

  const onPlaybackEnd = useCallback(() => {
    // The played-lines effect may run after this for the final line; record
    // the full narration here so a reload before it never replays the story.
    writeListeningProgress(progressKey, pageData.line_count);
    setFurthestHeard(pageData.line_count);
    setPhase((current) => {
      if (current !== "listening") return current;
      return hasSentences ? "selecting" : "complete";
    });
  }, [hasSentences, progressKey, pageData.line_count]);

  // Once the phase is done there is nothing left to resume.
  useEffect(() => {
    if (phase === "complete") clearListeningProgress(progressKey);
  }, [phase, progressKey]);

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

  /** Listen to the story again; placed cards are kept. */
  const backToAudio = () => {
    setWrongIds(new Set());
    setPhase("idle");
  };

  /** Skip the rest of a narration already heard in full and go to the cards. */
  const skipToOrdering = () => {
    audioPlayer.pauseAudio();
    setPhase("selecting");
  };

  // Free movement within the part already heard: back to line 1, forward no
  // further than the first line not yet heard (index `furthestHeard`).
  const canStepBack =
    (phase === "listening" || phase === "paused") && audioLineIndex > 0;
  const canStepForward =
    (phase === "listening" || phase === "paused") &&
    audioLineIndex < furthestHeard;

  /** Jump `delta` lines from the current one and carry on playing from there. */
  const stepLine = (delta: number) => {
    const target = audioLineIndex + delta;
    if (target < 0 || target > furthestHeard) return;
    setPhase("listening");
    // The continuation API starts at index + 1.
    audioPlayer.playNextLineFromIndex(target - 1);
  };

  const playSentenceAudio = (urls?: string[]) => {
    sentenceAudioRef.current?.pause();
    if (!urls?.length) return;
    let index = 0;
    const playNext = () => {
      if (index >= urls.length) return;
      const audio = new Audio(urls[index]);
      sentenceAudioRef.current = audio;
      audio.onended = () => {
        index += 1;
        playNext();
      };
      audio.play().catch(() => {});
    };
    playNext();
  };

  const handlePick = async (sentenceId: number) => {
    if (phase !== "selecting" || checking || correctIds.has(sentenceId)) {
      return;
    }
    setChecking(true);
    setCheckError(null);
    try {
      const result = await onCheckPick(sentenceId, nextPosition);
      setAttempts((n) => n + 1);
      if (result.correct) {
        const nextCorrect = new Set(correctIds);
        nextCorrect.add(sentenceId);
        setCorrectIds(nextCorrect);
        setPlacedIds((ids) =>
          ids.includes(sentenceId) ? ids : [...ids, sentenceId],
        );
        setWrongIds(new Set());
        const card = pageData.sentences.find((s) => s.id === sentenceId);
        playSentenceAudio(card?.audio_urls);
        const following = nextPosition + 1;
        if (following > pageData.sentences.length) {
          setPhase("complete");
        } else {
          setNextPosition(following);
        }
      } else {
        setWrongIds((current) => new Set(current).add(sentenceId));
      }
    } catch (err) {
      console.error("Failed to check recall pick:", err);
      setCheckError("Couldn't check your answer. Please try again.");
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
  const ordinal = ORDINALS[nextPosition - 1] ?? String(nextPosition);

  const playButtonLabel =
    phase === "idle"
      ? "Start"
      : phase === "listening"
        ? "Pause Audio"
        : "Resume Audio";

  return (
    <div className="max-w-6xl mx-auto px-4 py-6">
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
              Play the story audio here. When it ends, you'll select the five
              key sentences in the order they occur in the story.
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
                onClick={() => stepLine(-1)}
                disabled={!canStepBack}
                className="inline-flex items-center gap-2 px-4 py-3 bg-gray-100 text-gray-800 border border-gray-300 rounded-lg text-base transition-colors duration-200 cursor-pointer hover:bg-gray-200 disabled:opacity-40 disabled:cursor-not-allowed disabled:hover:bg-gray-100"
                type="button"
              >
                <span className="material-icons">skip_previous</span>
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
            {phase !== "idle" && (
              <button
                onClick={() => stepLine(1)}
                disabled={!canStepForward}
                className="inline-flex items-center gap-2 px-4 py-3 bg-gray-100 text-gray-800 border border-gray-300 rounded-lg text-base transition-colors duration-200 cursor-pointer hover:bg-gray-200 disabled:opacity-40 disabled:cursor-not-allowed disabled:hover:bg-gray-100"
                type="button"
              >
                Forward 1 line
                <span className="material-icons">skip_next</span>
              </button>
            )}
          </div>

          {heardAll && hasSentences && (
            <div className="flex justify-center mt-4">
              <button
                onClick={skipToOrdering}
                className="inline-flex items-center gap-2 px-4 py-3 bg-blue-500 text-white border-none rounded-lg text-base transition-colors duration-200 cursor-pointer hover:bg-blue-600"
                type="button"
                data-testid="recall-skip-to-ordering"
              >
                <span className="material-icons">fast_forward</span>
                Skip to ordering
              </button>
            </div>
          )}
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
        <section className="w-full" data-testid="recall-selecting">
          <div className="bg-gray-50 border border-gray-300 p-3 mb-4 rounded-lg text-center">
            <div className="flex items-start justify-center">
              <span className="material-icons text-gray-600 mr-2 mt-0.5">
                info
              </span>
              <p className="text-gray-700" data-testid="recall-prompt">
                {phase === "complete"
                  ? "Every sentence is in its place. This is the story's order."
                  : `Select the box that occurs ${ordinal} in the story.`}
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

          <div
            className="flex flex-wrap justify-center gap-3 sm:gap-5 w-full max-w-4xl mx-auto"
            data-testid="recall-cards"
            role="list"
          >
            {orderedRecallCards(pageData.sentences, placedIds).map((card) => {
              const result = correctIds.has(card.id)
                ? "correct"
                : wrongIds.has(card.id)
                  ? "wrong"
                  : "pending";
              return (
                <RecallSelectCard
                  key={card.id}
                  card={card}
                  result={result}
                  locked={phase === "complete" || result === "correct"}
                  checking={checking}
                  isRTL={isRTL}
                  onSelect={() => handlePick(card.id)}
                />
              );
            })}
          </div>

          {phase === "selecting" && attempts > 0 && (
            <div className="flex justify-center mt-4">
              <span
                className="text-gray-600 text-sm"
                data-testid="recall-attempts"
              >
                Attempts: <strong>{attempts}</strong>
              </span>
            </div>
          )}

          {phase === "selecting" && hasNarration && (
            <div className="flex justify-center mt-4">
              <button
                onClick={backToAudio}
                disabled={checking}
                className="inline-flex items-center gap-2 px-4 py-3 bg-gray-100 text-gray-800 border border-gray-300 rounded-lg text-base transition-colors duration-200 cursor-pointer hover:bg-gray-200 disabled:opacity-40 disabled:cursor-not-allowed disabled:hover:bg-gray-100"
                type="button"
                data-testid="recall-back-to-audio"
              >
                <span className="material-icons">headphones</span>
                Back to audio
              </button>
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
    // A value past the line count is stale (the story was re-cut); start over
    // rather than treating it as fully heard.
    if (!Number.isInteger(heard) || heard < 0 || heard > lineCount) return 0;
    return heard;
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

function orderedRecallCards(sentences: RecallCard[], placedIds: number[]) {
  const byId = new Map(sentences.map((card) => [card.id, card]));
  const placed = placedIds
    .map((id) => byId.get(id))
    .filter((card): card is RecallCard => card != null);
  const placedSet = new Set(placed.map((card) => card.id));
  return [...placed, ...sentences.filter((card) => !placedSet.has(card.id))];
}

type CardResult = "pending" | "correct" | "wrong";

interface RecallSelectCardProps {
  card: RecallCard;
  result: CardResult;
  locked: boolean;
  checking: boolean;
  isRTL: boolean;
  onSelect: () => void;
}

function RecallSelectCard({
  card,
  result,
  locked,
  checking,
  isRTL,
  onSelect,
}: RecallSelectCardProps) {
  const clickable = !locked && !checking;
  const tone =
    result === "correct"
      ? "border-green-500"
      : result === "wrong"
        ? "border-red-400"
        : "border-gray-200";

  return (
    <button
      type="button"
      role="listitem"
      disabled={!clickable}
      onClick={onSelect}
      data-testid={`recall-card-${card.id}`}
      data-result={result}
      aria-label={
        result === "correct"
          ? "Already placed"
          : result === "wrong"
            ? "Not this one"
            : "Sentence option"
      }
      className={`flex flex-col min-w-0 w-[calc((100%-0.75rem)/2)] sm:w-[calc((100%-2.5rem)/3)] rounded-xl border-4 bg-white shadow-sm overflow-hidden ${tone} ${
        clickable
          ? "cursor-pointer hover:border-primary-400 hover:scale-[1.02] focus:outline-none focus-visible:ring-4 focus-visible:ring-primary-300"
          : result === "correct"
            ? "cursor-default"
            : "cursor-not-allowed"
      }`}
    >
      <div className="relative aspect-[2/1] w-full overflow-hidden bg-slate-50">
        {card.image_url ? (
          <img
            src={card.image_url}
            alt=""
            className="absolute inset-0 h-full w-full object-contain"
            draggable={false}
          />
        ) : (
          <span className="flex h-full w-full items-center justify-center text-slate-400 text-xs p-2 text-center">
            No picture
          </span>
        )}
      </div>
      <p
        className="shrink-0 px-3 py-3 text-center text-2xl sm:text-3xl leading-snug text-gray-900"
        dir={isRTL ? "rtl" : "ltr"}
        lang={isRTL ? "he" : undefined}
      >
        <RecallCardText card={card} />
      </p>
    </button>
  );
}

function RecallCardText({ card }: { card: RecallCard }) {
  if (card.text && card.text.length > 0) {
    return card.text.map((segment, index) =>
      segment.type === "target" ? (
        <span key={index} className="target-word text-amber-700 font-semibold">
          {segment.text}
        </span>
      ) : (
        <span key={index}>{segment.text}</span>
      ),
    );
  }
  return card.hebrew_text;
}
