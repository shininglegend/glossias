import { useState, useEffect, useRef } from "react";
import { useParams, useNavigate, Link } from "react-router";
import { useApiService } from "../services/api";
import { useNavigationGuidance } from "../hooks/useNavigationGuidance";
import { useAuthenticatedFetch } from "../lib/authFetch";
import type { VocabLine } from "../services/api";
import { useAudioPlayer } from "./story-components/AudioPlayer";
import { StoryHeader } from "./story-components/StoryHeader";
import { StoryLine } from "./story-components/StoryLine";
import { CompletionMessage } from "./story-components/CompletionMessage";

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

const WAIT_TIME_WHEN_NOT_KNOWN = 2.5 * 1000;

// Transform translate line to vocab line format
const transformToVocabLine = (
  translateLine: LineWithTranslation
): VocabLine => {
  return {
    text: [{ type: "text", text: translateLine.text }],
    audio_files: [],
    signed_audio_urls: {},
  };
};

export function StoriesTranslate() {
  const { id } = useParams<{ id: string }>();
  const api = useApiService();
  const navigate = useNavigate();
  const { getNavigationGuidance } = useNavigationGuidance();
  const authenticatedFetch = useAuthenticatedFetch();

  const [pageData, setPageData] = useState<TranslatePageData | null>(null);
  const [vocabLines, setVocabLines] = useState<VocabLine[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [audioURLs, setAudioURLs] = useState<Record<string, string>>({});

  const [currentLineIndex, setCurrentLineIndex] = useState(0);
  const [isPlaying, setIsPlaying] = useState(false);
  const [showComprehensionPrompt, setShowComprehensionPrompt] = useState(false);
  const [revealedTranslations, setRevealedTranslations] = useState<Set<number>>(
    new Set()
  );
  const [requestedLineIndices, setRequestedLineIndices] = useState<number[]>(
    []
  );
  const [completedLines, setCompletedLines] = useState<Set<number>>(new Set());
  const [playedLines, setPlayedLines] = useState<Set<number>>(new Set());
  const [isAutoWaiting, setIsAutoWaiting] = useState(false);
  const [allLinesCompleted, setAllLinesCompleted] = useState(false);
  const [nextStepName, setNextStepName] = useState<string>("Next Step");

  const currentAudioRef = useRef<HTMLAudioElement | null>(null);
  const autoWaitTimeoutRef = useRef<NodeJS.Timeout | null>(null);

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
          `/api/stories/${id}/translate`
        );
        if (translateResponse.ok) {
          const translateData = await translateResponse.json();
          if (translateData.success && translateData.data) {
            setPageData(translateData.data);

            // Transform lines to VocabLine format
            const transformed = translateData.data.lines.map(
              (line: LineWithTranslation) => transformToVocabLine(line)
            );
            setVocabLines(transformed);

            // Fetch audio URLs
            const audioResponse = await authenticatedFetch(
              `/api/stories/${id}/audio/signed?label=complete`
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
  }, [id, authenticatedFetch]);

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

  // Audio player integration
  const audioPlayerData =
    vocabLines.length > 0
      ? {
          story_id: pageData?.story_id || "",
          story_title: pageData?.story_title || "",
          language: pageData?.language || "he",
          lines: vocabLines,
          vocab_bank: [],
        }
      : null;

  const audioPlayer = useAudioPlayer({
    audioURLs,
    pageData: audioPlayerData,
    onPlayedLinesChange: setPlayedLines,
    onCurrentLineChange: setCurrentLineIndex,
    onPlayingStateChange: setIsPlaying,
    completedLines,
    pauseAfterEveryLine: true,
  });

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (currentAudioRef.current) {
        currentAudioRef.current.pause();
      }
      if (autoWaitTimeoutRef.current) {
        clearTimeout(autoWaitTimeoutRef.current);
      }
    };
  }, []);

  // Watch for line completion to show comprehension prompt
  // When a line finishes playing, it gets added to playedLines and isPlaying becomes false
  useEffect(() => {
    // Check if the current line just finished playing (is in playedLines but not completed)
    if (
      playedLines.has(currentLineIndex) &&
      !completedLines.has(currentLineIndex) &&
      !showComprehensionPrompt &&
      !allLinesCompleted &&
      !isAutoWaiting
    ) {
      // Pause audio before showing prompt
      audioPlayer.pauseAudio();
      setShowComprehensionPrompt(true);
    }
  }, [
    playedLines,
    currentLineIndex,
    completedLines,
    showComprehensionPrompt,
    allLinesCompleted,
    isAutoWaiting,
    audioPlayer,
  ]);

  const handleComprehendYes = () => {
    setShowComprehensionPrompt(false);
    moveToNextLine();
  };

  const handleComprehendNo = async () => {
    setShowComprehensionPrompt(false);
    const lineNumber = currentLineIndex + 1;
    setRevealedTranslations((prev) => new Set([...prev, lineNumber]));

    // Track this line as requested (0-indexed for API)
    setRequestedLineIndices((prev) => [...prev, currentLineIndex]);

    setIsAutoWaiting(true);

    // Wait x seconds before auto-continuing
    autoWaitTimeoutRef.current = setTimeout(() => {
      setIsAutoWaiting(false);
      moveToNextLine();
    }, WAIT_TIME_WHEN_NOT_KNOWN);
  };

  const moveToNextLine = async () => {
    if (!pageData) return;

    // Mark current line as completed
    setCompletedLines((prev) => new Set([...prev, currentLineIndex]));

    const nextIndex = currentLineIndex + 1;
    if (nextIndex < pageData.lines.length) {
      audioPlayer.playNextLineFromIndex(currentLineIndex);
    } else {
      // All lines completed - save final requested lines
      await saveRequestedLines(requestedLineIndices);
      setAllLinesCompleted(true);
      setIsPlaying(false);
    }
  };

  const saveRequestedLines = async (lines: number[]) => {
    if (!id) return;

    try {
      const url = new URL(`/api/stories/${id}/translate`, window.location.origin);
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
  };

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

  const RTL_LANGUAGES = ["he", "ar", "fa", "ur"];
  const isRTL = RTL_LANGUAGES.includes(pageData.language);

  const togglePlayPause = () => {
    if (isPlaying) {
      audioPlayer.pauseAudio();
      if (autoWaitTimeoutRef.current) {
        clearTimeout(autoWaitTimeoutRef.current);
      }
      setIsPlaying(false);
      setIsAutoWaiting(false);
    } else {
      // Play/Resume
      audioPlayer.playStoryAudio();
    }
  };

  return (
    <>
      <header>
        <h1>{pageData.story_title}</h1>
        <h2>Translation</h2>

        <div className="bg-gray-50 border border-gray-300 p-4 mb-4 rounded-lg text-center">
          <div className="flex items-start justify-center">
            <span className="material-icons text-gray-600 mr-2 mt-1">info</span>
            <div>
              <p className="text-gray-700 mb-2">
                Listen to each line of the story. After each line, indicate
                whether you fully comprehend it. If not, the English translation
                will be revealed.
              </p>
            </div>
          </div>
        </div>

        <button
          onClick={togglePlayPause}
          className={`inline-flex items-center gap-2 px-5 py-3 my-5 text-white border-none rounded-lg text-base cursor-pointer transition-colors duration-200 ${
            isPlaying
              ? "bg-red-500 hover:bg-red-600"
              : "bg-green-500 hover:bg-green-600"
          }`}
          type="button"
        >
          <span className="material-icons">
            {isPlaying ? "pause" : "play_arrow"}
          </span>
          {isPlaying ? "Pause Audio" : "Play Audio"}
        </button>
      </header>

      <div className={`max-w-4xl mx-auto px-5 pb-16`}>
        <div className="story-lines text-2xl max-w-3xl mx-auto">
          {vocabLines.length > 0 && (
            <div
              className={isRTL ? "text-right" : "text-left"}
              dir={isRTL ? "rtl" : "ltr"}
            >
              {vocabLines.map((line, lineIndex) => {
                const lineNumber = pageData!.lines[lineIndex].line_number;
                const translation = pageData!.lines[lineIndex].translation;
                const showTranslation = revealedTranslations.has(lineNumber);

                return (
                  <StoryLine
                    key={lineIndex}
                    line={line}
                    lineIndex={lineIndex}
                    vocabBank={[]}
                    selectedAnswers={{}}
                    lineResults={{}}
                    completedLines={completedLines}
                    playedLines={playedLines}
                    checkingLines={new Set()}
                    isCurrentLine={
                      (currentLineIndex === lineIndex && isPlaying) ||
                      (currentLineIndex === lineIndex &&
                        showComprehensionPrompt)
                    }
                    isRTL={!!isRTL}
                    prefetchedAudio={audioPlayer.prefetchedAudio}
                    originalLine={undefined}
                    pendingAnswers={new Set()}
                    lockedAnswers={new Set()}
                    onAnswerChange={() => {}}
                    onPlayLineAudio={audioPlayer.playLineAudio}
                    translation={translation}
                    showTranslation={showTranslation}
                  />
                );
              })}
            </div>
          )}
        </div>

        {allLinesCompleted && (
          <CompletionMessage
            currentStepName="translation"
            nextStepName={nextStepName}
            onContinue={handleContinue}
          />
        )}
      </div>

      {/* Comprehension Prompt - Inline at bottom */}
      {showComprehensionPrompt && (
        <div className="fixed bottom-0 left-0 right-0 bg-white border-t-4 border-blue-500 shadow-2xl z-50">
          <div className="max-w-2xl mx-auto px-6 py-4">
            <div className="flex items-center justify-between gap-6">
              <p className="text-lg font-semibold text-gray-800">
                Do you fully comprehend this line?
              </p>
              <div className="flex gap-3">
                <button
                  onClick={handleComprehendYes}
                  className="px-6 py-2 bg-green-500 text-white rounded-lg hover:bg-green-600 font-medium transition-colors"
                >
                  Yes
                </button>
                <button
                  onClick={handleComprehendNo}
                  className="px-6 py-2 bg-orange-500 text-white rounded-lg hover:bg-orange-600 font-medium transition-colors"
                >
                  No, show translation
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Auto-waiting indicator */}
      {isAutoWaiting && (
        <div className="fixed bottom-8 left-1/2 transform -translate-x-1/2 bg-blue-500 text-white px-6 py-3 rounded-lg shadow-lg">
          <p className="text-center">
            Translation revealed. Continuing in a moment...
          </p>
        </div>
      )}
    </>
  );
}
