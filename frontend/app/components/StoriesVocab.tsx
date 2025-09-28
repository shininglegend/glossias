import React, { useState, useEffect, useRef } from "react";
import { useParams, Link } from "react-router";
import { useApiService } from "../services/api";
import { useAuthenticatedFetch } from "../lib/authFetch";
import type { VocabData } from "../services/api";
import "./StoriesVocab.css";

interface AudioURLsResponse {
  success: boolean;
  data: Record<string, string>; // lineNumber -> signedURL
}

export function StoriesVocab() {
  const { id } = useParams<{ id: string }>();
  const api = useApiService();
  const authenticatedFetch = useAuthenticatedFetch();
  const [pageData, setPageData] = useState<VocabData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [audioURLs, setAudioURLs] = useState<Record<string, string>>({});
  const [currentAudio, setCurrentAudio] = useState<HTMLAudioElement | null>(
    null,
  );
  const [isPlaying, setIsPlaying] = useState(false);
  const [currentLineIndex, setCurrentLineIndex] = useState(0);
  const [selectedAnswers, setSelectedAnswers] = useState<{
    [key: number]: string;
  }>({});
  const [lineResults, setLineResults] = useState<{
    [key: number]: boolean | null;
  }>({});
  const [completedLines, setCompletedLines] = useState<Set<number>>(new Set());
  const [playedLines, setPlayedLines] = useState<Set<number>>(new Set());
  const [metadata, setMetadata] = useState<any>(null);
  const [prefetchedAudio, setPrefetchedAudio] = useState<
    Record<string, HTMLAudioElement>
  >({});
  const audioRef = useRef<HTMLAudioElement | null>(null);

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

  const prefetchAudio = async (urls: Record<string, string>) => {
    const audioCache: Record<string, HTMLAudioElement> = {};

    // Prefetch all audio files
    const prefetchPromises = Object.entries(urls).map(([lineNumber, url]) => {
      return new Promise<void>((resolve) => {
        const audio = new Audio(url);
        audio.preload = "auto";

        const onCanPlayThrough = () => {
          audioCache[lineNumber] = audio;
          audio.removeEventListener("canplaythrough", onCanPlayThrough);
          audio.removeEventListener("error", onError);
          resolve();
        };

        const onError = () => {
          console.warn(`Failed to prefetch audio for line ${lineNumber}`);
          audio.removeEventListener("canplaythrough", onCanPlayThrough);
          audio.removeEventListener("error", onError);
          resolve();
        };

        audio.addEventListener("canplaythrough", onCanPlayThrough);
        audio.addEventListener("error", onError);

        // Start loading
        audio.load();
      });
    });

    await Promise.all(prefetchPromises);
    setPrefetchedAudio(audioCache);
  };

  useEffect(() => {
    const fetchPageData = async () => {
      if (!id) {
        setError("Story ID is required");
        setLoading(false);
        return;
      }

      try {
        const [vocabResponse, metadataResponse] = await Promise.all([
          api.getStoryVocab(id),
          api.getStoryMetadata(id),
        ]);

        if (vocabResponse.success && vocabResponse.data) {
          setPageData(vocabResponse.data);
        } else {
          setError(vocabResponse.error || "Failed to fetch page data");
        }

        if (metadataResponse.success && metadataResponse.data) {
          setMetadata(metadataResponse.data);
        }

        // Fetch signed audio URLs
        const urls = await fetchAudioURLs();
        setAudioURLs(urls);

        // Prefetch all audio files
        if (Object.keys(urls).length > 0) {
          prefetchAudio(urls);
        }
      } catch (err) {
        setError("Failed to fetch page data");
      } finally {
        setLoading(false);
      }
    };

    fetchPageData();
  }, [id]);

  const stopAudio = () => {
    if (currentAudio) {
      currentAudio.pause();
      currentAudio.currentTime = 0;
    }
    setCurrentAudio(null);
    setIsPlaying(false);
    setCurrentLineIndex(0);
  };

  const playLineAudio = (lineIndex: number) => {
    const lineKey = (lineIndex + 1).toString();
    const audio = prefetchedAudio[lineKey];
    if (!audio) return;

    stopAudio();

    // Reset audio to beginning
    audio.currentTime = 0;
    setCurrentAudio(audio);
    setIsPlaying(true);
    setCurrentLineIndex(lineIndex);

    audio.play().catch((err) => {
      console.error("Failed to play audio:", err);
      setIsPlaying(false);
    });

    const onEnded = () => {
      setCurrentAudio(null);
      setIsPlaying(false);
      setPlayedLines((prev) => new Set([...prev, lineIndex]));
      audio.removeEventListener("ended", onEnded);
    };

    audio.addEventListener("ended", onEnded);
  };

  const playStoryAudio = () => {
    if (!pageData) return;

    if (isPlaying) {
      stopAudio();
      return;
    }

    setIsPlaying(true);
    playNextLineFromIndex(currentLineIndex);
  };

  const playNextLineFromIndex = (startIndex: number) => {
    if (!pageData || startIndex >= pageData.lines.length) {
      setIsPlaying(false);
      setCurrentLineIndex(0);
      return;
    }

    const line = pageData.lines[startIndex];
    setCurrentLineIndex(startIndex);

    // Play the line audio if available
    const lineKey = (startIndex + 1).toString();
    const audio = prefetchedAudio[lineKey];
    if (audio) {
      // Reset audio to beginning
      audio.currentTime = 0;
      setCurrentAudio(audio);

      audio.play().catch((err) => {
        console.error("Failed to play audio:", err);
        playNextLineFromIndex(startIndex + 1);
      });

      const onEnded = () => {
        setCurrentAudio(null);
        setPlayedLines((prev) => new Set([...prev, startIndex]));
        audio.removeEventListener("ended", onEnded);

        // If this line has vocab and isn't completed, stop here
        if (line.has_vocab_or_grammar && !completedLines.has(startIndex)) {
          setIsPlaying(false);
          return;
        }

        playNextLineFromIndex(startIndex + 1);
      };

      audio.addEventListener("ended", onEnded);
    } else {
      // Mark as played even without audio
      setPlayedLines((prev) => new Set([...prev, startIndex]));

      // If this line has vocab and isn't completed, stop here
      if (line.has_vocab_or_grammar && !completedLines.has(startIndex)) {
        setIsPlaying(false);
        return;
      }
      playNextLineFromIndex(startIndex + 1);
    }
  };

  const handleAnswerChange = (lineIndex: number, value: string) => {
    setSelectedAnswers((prev) => ({
      ...prev,
      [lineIndex]: value,
    }));
    // Reset result for this line
    setLineResults((prev) => ({
      ...prev,
      [lineIndex]: null,
    }));
  };

  const checkLineAnswer = async (lineIndex: number) => {
    if (!id || !selectedAnswers[lineIndex]) return;

    try {
      const response = await api.checkVocabLine(
        id,
        lineIndex,
        selectedAnswers[lineIndex],
      );
      if (response.success && response.data) {
        const isCorrect = response.data.correct;
        setLineResults((prev) => ({
          ...prev,
          [lineIndex]: isCorrect,
        }));

        if (isCorrect) {
          setCompletedLines((prev) => new Set([...prev, lineIndex]));
          // Continue playing audio from next line
          setTimeout(() => {
            setCurrentLineIndex(lineIndex + 1);
            setIsPlaying(true);
            playNextLineFromIndex(lineIndex + 1);
          }, 1000);
        }
      }
    } catch (err) {
      console.error("Failed to check answer:", err);
    }
  };

  const allVocabCompleted = () => {
    if (!pageData) return false;
    return pageData.lines.every((line, index) => {
      return !line.has_vocab_or_grammar || completedLines.has(index);
    });
  };

  useEffect(() => {
    return () => {
      stopAudio();
    };
  }, []);

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
    <>
      <header>
        <h1>{pageData.story_title}</h1>
        <h2>Step 2: Vocabulary Practice</h2>
        <p>Fill in the blanks with the correct vocabulary words:</p>
        <button
          onClick={playStoryAudio}
          className={`inline-flex items-center gap-2 px-5 py-3 my-5 text-white border-none rounded-lg text-base cursor-pointer transition-colors duration-200 ${
            isPlaying
              ? "bg-red-500 hover:bg-red-600"
              : "bg-blue-500 hover:bg-blue-600"
          }`}
          type="button"
        >
          <span className="material-icons">
            {isPlaying ? "pause" : "play_arrow"}
          </span>
          {isPlaying ? "Pause Audio" : "Play Audio"}
        </button>
      </header>
      <div className="max-w-4xl mx-auto px-5">
        <div className="story-lines text-2xl max-w-3xl mx-auto">
          {pageData.lines.length > 0 &&
            (() => {
              const RTL_LANGUAGES = ["he", "ar", "fa", "ur"];
              const languageCode = metadata?.description?.language;
              const isRTL =
                languageCode && RTL_LANGUAGES.includes(languageCode);

              return (
                <div
                  className={isRTL ? "text-right" : "text-left"}
                  dir={isRTL ? "rtl" : "ltr"}
                >
                  {pageData.lines.map((line, lineIndex) => (
                    <div
                      key={lineIndex}
                      className={`story-line inline ${line.has_vocab_or_grammar ? "has-vocab" : ""} ${
                        currentLineIndex === lineIndex && isPlaying
                          ? "bg-yellow-100 px-1 py-0.5 rounded"
                          : ""
                      }`}
                    >
                      <div className="line-content text-3xl inline">
                        {line.text.map((text, textIndex) => {
                          if (text === "%") {
                            const result = lineResults[lineIndex];
                            const isDisabled =
                              completedLines.has(lineIndex) ||
                              !playedLines.has(lineIndex);
                            return (
                              <span
                                key={textIndex}
                                className="vocab-container inline-block mx-1"
                              >
                                <select
                                  className={`vocab-select inline-block min-w-24 px-2 py-1 text-2xl border-2 rounded cursor-pointer bg-white transition-all duration-200 focus:outline-none focus:border-blue-500 ${
                                    result === true
                                      ? "border-green-500 bg-green-50"
                                      : result === false
                                        ? "border-red-500 bg-red-50"
                                        : !playedLines.has(lineIndex)
                                          ? "border-gray-200 bg-gray-50"
                                          : "border-gray-300"
                                  } ${
                                    isDisabled
                                      ? "opacity-60 cursor-not-allowed"
                                      : ""
                                  }`}
                                  value={selectedAnswers[lineIndex] || ""}
                                  onChange={(e) =>
                                    handleAnswerChange(
                                      lineIndex,
                                      e.target.value,
                                    )
                                  }
                                  disabled={isDisabled}
                                  title={
                                    !playedLines.has(lineIndex)
                                      ? "Play the audio first to unlock this vocabulary"
                                      : ""
                                  }
                                >
                                  <option value="">
                                    {!playedLines.has(lineIndex) ? "-" : "___"}
                                  </option>
                                  {pageData.vocab_bank.map(
                                    (word, wordIndex) => (
                                      <option
                                        key={wordIndex}
                                        value={word}
                                        disabled={!playedLines.has(lineIndex)}
                                      >
                                        {word}
                                      </option>
                                    ),
                                  )}
                                </select>
                                {selectedAnswers[lineIndex] &&
                                  !completedLines.has(lineIndex) &&
                                  playedLines.has(lineIndex) && (
                                    <button
                                      onClick={() => checkLineAnswer(lineIndex)}
                                      className="check-button w-6 h-6 bg-blue-500 text-white border-none rounded-full cursor-pointer text-sm flex items-center justify-center transition-colors duration-200 hover:bg-blue-600"
                                      type="button"
                                    >
                                      ✓
                                    </button>
                                  )}
                                {result === false && (
                                  <span className="error-indicator text-red-500 text-lg font-bold">
                                    ✗
                                  </span>
                                )}
                                {completedLines.has(lineIndex) && (
                                  <span className="success-indicator text-green-500 text-lg font-bold">
                                    ✓
                                  </span>
                                )}
                              </span>
                            );
                          } else {
                            return <span key={textIndex}>{text}</span>;
                          }
                        })}
                      </div>
                      {line.has_vocab_or_grammar &&
                        !completedLines.has(lineIndex) &&
                        prefetchedAudio[(lineIndex + 1).toString()] &&
                        (playedLines.has(lineIndex) ||
                          currentLineIndex === lineIndex) && (
                          <button
                            onClick={() => playLineAudio(lineIndex)}
                            className="inline-flex items-center justify-center w-8 h-8 bg-gray-500 text-white border-none rounded-full cursor-pointer ml-3 transition-colors duration-200 hover:bg-gray-600 align-middle"
                            type="button"
                          >
                            <span className="material-icons text-lg">
                              play_arrow
                            </span>
                          </button>
                        )}
                    </div>
                  ))}
                </div>
              );
            })()}
        </div>

        {allVocabCompleted() && (
          <div className="text-center mt-10 p-8 bg-green-50 rounded-xl border-2 border-green-500">
            <div className="mb-5">
              <h3 className="text-green-700 m-0 text-2xl">
                Great job! You've completed all vocabulary exercises.
              </h3>
            </div>
            <div className="mt-5">
              <Link
                to={`/stories/${id}/translate`}
                className="next-button inline-flex items-center gap-2 px-8 py-4 bg-green-500 text-white no-underline rounded-lg text-lg font-semibold transition-all duration-200 shadow-lg hover:bg-green-600"
              >
                <span>Continue to Translation</span>
                <span className="material-icons">arrow_forward</span>
              </Link>
            </div>
          </div>
        )}
      </div>
    </>
  );
}
