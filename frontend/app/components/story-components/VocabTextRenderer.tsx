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
  onAnswerChange: (vocabKey: string, value: string) => void;
  onCheckAnswer: (vocabKey: string) => void;
}

// Helper function to check if a line contains vocabulary placeholders
const lineHasVocab = (line: { text: string[] }): boolean => {
  return line.text.includes("%");
};

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
  checkingLines,
  isCurrentLine,
  isRTL,
  originalLine,
  onAnswerChange,
  onCheckAnswer,
}) => {
  let vocabIndex = 0; // Track vocab items within this line
  const isDisabled =
    completedLines.has(lineIndex) || !playedLines.has(lineIndex);

  // Check if all vocab items on this line have answers
  const totalVocabOnLine = line.text.filter((t) => t === "%").length;
  const lineVocabKeys = Array.from(
    { length: totalVocabOnLine },
    (_, i) => `${lineIndex}-${i}`
  );
  const allVocabAnswered = lineVocabKeys.every(
    (key) => selectedAnswers[key] && selectedAnswers[key].trim() !== ""
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
                } ${isDisabled ? "opacity-60 cursor-not-allowed" : ""}`}
                value={selectedAnswers[vocabKey] || ""}
                onChange={(e) => onAnswerChange(vocabKey, e.target.value)}
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
              {allVocabAnswered &&
                !completedLines.has(lineIndex) &&
                playedLines.has(lineIndex) &&
                vocabIndex === 1 && ( // Only show button on first vocab item
                  <button
                    onClick={() => onCheckAnswer(vocabKey)}
                    className="check-button w-6 h-6 bg-blue-500 text-white border-none rounded-full cursor-pointer text-sm flex items-center justify-center transition-colors duration-200 hover:bg-blue-600"
                    type="button"
                    disabled={checkingLines.has(lineIndex)}
                  >
                    {checkingLines.has(lineIndex) ? (
                      <div className="animate-spin w-3 h-3 border border-white border-t-transparent rounded-full"></div>
                    ) : (
                      "✓"
                    )}
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
