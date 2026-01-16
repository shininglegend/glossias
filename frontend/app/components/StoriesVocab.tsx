import { useState, useEffect } from "react";
import { useParams, useNavigate, Link } from "react-router";
import { useApiService } from "../services/api";
import { useNavigationGuidance } from "../hooks/useNavigationGuidance";
import { useAuthenticatedFetch } from "../lib/authFetch";
import type { VocabData, VocabLine } from "../services/api";
import { useAudioPlayer } from "./story-components/AudioPlayer";
import { StoryHeader } from "./story-components/StoryHeader";
import { StoryLine } from "./story-components/StoryLine";
import { CompletionMessage } from "./story-components/CompletionMessage";

import "./StoriesVocab.css";

interface AudioURLsResponse {
  success: boolean;
  data: Record<string, string>; // lineNumber -> signedURL
}

// Helper function to check if a line contains vocabulary placeholders
const lineHasVocab = (line: VocabLine): boolean => {
  return line.text.some((segment) => segment.type === "blank");
};

export function StoriesVocab() {
  const { id } = useParams<{ id: string }>();
  const api = useApiService();
  const navigate = useNavigate();
  const { getNavigationGuidance } = useNavigationGuidance();
  const authenticatedFetch = useAuthenticatedFetch();
  const [pageData, setPageData] = useState<VocabData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [audioURLs, setAudioURLs] = useState<Record<string, string>>({});
  const [selectedAnswers, setSelectedAnswers] = useState<{
    [key: string]: string;
  }>({});
  const [lineResults, setLineResults] = useState<{
    [key: string]: boolean | null;
  }>({});
  const [pendingAnswers, setPendingAnswers] = useState<Set<string>>(new Set());
  const [lockedAnswers, setLockedAnswers] = useState<Set<string>>(new Set());
  const [completedLines, setCompletedLines] = useState<Set<number>>(new Set());
  const [originalLines, setOriginalLines] = useState<Record<number, string>>(
    {},
  );
  const [playedLines, setPlayedLines] = useState<Set<number>>(new Set());
  const [checkingLines, setCheckingLines] = useState<Set<number>>(new Set());
  const [currentLineIndex, setCurrentLineIndex] = useState(0);
  const [isPlaying, setIsPlaying] = useState(false);
  const [prefetchedAudio, setPrefetchedAudio] = useState<
    Record<string, HTMLAudioElement>
  >({});
  const [nextStepName, setNextStepName] = useState<string>("Next Step");

  const fetchAudioURLs = async () => {
    if (!id) return {};
    try {
      const response = await authenticatedFetch(
        `/api/stories/${id}/audio/signed?label=complete`,
      );
      if (!response.ok) return {};
      const data: AudioURLsResponse = await response.json();
      return data.success ? data.data : {};
    } catch (e) {
      console.error(`Failed to fetch audio URLs:`, e);
      return {};
    }
  };

  useEffect(() => {
    const fetchPageData = async () => {
      if (!id) {
        setError("Story ID is required");
        setLoading(false);
        return;
      }

      try {
        const [vocabResponse] = await Promise.all([api.getStoryVocab(id)]);

        if (vocabResponse.success && vocabResponse.data) {
          setPageData(vocabResponse.data);
        } else {
          setError(vocabResponse.error || "Failed to fetch page data");
        }

        // Fetch signed audio URLs
        const urls = await fetchAudioURLs();
        setAudioURLs(urls);
      } catch (err) {
        setError("Failed to fetch page data");
      } finally {
        setLoading(false);
      }
    };

    fetchPageData();
  }, [id]);

  useEffect(() => {
    const fetchNextStep = async () => {
      if (!id) return;
      try {
        const guidance = await getNavigationGuidance(id, "vocab");
        if (guidance) {
          setNextStepName(guidance.displayName);
        }
      } catch (error) {
        console.error("Failed to get navigation guidance:", error);
      }
    };

    fetchNextStep();
  }, [id, getNavigationGuidance]);

  const handleAnswerChange = async (vocabKey: string, value: string) => {
    // Lock this answer immediately
    setLockedAnswers((prev) => new Set([...prev, vocabKey]));
    setPendingAnswers((prev) => new Set([...prev, vocabKey]));

    setSelectedAnswers((prev) => ({
      ...prev,
      [vocabKey]: value,
    }));

    // Reset result for this vocab item
    setLineResults((prev) => ({
      ...prev,
      [vocabKey]: null,
    }));

    // Check this individual answer
    await checkIndividualAnswer(vocabKey, value);
  };

  const checkIndividualAnswer = async (vocabKey: string, value: string) => {
    if (!id || !value) return;

    try {
      // Send individual answer to API
      const result = await api.checkIndividualVocab(id, vocabKey, value);
      const isCorrect = result.success && result.data?.correct;
      const lineComplete = result.success && result.data?.line_complete;
      const originalLine = result.data?.original_line;

      // Update the result for this vocab item
      setLineResults((prev) => ({
        ...prev,
        [vocabKey]: isCorrect || false,
      }));

      if (!isCorrect) {
        // Unlock if incorrect
        setLockedAnswers((prev) => {
          const newSet = new Set(prev);
          newSet.delete(vocabKey);
          return newSet;
        });
      }

      // Handle line completion from API response
      if (lineComplete && originalLine) {
        const lineIndex = parseInt(vocabKey.split("-")[0]);
        setCompletedLines((prev) => new Set([...prev, lineIndex]));
        setOriginalLines((prev) => {
          const updated = {
            ...prev,
            [lineIndex]: originalLine,
          };
          return updated;
        });

        // Play next line audio
        setTimeout(() => {
          audioPlayer.playNextLineFromIndex(lineIndex);
        }, 1000);
      }
    } catch (err) {
      console.error("Failed to check answer:", err);
      // Unlock on error
      setLockedAnswers((prev) => {
        const newSet = new Set(prev);
        newSet.delete(vocabKey);
        return newSet;
      });
    } finally {
      setPendingAnswers((prev) => {
        const newSet = new Set(prev);
        newSet.delete(vocabKey);
        return newSet;
      });
    }
  };

  const allVocabCompleted = () => {
    if (!pageData) return false;
    return pageData.lines.every((line, index) => {
      return !lineHasVocab(line) || completedLines.has(index);
    });
  };

  const handleContinue = async () => {
    try {
      const guidance = await getNavigationGuidance(id!, "vocab");
      if (guidance) {
        navigate(`/stories/${id}/${guidance.nextPage}`);
      }
    } catch (error) {
      console.error("Failed to get navigation guidance:", error);
    }
  };

  const audioPlayer = useAudioPlayer({
    audioURLs,
    pageData,
    onPlayedLinesChange: setPlayedLines,
    onCurrentLineChange: setCurrentLineIndex,
    onPlayingStateChange: setIsPlaying,
    completedLines,
  });

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
  const languageCode = pageData.language;
  const isRTL = languageCode && RTL_LANGUAGES.includes(languageCode);

  return (
    <>
      <StoryHeader
        storyTitle={pageData.story_title}
        isPlaying={isPlaying}
        onPlayStoryAudio={audioPlayer.playStoryAudio}
      />
      <div className="max-w-4xl mx-auto px-5">
        <div className="story-lines text-2xl max-w-3xl mx-auto">
          {pageData.lines.length > 0 && (
            <div
              className={isRTL ? "text-right" : "text-left"}
              dir={isRTL ? "rtl" : "ltr"}
            >
              {pageData.lines.map((line, lineIndex) => (
                <StoryLine
                  key={lineIndex}
                  line={line}
                  lineIndex={lineIndex}
                  vocabBank={pageData.vocab_bank}
                  selectedAnswers={selectedAnswers}
                  lineResults={lineResults}
                  completedLines={completedLines}
                  playedLines={playedLines}
                  checkingLines={checkingLines}
                  isCurrentLine={currentLineIndex === lineIndex && isPlaying}
                  isRTL={!!isRTL}
                  prefetchedAudio={audioPlayer.prefetchedAudio}
                  originalLine={originalLines[lineIndex]}
                  onAnswerChange={handleAnswerChange}
                  pendingAnswers={pendingAnswers}
                  lockedAnswers={lockedAnswers}
                  onPlayLineAudio={audioPlayer.playLineAudio}
                />
              ))}
            </div>
          )}
        </div>

        {allVocabCompleted() && (
          <CompletionMessage
          currentStepName="vocabulary"
            nextStepName={nextStepName}
            onContinue={handleContinue}
          />
        )}
      </div>
    </>
  );
}
