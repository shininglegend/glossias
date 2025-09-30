import { useState, useEffect } from "react";
import { useParams, useNavigate, Link } from "react-router";
import { useApiService } from "../services/api";
import { useNavigationGuidance } from "../hooks/useNavigationGuidance";
import { useAuthenticatedFetch } from "../lib/authFetch";
import type { VocabData } from "../services/api";
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
const lineHasVocab = (line: { text: string[] }): boolean => {
  return line.text.includes("%");
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
  const [completedLines, setCompletedLines] = useState<Set<number>>(new Set());
  const [originalLines, setOriginalLines] = useState<Record<number, string>>(
    {}
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
        `/api/stories/${id}/audio/signed?label=complete`
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

  const handleAnswerChange = (vocabKey: string, value: string) => {
    setSelectedAnswers((prev) => ({
      ...prev,
      [vocabKey]: value,
    }));
    // Reset result for this vocab item
    setLineResults((prev) => ({
      ...prev,
      [vocabKey]: null,
    }));
  };

  const checkLineAnswer = async (vocabKey: string) => {
    if (!id || !selectedAnswers[vocabKey]) return;

    // Extract lineIndex from vocabKey (format: "lineIndex-vocabIndex")
    const lineIndex = parseInt(vocabKey.split("-")[0]);

    // Check if all vocab items on this line have answers
    const lineVocabKeys = Object.keys(selectedAnswers)
      .filter((key) => key.startsWith(`${lineIndex}-`))
      .sort(); // Ensure consistent order by vocab index

    const totalVocabOnLine =
      pageData?.lines[lineIndex]?.text.filter((t) => t === "%").length || 0;

    // Only proceed if we have answers for all vocab items on this line
    const answersForLine = lineVocabKeys
      .map((key) => selectedAnswers[key])
      .filter(Boolean);

    if (answersForLine.length !== totalVocabOnLine) {
      // Not all vocab items filled yet, just return
      return;
    }

    setCheckingLines((prev) => new Set([...prev, lineIndex]));

    try {
      // Send all answers for this line to the API
      const response = await api.checkVocabLine(id, lineIndex, answersForLine);

      if (response.success && response.data) {
        const individualResults = response.data.results || [];
        const allCorrect = response.data.allCorrect;

        // Update results for each vocab item individually
        const newResults: { [key: string]: boolean | null } = {};
        lineVocabKeys.forEach((key, index) => {
          newResults[key] = individualResults[index] ?? false;
        });

        setLineResults((prev) => ({
          ...prev,
          ...newResults,
        }));

        if (allCorrect) {
          setCompletedLines((prev) => new Set([...prev, lineIndex]));
          // Store original line if provided
          if (response.data?.originalLine) {
            setOriginalLines((prev) => ({
              ...prev,
              [lineIndex]: response.data!.originalLine!,
            }));
          }
          // Continue playing audio from next line
          setTimeout(() => {
            audioPlayer.playNextLineFromIndex(lineIndex);
          }, 1000);
        }
      }
    } catch (err) {
      console.error("Failed to check answer:", err);
    } finally {
      setCheckingLines((prev) => {
        const newSet = new Set(prev);
        newSet.delete(lineIndex);
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
                  onCheckAnswer={checkLineAnswer}
                  onPlayLineAudio={audioPlayer.playLineAudio}
                />
              ))}
            </div>
          )}
        </div>

        {allVocabCompleted() && (
          <CompletionMessage
            nextStepName={nextStepName}
            onContinue={handleContinue}
          />
        )}
      </div>
    </>
  );
}
