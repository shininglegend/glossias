import React from "react";
import type { VocabLine } from "../../services/api";

interface VocabTextRendererProps {
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
  originalLine?: string;
  pendingAnswers: Set<string>;
  lockedAnswers: Set<string>;
  onAnswerChange: (vocabKey: string, value: string) => void;
}

// Helper function for RTL indentation
const processTextForRTL = (text: string, isRTL: boolean) => {
  if (!isRTL || typeof text !== "string") {
    return { displayText: text, indentLevel: 0 };
  }

  const leadingTabs = text.match(/^\t*/)?.[0] || "";
  const tabCount = leadingTabs.length;
  const textWithoutTabs = text.slice(tabCount);

  return {
    displayText: textWithoutTabs,
    indentLevel: tabCount,
  };
};

export const VocabTextRenderer: React.FC<VocabTextRendererProps> = ({
  line,
  lineIndex,
  vocabBank,
  selectedAnswers,
  lineResults,
  completedLines,
  playedLines,
  isRTL,
  originalLine,
  pendingAnswers,
  lockedAnswers,
  onAnswerChange,
}) => {
  const isDisabled =
    completedLines.has(lineIndex) || !playedLines.has(lineIndex);

  // If line is completed and we have the original text, display it
  if (completedLines.has(lineIndex) && originalLine) {
    const { displayText, indentLevel } = processTextForRTL(originalLine, isRTL);
    return (
      <div className="line-content text-3xl inline">
        <span
          className={isRTL ? `rtl-indent-${indentLevel}` : ""}
          style={isRTL ? {} : { marginLeft: `${indentLevel * 1.5}rem` }}
        >
          {displayText}
        </span>
      </div>
    );
  }

  return (
    <div className="line-content text-3xl inline">
      {line.text.map((segment, segmentIndex) => {
        const { displayText, indentLevel } = processTextForRTL(
          segment.text,
          isRTL,
        );

        if (segment.type === "blank" && segment.vocab_key) {
          const vocabKey = segment.vocab_key;
          const result = lineResults[vocabKey];
          const isPending = pendingAnswers.has(vocabKey);
          const isLocked = lockedAnswers.has(vocabKey);

          return (
            <span
              key={segmentIndex}
              className="vocab-container inline-block mx-1"
            >
              <select
                className={`vocab-select inline-block min-w-24 px-2 py-1 text-2xl border-2 rounded cursor-pointer bg-white transition-all duration-200 focus:outline-none focus:border-primary-500 ${
                  result === true
                    ? "border-green-500 bg-green-50"
                    : result === false
                      ? "border-red-500 bg-red-50"
                      : !playedLines.has(lineIndex)
                        ? "border-gray-200 bg-gray-50"
                        : "border-gray-300"
                } ${isDisabled || isLocked ? "opacity-60 cursor-not-allowed" : ""}`}
                value={selectedAnswers[vocabKey] || ""}
                onChange={(e) => onAnswerChange(vocabKey, e.target.value)}
                disabled={isDisabled || isLocked}
                title={
                  !playedLines.has(lineIndex)
                    ? "Play the audio first to unlock this vocabulary"
                    : isLocked
                      ? "Answer is being checked"
                      : ""
                }
              >
                <option value="">
                  {!playedLines.has(lineIndex) ? "---" : "choose"}
                </option>
                {vocabBank.map((word, wordIndex) => (
                  <option
                    key={wordIndex}
                    value={word}
                    disabled={!playedLines.has(lineIndex)}
                  >
                    {word}
                  </option>
                ))}
              </select>

              {isPending && (
                <span className="loading-indicator ml-1 inline-flex items-center">
                  <div className="animate-spin w-4 h-4 border-2 border-primary-500 border-t-transparent rounded-full"></div>
                </span>
              )}
              {result === false && !isPending && (
                <span className="error-indicator text-red-500 text-lg font-bold ml-1">
                  ✗
                </span>
              )}
              {result === true && !isPending && (
                <span className="success-indicator text-green-500 text-lg font-bold ml-1">
                  ✓
                </span>
              )}
            </span>
          );
        } else if (segment.type === "completed") {
          // For completed vocab, show locked dropdown with green styling and checkmark
          return (
            <span
              key={segmentIndex}
              className="vocab-container inline-block mx-1"
            >
              <select
                className="vocab-select inline-block min-w-24 px-2 py-1 text-2xl border-2 rounded cursor-not-allowed bg-green-50 border-green-500 opacity-60"
                value={displayText}
                disabled={true}
              >
                <option value={displayText}>{displayText}</option>
              </select>
              <span className="success-indicator text-green-500 text-lg font-bold ml-1">
                ✓
              </span>
            </span>
          );
        } else {
          // For "text" segments, just display the text
          return (
            <span
              key={segmentIndex}
              style={
                indentLevel > 0 ? { paddingRight: `${indentLevel * 2}em` } : {}
              }
            >
              {displayText}
            </span>
          );
        }
      })}
    </div>
  );
};
