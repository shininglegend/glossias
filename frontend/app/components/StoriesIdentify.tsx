import {
  useState,
  useEffect,
  useRef,
  useReducer,
  useCallback,
  useMemo,
} from "react";
import { useParams, useNavigate, Link } from "react-router";
import { useApiService } from "../services/api";
import type {
  IdentifyData,
  IdentifyTargetWord,
  VocabLine,
} from "../services/api";
import { useNavigationGuidance } from "../hooks/useNavigationGuidance";
import { useAudioPlayer } from "./story-components/AudioPlayer";
import { StoryLine } from "./story-components/StoryLine";
import { CompletionMessage } from "./story-components/CompletionMessage";
import { IdentifyQuizModal } from "./story-components/IdentifyQuizModal";
import {
  identifyReducer,
  createIdentifyState,
  identifiedWords,
  linesWithUnansweredTargets,
} from "../lib/identifyMachine";
import "./StoriesVocab.css";

const RTL_LANGUAGES = ["he", "ar", "fa", "ur"];
const EMPTY_SET = new Set<number>();

/**
 * Loads the Identify payload and hands it to `IdentifySession`, which owns the
 * playback/quiz state machine (created with the real line count on mount).
 */
export function StoriesIdentify() {
  const { id } = useParams<{ id: string }>();
  const api = useApiService();
  const navigate = useNavigate();
  const { getNavigationGuidance } = useNavigationGuidance();

  const [pageData, setPageData] = useState<IdentifyData | null>(null);
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
        const response = await api.getStoryIdentify(id);
        if (response.success && response.data) {
          setPageData(response.data);
        } else {
          setError(response.error || "Failed to fetch story");
        }
      } catch (err) {
        console.error("Failed to fetch identify data:", err);
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
        const guidance = await getNavigationGuidance(id, "identify");
        if (guidance) setNextStepName(guidance.displayName);
      } catch (err) {
        console.error("Failed to get navigation guidance:", err);
      }
    };
    fetchNextStep();
  }, [id, getNavigationGuidance]);

  const checkPick = useCallback(
    async (lineIndex: number, targetId: number, selectedId: number) => {
      if (!id) throw new Error("Story ID is required");
      const response = await api.checkIdentify(
        id,
        lineIndex,
        targetId,
        selectedId,
      );
      if (!response.success || !response.data) {
        throw new Error(response.error || "Failed to check answer");
      }
      return response.data.correct;
    },
    [id, api],
  );

  const handleContinue = async () => {
    if (!id) return;
    try {
      const guidance = await getNavigationGuidance(id, "identify");
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
    <IdentifySession
      pageData={pageData}
      nextStepName={nextStepName}
      onCheckPick={checkPick}
      onContinue={handleContinue}
    />
  );
}

interface IdentifySessionProps {
  pageData: IdentifyData;
  nextStepName: string;
  /** Grades a pick server-side; resolves to whether it was correct. */
  onCheckPick: (
    lineIndex: number,
    targetId: number,
    selectedId: number,
  ) => Promise<boolean>;
  onContinue: () => void;
}

export function IdentifySession({
  pageData,
  nextStepName,
  onCheckPick,
  onContinue,
}: IdentifySessionProps) {
  // The audio hook speaks VocabData; adapt once.
  const [vocabLines] = useState<VocabLine[]>(() =>
    pageData.lines.map((line) => ({
      text: line.text,
      audio_files: [],
      signed_audio_urls: {},
    })),
  );
  const audioURLs = useMemo<Record<string, string>>(
    () => ({ ...pageData.audio_urls }),
    [pageData.audio_urls],
  );
  const wordsById = useMemo(() => {
    const map = new Map<number, IdentifyTargetWord>();
    for (const w of pageData.target_words) map.set(w.id, w);
    return map;
  }, [pageData.target_words]);

  // Audio-player status mirrored into React state.
  const [audioLineIndex, setAudioLineIndex] = useState(0);
  const [isPlaying, setIsPlaying] = useState(false);
  const [playedLines, setPlayedLines] = useState<Set<number>>(new Set());

  const [state, dispatch] = useReducer(identifyReducer, pageData, (data) =>
    createIdentifyState(data.lines.length, {
      picks: (data.correct_picks ?? []).map((p) => ({
        line: p.line_index,
        targetId: p.target_vocab_id,
      })),
      completed: data.completed,
    }),
  );
  const { phase, command, commandSeq, picks } = state;

  // Pause only after lines that still hold an unanswered target word, so a
  // resumed visit plays straight through the quizzes already answered.
  const pauseOnLines = useMemo(
    () =>
      linesWithUnansweredTargets(
        picks,
        pageData.lines.map((l) => l.target_vocab_ids),
      ),
    [picks, pageData.lines],
  );

  const onStoryEnded = useCallback(() => dispatch({ type: "STORY_ENDED" }), []);
  const linesRef = useRef(pageData.lines);
  linesRef.current = pageData.lines;
  const onPauseAfterLine = useCallback((lineIndex: number) => {
    dispatch({
      type: "LINE_ENDED",
      line: lineIndex,
      targets: linesRef.current[lineIndex]?.target_vocab_ids ?? [],
    });
  }, []);

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
    pauseOnLines,
    onPlaybackEnd: onStoryEnded,
    onPauseAfterLine,
  });

  // Mirror the audio player's line into the machine.
  useEffect(() => {
    dispatch({ type: "LINE_CHANGED", index: audioLineIndex });
  }, [audioLineIndex]);

  // Run each audio command the machine emits exactly once.
  const lastCommandSeqRef = useRef(0);
  const { playNextLineFromIndex, replayLine } = audioPlayer;
  useEffect(() => {
    if (commandSeq === lastCommandSeqRef.current || !command) return;
    lastCommandSeqRef.current = commandSeq;
    if (command.type === "playFrom") {
      // The continuation API starts at index + 1.
      playNextLineFromIndex(command.index - 1);
    } else if (command.type === "replay") {
      let cancelled = false;
      replayLine(command.line).then(() => {
        if (!cancelled) dispatch({ type: "REPLAY_DONE" });
      });
      return () => {
        cancelled = true;
      };
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [commandSeq]);

  // Picture picks: graded by the server, then fed back to the machine.
  const [checking, setChecking] = useState(false);
  const [pickError, setPickError] = useState<string | null>(null);
  const handlePick = async (selectedId: number) => {
    if (phase.kind !== "quiz" || checking) return;
    const { line, targetId } = phase;
    setChecking(true);
    setPickError(null);
    try {
      const correct = await onCheckPick(line, targetId, selectedId);
      dispatch({ type: "PICK_RESULT", selected: selectedId, correct });
    } catch (err) {
      console.error("Failed to check identify pick:", err);
      setPickError("Couldn't save your answer. Please try again.");
    } finally {
      setChecking(false);
    }
  };

  const handlePlayPause = () => {
    if (phase.kind === "idle") {
      dispatch({ type: "START" });
    } else if (phase.kind === "playing") {
      audioPlayer.pauseAudio();
      dispatch({ type: "PAUSE" });
    } else if (phase.kind === "paused") {
      dispatch({ type: "RESUME" });
    }
  };

  const isRTL = RTL_LANGUAGES.includes(pageData.language);
  const isComplete = phase.kind === "complete";
  const canToggle =
    phase.kind === "idle" ||
    phase.kind === "playing" ||
    phase.kind === "paused";
  const activeLine =
    phase.kind === "quiz" || phase.kind === "replaying"
      ? phase.line
      : phase.kind === "playing" && isPlaying
        ? audioLineIndex
        : null;
  const hasTargets = pageData.target_words.length > 0;
  const identified = identifiedWords(picks);
  const identifiedCount = pageData.target_words.filter((w) =>
    identified.includes(w.id),
  ).length;
  const resumedMidway =
    phase.kind === "idle" && picks.length > 0 && state.currentLine > 0;

  const playButtonLabel =
    phase.kind === "idle"
      ? "Start"
      : phase.kind === "playing"
        ? "Pause Audio"
        : phase.kind === "paused"
          ? "Resume Audio"
          : phase.kind === "replaying"
            ? "Replaying…"
            : "Quiz…";

  return (
    <>
      <header>
        <h1>{pageData.story_title}</h1>
        <h2>Identify</h2>

        {isComplete && (
          <CompletionMessage
            currentStepName="identify"
            nextStepName={nextStepName}
            onContinue={onContinue}
          />
        )}

        {isComplete && hasTargets && (
          <div
            className="bg-green-50 border-l-4 border-green-400 p-3 mb-4 rounded-r-lg text-left"
            data-testid="identify-finished"
          >
            <p className="text-gray-800">
              You identified all {pageData.target_words.length} target words in
              this story. This phase is finished and can't be repeated — the
              story text is shown below for reference.
            </p>
          </div>
        )}

        {resumedMidway && (
          <div
            className="bg-yellow-50 border-l-4 border-yellow-400 p-3 mb-4 rounded-r-lg text-left"
            data-testid="identify-resumed"
          >
            <p className="text-gray-800">
              Welcome back — you'll pick up from line {state.currentLine + 1},
              the last line where you identified a word. Words you already
              identified won't be asked again.
            </p>
          </div>
        )}

        <div className="bg-gray-50 border border-gray-300 p-4 mb-4 rounded-lg text-center">
          <div className="flex items-start justify-center">
            <span className="material-icons text-gray-600 mr-2 mt-1">info</span>
            <div>
              <p className="text-gray-700 mb-2">
                Listen to the story and follow along. The{" "}
                <span className="text-amber-700 font-semibold underline decoration-amber-400 decoration-2 underline-offset-4">
                  highlighted words
                </span>{" "}
                are this story's target vocabulary.
              </p>
              <p className="text-gray-700">
                After a line with a target word, the audio pauses: pick the
                picture that matches the word, then the line plays again.
              </p>
            </div>
          </div>
        </div>

        {!hasTargets && (
          <div className="bg-yellow-50 border-l-4 border-yellow-400 p-3 mb-4 rounded-r-lg text-left">
            <p className="text-gray-800">
              This story has no target vocabulary yet, so there are no picture
              quizzes. Listen through, then continue.
            </p>
          </div>
        )}

        {!isComplete && (
          <div className="flex flex-wrap items-center justify-center gap-4 my-5">
            <button
              onClick={handlePlayPause}
              disabled={!canToggle}
              className={`inline-flex items-center gap-2 px-5 py-3 text-white border-none rounded-lg text-base transition-colors duration-200 ${
                !canToggle
                  ? "bg-gray-400 cursor-not-allowed"
                  : phase.kind === "playing"
                    ? "bg-red-500 hover:bg-red-600 cursor-pointer"
                    : "bg-green-500 hover:bg-green-600 cursor-pointer"
              }`}
              type="button"
            >
              <span className="material-icons">
                {phase.kind === "playing" || !canToggle
                  ? "pause"
                  : "play_arrow"}
              </span>
              {playButtonLabel}
            </button>

            {hasTargets && (
              <div
                className="text-gray-700 text-base"
                aria-live="polite"
                data-testid="identify-counter"
              >
                Words identified: <strong>{identifiedCount}</strong> /{" "}
                {pageData.target_words.length}
              </div>
            )}
          </div>
        )}
      </header>

      <div className="max-w-4xl mx-auto px-5 pb-24">
        <div className="story-lines text-2xl max-w-3xl mx-auto">
          <div
            className={isRTL ? "text-right" : "text-left"}
            dir={isRTL ? "rtl" : "ltr"}
          >
            {vocabLines.map((line, lineIndex) => (
              <div
                key={lineIndex}
                className="inline"
                data-testid={`identify-line-${lineIndex}`}
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
                  pendingAnswers={EMPTY_STRING_SET}
                  lockedAnswers={EMPTY_STRING_SET}
                  onAnswerChange={() => {}}
                  onPlayLineAudio={() => {}}
                />{" "}
              </div>
            ))}
          </div>
        </div>
      </div>

      <IdentifyQuizModal
        isOpen={phase.kind === "quiz"}
        target={
          phase.kind === "quiz" ? wordsById.get(phase.targetId) : undefined
        }
        options={pageData.target_words}
        wrongPicks={phase.kind === "quiz" ? phase.wrongPicks : []}
        checking={checking}
        error={pickError}
        isRTL={isRTL}
        onPick={handlePick}
      />
    </>
  );
}

const EMPTY_STRING_SET = new Set<string>();
