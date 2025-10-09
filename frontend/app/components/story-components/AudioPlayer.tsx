import { useState, useEffect } from "react";
import type { VocabData, VocabLine } from "../../services/api";

interface UseAudioPlayerProps {
  audioURLs: Record<string, string>;
  pageData: VocabData | null;
  onPlayedLinesChange: (lines: Set<number>) => void;
  onCurrentLineChange: (index: number) => void;
  onPlayingStateChange: (isPlaying: boolean) => void;
  completedLines: Set<number>;
}

// Helper function to check if a line contains vocabulary placeholders
const lineHasVocab = (line: VocabLine): boolean => {
  return line.text.some((segment) => segment.type === "blank");
};

export const useAudioPlayer = ({
  audioURLs,
  pageData,
  onPlayedLinesChange,
  onCurrentLineChange,
  onPlayingStateChange,
  completedLines,
}: UseAudioPlayerProps) => {
  const [currentAudio, setCurrentAudio] = useState<HTMLAudioElement | null>(
    null,
  );
  const [isPlaying, setIsPlaying] = useState(false);
  const [currentLineIndex, setCurrentLineIndex] = useState(0);
  const [playedLines, setPlayedLines] = useState<Set<number>>(new Set());
  const [prefetchedAudio, setPrefetchedAudio] = useState<
    Record<string, HTMLAudioElement>
  >({});

  const prefetchAudio = async (urls: Record<string, string>) => {
    const audioCache: Record<string, HTMLAudioElement> = {};

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
        audio.load();
      });
    });

    await Promise.all(prefetchPromises);
    setPrefetchedAudio(audioCache);
  };

  useEffect(() => {
    if (Object.keys(audioURLs).length > 0) {
      prefetchAudio(audioURLs);
    }
  }, [audioURLs]);

  useEffect(() => {
    onPlayedLinesChange(playedLines);
  }, [playedLines, onPlayedLinesChange]);

  useEffect(() => {
    onCurrentLineChange(currentLineIndex);
  }, [currentLineIndex, onCurrentLineChange]);

  useEffect(() => {
    onPlayingStateChange(isPlaying);
  }, [isPlaying, onPlayingStateChange]);

  const stopAudio = () => {
    if (currentAudio) {
      currentAudio.pause();
      currentAudio.currentTime = 0;
    }
    setCurrentAudio(null);
    setIsPlaying(false);
    setCurrentLineIndex(0);
  };

  const pauseAudio = () => {
    if (currentAudio) {
      currentAudio.pause();
    }
    setCurrentAudio(null);
    setIsPlaying(false);
  };

  const playLineAudio = (lineIndex: number) => {
    const lineKey = (lineIndex + 1).toString();
    const audio = prefetchedAudio[lineKey];
    if (!audio) return;

    stopAudio();

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
      pauseAudio();
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

    const lineKey = (startIndex + 1).toString();
    const audio = prefetchedAudio[lineKey];
    if (audio) {
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

        if (lineHasVocab(line) && !completedLines.has(startIndex)) {
          setIsPlaying(false);
          return;
        }

        playNextLineFromIndex(startIndex + 1);
      };

      audio.addEventListener("ended", onEnded);
    } else {
      setPlayedLines((prev) => new Set([...prev, startIndex]));

      if (lineHasVocab(line) && !completedLines.has(startIndex)) {
        setIsPlaying(false);
        return;
      }
      playNextLineFromIndex(startIndex + 1);
    }
  };

  const playNextLineFromIndexContinuation = (index: number) => {
    setCurrentLineIndex(index + 1);
    setIsPlaying(true);
    playNextLineFromIndex(index + 1);
  };

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      stopAudio();
    };
  }, []);

  return {
    isPlaying,
    currentLineIndex,
    playedLines,
    prefetchedAudio,
    playLineAudio,
    playStoryAudio,
    playNextLineFromIndex: playNextLineFromIndexContinuation,
  };
};
