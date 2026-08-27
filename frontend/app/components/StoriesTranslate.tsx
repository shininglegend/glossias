import { useState, useEffect, useRef, useReducer, useCallback } from "react";
import { useParams, useNavigate, Link } from "react-router";
import { useNavigationGuidance } from "../hooks/useNavigationGuidance";
import { useAuthenticatedFetch } from "../lib/authFetch";
import type { VocabLine } from "../services/api";
import { useAudioPlayer } from "./story-components/AudioPlayer";
import { StoryLine } from "./story-components/StoryLine";
import { CompletionMessage } from "./story-components/CompletionMessage";
import {
  translateReducer,
  createTranslateState,
  canRequest,
  canFastForward,
  requestBlockReason,
  effectiveMinRequests,
  MAX_REQUESTS,
  PREDICT_MS,
  REVEAL_MS,
} from "../lib/translateMachine";

interface LineWithTranslation {
  text: string;
  translation: string;
  line_number: number;
}

interface TranslatePageData {
  story_id: string;
  story_title: string;
  language: string;
  lines: LineWithTranslation[];
  has_translation: boolean;
}

const RTL_LANGUAGES = ["he", "ar", "fa", "ur"];
const EMPTY_SET = new Set<number>();
const noop = () => {};

// Transform translate line to vocab line format
const transformToVocabLine = (
  translateLine: LineWithTranslation,
): VocabLine => {
  return {
    text: [{ type: "text", text: translateLine.text }],
    audio_files: [],
    signed_audio_urls: {},
  };
};

/**
 * Loads the story and hands it to `TranslateSession`, which owns the
 * playback state machine (created with the real line count on mount).
 */
export function StoriesTranslate() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { getNavigationGuidance } = useNavigationGuidance();
  const authenticatedFetch = useAuthenticatedFetch();

  const [pageData, setPageData] = useState<TranslatePageData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [audioURLs, setAudioURLs] = useState<Record<string, string>>({});
  const [nextStepName, setNextStepName] = useState<string>("Next Step");

  useEffect(() => {
    const fetchData = async () => {
      if (!id) {
        setError("Story ID is required");
        setLoading(false);
        return;
      }

      try {
        // Fetch translation page data (GET request returns all lines with translations)
        const translateResponse = await authenticatedFetch(
          `/api/stories/${id}/translate`,
        );
        if (translateResponse.ok) {
          const translateData = await translateResponse.json();
          if (translateData.success && translateData.data) {
            setPageData(translateData.data);

            // Fetch audio URLs
            const audioResponse = await authenticatedFetch(
              `/api/stories/${id}/audio/signed?label=complete`,
            );
            if (audioResponse.ok) {
              const audioData = await audioResponse.json();
              if (audioData.success) {
                setAudioURLs(audioData.data);
              }
            }
          } else {
            setError("Failed to fetch translation data");
          }
        } else {
          setError("Failed to fetch translation data");
        }
      } catch (err) {
        console.error("Failed to fetch data:", err);
        setError("Failed to fetch page data");
      } finally {
        setLoading(false);
      }
    };

    fetchData();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id]);

  useEffect(() => {
    const fetchNextStep = async () => {
      if (!id) return;
      try {
        const guidance = await getNavigationGuidance(id, "translate");
        if (guidance) {
          setNextStepName(guidance.displayName);
        }
      } catch (error) {
        console.error("Failed to get navigation guidance:", error);
      }
    };

    fetchNextStep();
  }, [id, getNavigationGuidance]);

  const saveRequestedLines = useCallback(
    async (lines: number[]) => {
      if (!id) return;
      try {
        const url = new URL(
          `/api/stories/${id}/translate`,
          window.location.origin,
        );
        url.searchParams.set("lines", `[${lines.join(",")}]`);

        const response = await authenticatedFetch(url.toString(), {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
        });

        if (!response.ok) {
          console.warn("Failed to save requested lines:", response.statusText);
        }
      } catch (error) {
        console.warn("Failed to save requested lines:", error);
      }
    },
    [id, authenticatedFetch],
  );

  const handleContinue = async () => {
    try {
      const guidance = await getNavigationGuidance(id!, "translate");
      if (guidance) {
        navigate(`/stories/${id}/${guidance.nextPage}`);
      }
    } catch (error) {
      console.error("Failed to get navigation guidance:", error);
    }
  };

  if (loading) {
    return (
      <div className="container">
        <p>Loading page...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="container">
        <p>Error: {error}</p>
        <Link to="/">Back to Stories</Link>
      </div>
    );
  }

  if (!pageData) {
    return (
      <div className="container">
        <p>No page data found</p>
        <Link to="/">Back to Stories</Link>
      </div>
    );
  }

  return (
    <TranslateSession
      pageData={pageData}
      audioURLs={audioURLs}
      nextStepName={nextStepName}
      onComplete={saveRequestedLines}
      onContinue={handleContinue}
    />
  );
}

interface TranslateSessionProps {
  pageData: TranslatePageData;
  audioURLs: Record<string, string>;
  nextStepName: string;
  /** Called once with the 0-based requested line indices when the phase ends. */
  onComplete: (requestedLines: number[]) => void;
  onContinue: () => void;
}

function TranslateSession({
  pageData,
  audioURLs,
  nextStepName,
  onComplete,
  onContinue,
}: TranslateSessionProps) {
  const [vocabLines] = useState<VocabLine[]>(() =>
    pageData.lines.map(transformToVocabLine),
  );

  // Audio-player status mirrored into React state.
  const [audioLineIndex, setAudioLineIndex] = useState(0);
  const [playedLines, setPlayedLines] = useState<Set<number>>(new Set());

  const [state, dispatch] = useReducer(
    translateReducer,
    pageData.lines.length,
    createTranslateState,
  );
  const { phase, pass, requested, revealed, lineCount, command, commandSeq } =
    state;

  const onStoryEnded = useCallback(() => dispatch({ type: "STORY_ENDED" }), []);
  // The hook stopped at the end of the line a request was waiting on. This is
  // an explicit event from the hook (not inferred from isPlaying), so a user
  // pause/resume while waiting cannot be mistaken for the line ending.
  const onLineEnded = useCallback(() => dispatch({ type: "LINE_ENDED" }), []);

  // While a request is pending, pause after whichever line is playing. The
  // hook reads this at `ended` time, so if the click lands as a line ends the
  // pause simply happens after the next line instead.
  const pauseOnLines =
    phase.kind === "awaitingLineEnd" ? new Set([audioLineIndex]) : EMPTY_SET;
  // On restart passes, lines already translated are skipped.
  const skipLines = pass > 1 ? new Set(requested) : EMPTY_SET;

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
    onPlayingStateChange: noop,
    completedLines: EMPTY_SET,
    pauseOnLines,
    skipLines,
    onPlaybackEnd: onStoryEnded,
    onPauseAfterLine: onLineEnded,
  });

  // Mirror the audio player's line into the machine.
  useEffect(() => {
    dispatch({ type: "LINE_CHANGED", index: audioLineIndex });
  }, [audioLineIndex]);

  // Timed transitions: 2s prediction beat, then 5s reveal hold.
  useEffect(() => {
    if (phase.kind === "predicting") {
      const t = setTimeout(
        () => dispatch({ type: "PREDICT_DONE" }),
        PREDICT_MS,
      );
      return () => clearTimeout(t);
    }
    if (phase.kind === "revealing") {
      const t = setTimeout(() => dispatch({ type: "REVEAL_DONE" }), REVEAL_MS);
      return () => clearTimeout(t);
    }
  }, [phase.kind]);

  // Run each audio command the machine emits exactly once.
  const lastCommandSeqRef = useRef(0);
  const { playNextLineFromIndex } = audioPlayer;
  useEffect(() => {
    if (commandSeq === lastCommandSeqRef.current || !command) return;
    lastCommandSeqRef.current = commandSeq;
    if (command.type === "playFrom") {
      // The continuation API starts at index + 1, honouring skipLines.
      playNextLineFromIndex(command.index - 1);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [commandSeq]);

  // Persist the requested lines once, on completion.
  const savedRef = useRef(false);
  useEffect(() => {
    if (phase.kind !== "complete" || savedRef.current) return;
    savedRef.current = true;
    onComplete(requested);
  }, [phase.kind, requested, onComplete]);

  const handlePlayPause = () => {
    if (phase.kind === "idle") {
      dispatch({ type: "START" });
    } else if (phase.kind === "playing" || phase.kind === "awaitingLineEnd") {
      audioPlayer.pauseAudio();
      dispatch({ type: "PAUSE" });
    } else if (phase.kind === "paused") {
      dispatch({ type: "RESUME" });
    }
  };

  const handleFastForward = () => {
    if (!canFastForward(state)) return;
    audioPlayer.playNextLineFromIndex(audioLineIndex);
  };

  const handleLineClick = (lineIndex: number) => {
    if (!canRequest(state, lineIndex)) return;
    dispatch({ type: "REQUEST", line: lineIndex });
  };

  const isRTL = RTL_LANGUAGES.includes(pageData.language);
  const minRequests = effectiveMinRequests(lineCount);
  const isComplete = phase.kind === "complete";
  // Audio can be paused while playing or while a request waits for its line
  // to finish; the timed beats cannot be interrupted.
  const isTimed = phase.kind === "predicting" || phase.kind === "revealing";
  const isAudible =
    phase.kind === "playing" || phase.kind === "awaitingLineEnd";
  const activeLine =
    phase.kind === "predicting" || phase.kind === "revealing"
      ? phase.requestedLine
      : phase.kind === "playing" || phase.kind === "awaitingLineEnd"
        ? audioLineIndex
        : null;

  // Explain why the student cannot request right now, if playing.
  const blockHint = (() => {
    if (phase.kind !== "playing") return null;
    if (requested.length >= MAX_REQUESTS) {
      return `You've used all ${MAX_REQUESTS} translations.`;
    }
    const reasons = [audioLineIndex, audioLineIndex - 1].map((l) =>
      requestBlockReason(state, l),
    );
    if (reasons.includes("consecutive-cap") && !reasons.includes(null)) {
      return "Three lines in a row translated — let a line play untranslated first.";
    }
    return null;
  })();

  const playButtonLabel =
    phase.kind === "idle"
      ? "Start"
      : isAudible
        ? "Pause Audio"
        : phase.kind === "paused"
          ? "Resume Audio"
          : "Playing…";

  return (
    <>
      <header>
        <h1>{pageData.story_title}</h1>
        <h2>Translation</h2>

        {isComplete && (
          <CompletionMessage
            currentStepName="translation"
            nextStepName={nextStepName}
            onContinue={onContinue}
          />
        )}

        <div className="bg-gray-50 border border-gray-300 p-4 mb-4 rounded-lg text-center">
          <div className="flex items-start justify-center">
            <span className="material-icons text-gray-600 mr-2 mt-1">info</span>
            <div>
              <p className="text-gray-700 mb-2">
                Listen to the story. Click the line that is playing (or the one
                just before it) to see its English translation.
              </p>
              <p className="text-gray-700">
                Request between {minRequests} and {MAX_REQUESTS} translations,
                with no more than 3 lines in a row.
              </p>
            </div>
          </div>
        </div>

        {!isComplete && (
          <div className="flex flex-wrap items-center justify-center gap-4 my-5">
            <button
              onClick={handlePlayPause}
              disabled={isTimed}
              className={`inline-flex items-center gap-2 px-5 py-3 text-white border-none rounded-lg text-base transition-colors duration-200 ${
                isTimed
                  ? "bg-gray-400 cursor-not-allowed"
                  : isAudible
                    ? "bg-red-500 hover:bg-red-600 cursor-pointer"
                    : "bg-green-500 hover:bg-green-600 cursor-pointer"
              }`}
              type="button"
            >
              <span className="material-icons">
                {isAudible || isTimed ? "pause" : "play_arrow"}
              </span>
              {playButtonLabel}
            </button>

            {pass > 1 && (
              <button
                onClick={handleFastForward}
                disabled={!canFastForward(state)}
                className={`inline-flex items-center gap-2 px-5 py-3 text-white border-none rounded-lg text-base transition-colors duration-200 ${
                  canFastForward(state)
                    ? "bg-blue-500 hover:bg-blue-600 cursor-pointer"
                    : "bg-gray-400 cursor-not-allowed"
                }`}
                type="button"
                aria-label="Skip to the next line"
              >
                <span className="material-icons">fast_forward</span>
                Skip line
              </button>
            )}

            <div
              className="text-gray-700 text-base"
              aria-live="polite"
              data-testid="translation-counter"
            >
              Translations: <strong>{requested.length}</strong> / {minRequests}–
              {MAX_REQUESTS}
            </div>
          </div>
        )}

        {pass > 1 && !isComplete && (
          <div className="bg-yellow-50 border-l-4 border-yellow-400 p-3 mb-4 rounded-r-lg text-left">
            <p className="text-gray-800">
              You need at least {minRequests} translations. The story is playing
              again — lines you already translated are skipped, and you can skip
              ahead with the button above.
            </p>
          </div>
        )}

        {blockHint && (
          <p className="text-gray-600 text-sm mb-4" aria-live="polite">
            {blockHint}
          </p>
        )}
      </header>

      <div className="max-w-4xl mx-auto px-5 pb-24">
        <div className="story-lines text-2xl max-w-3xl mx-auto">
          <div
            className={isRTL ? "text-right" : "text-left"}
            dir={isRTL ? "rtl" : "ltr"}
          >
            {vocabLines.map((line, lineIndex) => {
              const translation = pageData.lines[lineIndex].translation;
              const requestable = canRequest(state, lineIndex);
              const isRequested = requested.includes(lineIndex);

              return (
                <div
                  key={lineIndex}
                  className={`inline ${
                    requestable
                      ? "cursor-pointer hover:bg-primary-50 rounded"
                      : isRequested
                        ? "text-primary-800"
                        : ""
                  }`}
                  role={requestable ? "button" : undefined}
                  tabIndex={requestable ? 0 : undefined}
                  aria-label={
                    requestable
                      ? `Show translation for line ${lineIndex + 1}`
                      : undefined
                  }
                  onClick={() => handleLineClick(lineIndex)}
                  onKeyDown={(e) => {
                    if (requestable && (e.key === "Enter" || e.key === " ")) {
                      e.preventDefault();
                      handleLineClick(lineIndex);
                    }
                  }}
                  data-testid={`translate-line-${lineIndex}`}
                >
                  <StoryLine
                    line={line}
                    lineIndex={lineIndex}
                    vocabBank={[]}
                    selectedAnswers={{}}
                    lineResults={{}}
                    completedLines={EMPTY_SET}
                    playedLines={playedLines}
                    checkingLines={EMPTY_SET}
                    isCurrentLine={activeLine === lineIndex}
                    isRTL={isRTL}
                    prefetchedAudio={audioPlayer.prefetchedAudio}
                    originalLine={undefined}
                    pendingAnswers={new Set()}
                    lockedAnswers={new Set()}
                    onAnswerChange={() => {}}
                    onPlayLineAudio={() => {}}
                    translation={translation}
                    showTranslation={revealed.includes(lineIndex)}
                  />
                </div>
              );
            })}
          </div>
        </div>
      </div>

      {/* Pending-request status */}
      {phase.kind === "awaitingLineEnd" && (
        <div className="fixed bottom-8 left-1/2 transform -translate-x-1/2 bg-gray-700 text-white px-6 py-3 rounded-lg shadow-lg">
          <p className="text-center">Finishing this line…</p>
        </div>
      )}
      {phase.kind === "predicting" && (
        <div className="fixed bottom-8 left-1/2 transform -translate-x-1/2 bg-primary-500 text-white px-6 py-3 rounded-lg shadow-lg">
          <p className="text-center">
            What do you think line {phase.requestedLine + 1} means?
          </p>
        </div>
      )}
      {phase.kind === "revealing" && (
        <div className="fixed bottom-8 left-1/2 transform -translate-x-1/2 bg-primary-500 text-white px-6 py-3 rounded-lg shadow-lg">
          <p className="text-center">Translation revealed. Resuming shortly…</p>
        </div>
      )}
    </>
  );
}
