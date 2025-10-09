import React from "react";

interface VocabTextRendererProps {
  line: { text: string[] };
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
  let vocabIndex = 0; // Track vocab items within this line
  const isDisabled =
    completedLines.has(lineIndex) || !playedLines.has(lineIndex);

  // Check if all vocab items on this line have answers
  const totalVocabOnLine = line.text.filter((t) => t === "%").length;
  const lineVocabKeys = Array.from(
    { length: totalVocabOnLine },
    (_, i) => `${lineIndex}-${i}`,
  );

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
      {line.text.map((text, textIndex) => {
        const { displayText, indentLevel } = processTextForRTL(text, isRTL);

        if (displayText === "%") {
          const vocabKey = `${lineIndex}-${vocabIndex}`;
          const result = lineResults[vocabKey];
          const isPending = pendingAnswers.has(vocabKey);
          const isLocked = lockedAnswers.has(vocabKey);
          vocabIndex++; // Increment for next vocab item on this line
          return (
            <span key={textIndex} className="vocab-container inline-block mx-1">
              <select
                className={`vocab-select inline-block min-w-24 px-2 py-1 text-2xl border-2 rounded cursor-pointer bg-white transition-all duration-200 focus:outline-none focus:border-blue-500 ${
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
                  <div className="animate-spin w-4 h-4 border-2 border-blue-500 border-t-transparent rounded-full"></div>
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
        } else {
          return (
            <span
              key={textIndex}
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
