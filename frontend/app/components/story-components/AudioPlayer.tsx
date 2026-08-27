import { useState, useEffect, useRef, useCallback } from "react";
import type { VocabData, VocabLine } from "../../services/api";

interface UseAudioPlayerProps {
  audioURLs: Record<string, string>;
  pageData: VocabData | null;
  onPlayedLinesChange: (lines: Set<number>) => void;
  onCurrentLineChange: (index: number) => void;
  onPlayingStateChange: (isPlaying: boolean) => void;
  completedLines: Set<number>;
  /** Pause after every line that is not yet completed. */
  pauseAfterEveryLine?: boolean;
  /**
   * Pause only after these line indices (0-based) when they are not yet
   * completed. Takes precedence over `pauseAfterEveryLine` and the default
   * "pause after lines with vocab blanks" heuristic.
   */
  pauseOnLines?: Set<number>;
  /**
   * Line indices (0-based) to skip during sequential playback. Skipped lines
   * are not played and are not added to `playedLines`.
   */
  skipLines?: Set<number>;
  /**
   * Called once when sequential playback runs past the last line (after
   * `isPlaying` is cleared and the line index resets to 0). Not called by
   * `stopAudio`, `pauseAudio`, or single-line playback.
   */
  onPlaybackEnd?: () => void;
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
  pauseAfterEveryLine = false,
  pauseOnLines,
  skipLines,
  onPlaybackEnd,
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
  const [isResumeScheduled, setIsResumeScheduled] = useState(false);

  // Latest values, readable from `ended` handlers registered on earlier renders.
  const currentAudioRef = useRef<HTMLAudioElement | null>(null);
  const currentLineIndexRef = useRef(0);
  const prefetchedAudioRef = useRef<Record<string, HTMLAudioElement>>({});
  const optionsRef = useRef({
    pageData,
    completedLines,
    pauseAfterEveryLine,
    pauseOnLines,
    skipLines,
    onPlaybackEnd,
  });
  optionsRef.current = {
    pageData,
    completedLines,
    pauseAfterEveryLine,
    pauseOnLines,
    skipLines,
    onPlaybackEnd,
  };
  const resumeTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  // Detach handler for the most recently started playback, so a new play
  // call can cancel the previous line's `ended` listener.
  const detachEndedRef = useRef<(() => void) | null>(null);

  const setAudio = (audio: HTMLAudioElement | null) => {
    currentAudioRef.current = audio;
    setCurrentAudio(audio);
  };

  const setLineIndex = (index: number) => {
    currentLineIndexRef.current = index;
    setCurrentLineIndex(index);
  };

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
    prefetchedAudioRef.current = audioCache;
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

  const detachCurrentEnded = () => {
    if (detachEndedRef.current) {
      detachEndedRef.current();
      detachEndedRef.current = null;
    }
  };

  const clearResumeTimeout = () => {
    if (resumeTimeoutRef.current !== null) {
      clearTimeout(resumeTimeoutRef.current);
      resumeTimeoutRef.current = null;
    }
    setIsResumeScheduled(false);
  };

  const haltCurrentAudio = (resetTime: boolean) => {
    detachCurrentEnded();
    const audio = currentAudioRef.current;
    if (audio) {
      audio.pause();
      if (resetTime) audio.currentTime = 0;
    }
    setAudio(null);
    setIsPlaying(false);
  };

  const stopAudio = () => {
    clearResumeTimeout();
    haltCurrentAudio(true);
    setLineIndex(0);
  };

  const pauseAudio = () => {
    clearResumeTimeout();
    haltCurrentAudio(false);
  };

  /**
   * Decide whether sequential playback should pause after `lineIndex`.
   * Reads the latest options so `ended` handlers registered on an earlier
   * render still see up-to-date completed/pause sets.
   */
  const shouldPauseAfter = (lineIndex: number): boolean => {
    const { pageData, completedLines, pauseAfterEveryLine, pauseOnLines } =
      optionsRef.current;
    if (completedLines.has(lineIndex)) return false;
    if (pauseOnLines) return pauseOnLines.has(lineIndex);
    if (pauseAfterEveryLine) return true;
    const line = pageData?.lines[lineIndex];
    return line ? lineHasVocab(line) : false;
  };

  /**
   * Play a single line in isolation and stop. Marks the line as played and
   * sets it as the current line (existing behaviour, used by StoryLine).
   */
  const playLineAudio = (lineIndex: number) => {
    const lineKey = (lineIndex + 1).toString();
    const audio = prefetchedAudioRef.current[lineKey];
    if (!audio) return;

    stopAudio();

    audio.currentTime = 0;
    setAudio(audio);
    setIsPlaying(true);
    setLineIndex(lineIndex);

    const onEnded = () => {
      detachEndedRef.current = null;
      audio.removeEventListener("ended", onEnded);
      setAudio(null);
      setIsPlaying(false);
      setPlayedLines((prev) => new Set([...prev, lineIndex]));
    };

    audio.addEventListener("ended", onEnded);
    detachEndedRef.current = () => audio.removeEventListener("ended", onEnded);

    audio.play().catch((err) => {
      console.error("Failed to play audio:", err);
      setIsPlaying(false);
    });
  };

  /**
   * Replay a single line without touching sequential-playback state:
   * `currentLineIndex` and `playedLines` are left unchanged, so consumers
   * watching those (e.g. "line just finished" effects) are not re-triggered.
   * Any in-progress playback or scheduled resume is cancelled first.
   * Resolves when the replay ends, is interrupted, or fails to start
   * (immediately if no audio exists for the line).
   */
  const replayLine = (lineIndex: number): Promise<void> => {
    const lineKey = (lineIndex + 1).toString();
    const audio = prefetchedAudioRef.current[lineKey];
    if (!audio) return Promise.resolve();

    clearResumeTimeout();
    haltCurrentAudio(true);

    return new Promise<void>((resolve) => {
      audio.currentTime = 0;
      setAudio(audio);
      setIsPlaying(true);

      const onEnded = () => {
        detachEndedRef.current = null;
        audio.removeEventListener("ended", onEnded);
        setAudio(null);
        setIsPlaying(false);
        resolve();
      };

      audio.addEventListener("ended", onEnded);
      detachEndedRef.current = () => {
        audio.removeEventListener("ended", onEnded);
        resolve();
      };

      audio.play().catch((err) => {
        console.error("Failed to play audio:", err);
        detachEndedRef.current = null;
        audio.removeEventListener("ended", onEnded);
        setAudio(null);
        setIsPlaying(false);
        resolve();
      });
    });
  };

  const playNextLineFromIndex = (startIndex: number) => {
    const { pageData, skipLines, onPlaybackEnd } = optionsRef.current;
    if (!pageData || startIndex >= pageData.lines.length) {
      setIsPlaying(false);
      setLineIndex(0);
      onPlaybackEnd?.();
      return;
    }

    if (skipLines?.has(startIndex)) {
      playNextLineFromIndex(startIndex + 1);
      return;
    }

    setLineIndex(startIndex);

    const lineKey = (startIndex + 1).toString();
    const audio = prefetchedAudioRef.current[lineKey];
    if (audio) {
      audio.currentTime = 0;
      setAudio(audio);

      const onEnded = () => {
        detachEndedRef.current = null;
        audio.removeEventListener("ended", onEnded);
        setAudio(null);
        setPlayedLines((prev) => new Set([...prev, startIndex]));

        if (shouldPauseAfter(startIndex)) {
          setIsPlaying(false);
          return;
        }

        playNextLineFromIndex(startIndex + 1);
      };

      audio.addEventListener("ended", onEnded);
      detachEndedRef.current = () =>
        audio.removeEventListener("ended", onEnded);

      audio.play().catch((err) => {
        console.error("Failed to play audio:", err);
        detachEndedRef.current = null;
        audio.removeEventListener("ended", onEnded);
        playNextLineFromIndex(startIndex + 1);
      });
    } else {
      setPlayedLines((prev) => new Set([...prev, startIndex]));

      if (shouldPauseAfter(startIndex)) {
        setIsPlaying(false);
        return;
      }
      playNextLineFromIndex(startIndex + 1);
    }
  };

  const playStoryAudio = () => {
    if (!pageData) return;

    if (isPlaying) {
      pauseAudio();
      return;
    }

    clearResumeTimeout();
    setIsPlaying(true);
    playNextLineFromIndex(currentLineIndexRef.current);
  };

  const playNextLineFromIndexContinuation = (index: number) => {
    clearResumeTimeout();
    haltCurrentAudio(true);
    setLineIndex(index + 1);
    setIsPlaying(true);
    playNextLineFromIndex(index + 1);
  };

  /**
   * Resume sequential playback after `delayMs`. By default continues with the
   * line after the current one (same as `playNextLineFromIndex(current)`);
   * pass `fromIndex` to start at a specific line instead. Only one resume can
   * be pending — scheduling again replaces the earlier one. Cancelled by
   * `cancelScheduledResume`, `pauseAudio`, `stopAudio`, and any play call.
   */
  const scheduleResume = (delayMs: number, fromIndex?: number) => {
    clearResumeTimeout();
    setIsResumeScheduled(true);
    resumeTimeoutRef.current = setTimeout(() => {
      resumeTimeoutRef.current = null;
      setIsResumeScheduled(false);
      const start =
        fromIndex !== undefined ? fromIndex : currentLineIndexRef.current + 1;
      haltCurrentAudio(true);
      setLineIndex(start);
      setIsPlaying(true);
      playNextLineFromIndex(start);
    }, delayMs);
  };

  const cancelScheduledResume = useCallback(() => {
    if (resumeTimeoutRef.current !== null) {
      clearTimeout(resumeTimeoutRef.current);
      resumeTimeoutRef.current = null;
    }
    setIsResumeScheduled(false);
  }, []);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (resumeTimeoutRef.current !== null) {
        clearTimeout(resumeTimeoutRef.current);
      }
      detachEndedRef.current?.();
      const audio = currentAudioRef.current;
      if (audio) {
        audio.pause();
        audio.currentTime = 0;
      }
    };
  }, []);

  return {
    isPlaying,
    currentLineIndex,
    playedLines,
    prefetchedAudio,
    currentAudio,
    isResumeScheduled,
    playLineAudio,
    replayLine,
    playStoryAudio,
    playNextLineFromIndex: playNextLineFromIndexContinuation,
    pauseAudio,
    stopAudio,
    scheduleResume,
    cancelScheduledResume,
  };
};
