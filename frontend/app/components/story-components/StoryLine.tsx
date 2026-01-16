import React from "react";
import { VocabTextRenderer } from "./VocabTextRenderer";
import type { VocabLine } from "../../services/api";

interface StoryLineProps {
  line: VocabLine;
  lineIndex: number;
  vocabBank: string[];
  selectedAnswers: { [key: string]: string };
  lineResults: { [key: string]: boolean | null };
  completedLines: Set<number>;
  playedLines: Set<number>;
  checkingLines: Set<number>;
  isCurrentLine: boolean;
  isRTL: boolean;
  prefetchedAudio: Record<string, HTMLAudioElement>;
  originalLine?: string;
  pendingAnswers: Set<string>;
  lockedAnswers: Set<string>;
  onAnswerChange: (vocabKey: string, value: string) => void;
  onPlayLineAudio: (lineIndex: number) => void;
  translation?: string;
  showTranslation?: boolean;
}

// Helper function to check if a line contains vocabulary placeholders
const lineHasVocab = (line: VocabLine): boolean => {
  return line.text.some((segment) => segment.type === "blank");
};

export const StoryLine: React.FC<StoryLineProps> = ({
  line,
  lineIndex,
  vocabBank,
  selectedAnswers,
  lineResults,
  completedLines,
  playedLines,
  checkingLines,
  isCurrentLine,
  isRTL,
  prefetchedAudio,
  originalLine,
  pendingAnswers,
  lockedAnswers,
  onAnswerChange,
  onPlayLineAudio,
  translation,
  showTranslation,
}) => {
  const hasVocab = lineHasVocab(line);
  const hasAudio = prefetchedAudio[(lineIndex + 1).toString()];
  const shouldShowPlayButton =
    hasVocab &&
    !completedLines.has(lineIndex) &&
    hasAudio &&
    (playedLines.has(lineIndex) || isCurrentLine);

  return (
    <div
      className={`story-line inline ${hasVocab ? "has-vocab" : ""} ${
        isCurrentLine ? "bg-yellow-100 px-1 py-0.5 rounded" : ""
      }`}
    >
      <VocabTextRenderer
        line={line}
        lineIndex={lineIndex}
        vocabBank={vocabBank}
        selectedAnswers={selectedAnswers}
        lineResults={lineResults}
        completedLines={completedLines}
        playedLines={playedLines}
        checkingLines={checkingLines}
        isCurrentLine={isCurrentLine}
        isRTL={isRTL}
        originalLine={originalLine}
        pendingAnswers={pendingAnswers}
        lockedAnswers={lockedAnswers}
        onAnswerChange={onAnswerChange}
      />
      {shouldShowPlayButton && (
        <button
          onClick={() => onPlayLineAudio(lineIndex)}
          className="inline-flex items-center justify-center w-8 h-8 bg-gray-500 text-white border-none rounded-full cursor-pointer ml-3 transition-colors duration-200 hover:bg-gray-600 align-middle"
          type="button"
        >
          <span className="material-icons text-lg">play_arrow</span>
        </button>
      )}
      {showTranslation && translation && (
        <div
          className="mt-2 mb-4 bg-blue-50 border-l-4 border-blue-400 p-3 rounded-r-lg text-left block"
          dir="ltr"
        >
          <p className="text-lg text-blue-900">{translation}</p>
        </div>
      )}
    </div>
  );
};
