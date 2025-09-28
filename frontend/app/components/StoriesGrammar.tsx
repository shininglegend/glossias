import React, { useState, useEffect } from "react";
import { useParams, Link, useSearchParams } from "react-router";
import { useApiService } from "../services/api";
import type { GrammarData } from "../services/api";

interface ClickPosition {
  lineNumber: number;
  position: number;
}

interface CheckGrammarResponse {
  correct: number;
  wrong: number;
  total_answers: number;
  grammar_instances: Array<{
    line_number: number;
    position: [number, number];
    text: string;
    user_selected: boolean;
  }>;
  user_selections: Array<{
    line_number: number;
    position: [number, number];
    text: string;
    correct: boolean;
  }>;
  next_grammar_point_id: string | null;
}

export function StoriesGrammar() {
  const { id } = useParams<{ id: string }>();
  const [searchParams] = useSearchParams();
  const grammarPointId = searchParams.get("id");
  const api = useApiService();
  const [pageData, setPageData] = useState<GrammarData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedPositions, setSelectedPositions] = useState<ClickPosition[]>(
    [],
  );
  const [checkResults, setCheckResults] = useState<CheckGrammarResponse | null>(
    null,
  );
  const [isSubmitted, setIsSubmitted] = useState(false);

  useEffect(() => {
    const fetchPageData = async () => {
      if (!id) {
        setError("Story ID is required");
        setLoading(false);
        return;
      }

      try {
        const response = grammarPointId
          ? await api.getStoryGrammar(id, grammarPointId)
          : await api.getStoryGrammar(id);
        if (response.success && response.data) {
          setPageData(response.data);
        } else {
          setError(response.error || "Failed to fetch page data");
        }
      } catch (err) {
        setError("Failed to fetch page data");
      } finally {
        setLoading(false);
      }
    };

    // Reset state when starting new fetch
    setSelectedPositions([]);
    setCheckResults(null);
    setIsSubmitted(false);

    fetchPageData();
  }, [id, grammarPointId]);

  const handleTextClick = (lineIndex: number, charIndex: number) => {
    if (isSubmitted) return;

    const newPosition = { lineNumber: lineIndex, position: charIndex };
    const existingIndex = selectedPositions.findIndex(
      (pos) => pos.lineNumber === lineIndex && pos.position === charIndex,
    );

    if (existingIndex >= 0) {
      setSelectedPositions((prev) =>
        prev.filter((_, index) => index !== existingIndex),
      );
    } else if (selectedPositions.length < (pageData?.instances_count || 0)) {
      setSelectedPositions((prev) => [...prev, newPosition]);
    }
  };

  const submitAnswers = async () => {
    if (
      !pageData ||
      !id ||
      selectedPositions.length !== pageData.instances_count
    )
      return;

    const answersByLine = selectedPositions.reduce(
      (acc, pos) => {
        const existing = acc.find(
          (item) => item.line_number === pos.lineNumber,
        );
        if (existing) {
          existing.positions.push(pos.position);
        } else {
          acc.push({
            line_number: pos.lineNumber,
            positions: [pos.position],
          });
        }
        return acc;
      },
      [] as Array<{ line_number: number; positions: number[] }>,
    );

    try {
      const result = await api.checkGrammar(
        id!,
        pageData.grammar_point_id,
        answersByLine,
      );

      if (result.success) {
        setCheckResults(result.data);
        setIsSubmitted(true);
      }
    } catch (err) {
      console.error("Failed to submit answers:", err);
    }
  };

  const isPositionSelected = (lineIndex: number, charIndex: number) => {
    return selectedPositions.some(
      (pos) => pos.lineNumber === lineIndex && pos.position === charIndex,
    );
  };

  const isCorrectAnswerPosition = (lineIndex: number, charIndex: number) => {
    if (!checkResults) return false;
    return checkResults.grammar_instances.some(
      (instance) =>
        instance.line_number === lineIndex &&
        charIndex >= instance.position[0] &&
        charIndex <= instance.position[1],
    );
  };

  const getUserSelectionResult = (lineIndex: number, charIndex: number) => {
    if (!checkResults || !isPositionSelected(lineIndex, charIndex)) return null;
    return checkResults.user_selections.find(
      (selection) =>
        selection.line_number === lineIndex &&
        charIndex >= selection.position[0] &&
        charIndex <= selection.position[1],
    );
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

  return (
    <>
      <header>
        <h1>{pageData.story_title}</h1>
        <h2>Step 3: Grammar Focus</h2>
        <p>
          <strong>Grammar Point:</strong> {pageData.grammar_point}
        </p>
        {pageData.grammar_description && (
          <p>
            <strong>Description:</strong> {pageData.grammar_description}
          </p>
        )}
        <p>
          Find {pageData.instances_count} instances by clicking on the text.
          Selected: {selectedPositions.length}/{pageData.instances_count}
        </p>

        {!isSubmitted ? (
          <div className="mt-4">
            <button
              onClick={submitAnswers}
              disabled={selectedPositions.length !== pageData.instances_count}
              className={`px-6 py-3 rounded-lg font-medium ${
                selectedPositions.length === pageData.instances_count
                  ? "bg-blue-500 text-white hover:bg-blue-600"
                  : "bg-gray-300 text-gray-500 cursor-not-allowed"
              }`}
            >
              Check Answers ({selectedPositions.length}/
              {pageData.instances_count})
            </button>
          </div>
        ) : (
          <div className="mt-4 space-y-4">
            {checkResults && (
              <div className="bg-blue-50 border-l-4 border-blue-400 p-6 rounded-lg">
                <div className="flex items-center mb-3">
                  <span className="material-icons text-blue-600 mr-2">
                    assessment
                  </span>
                  <h3 className="text-xl font-bold text-blue-900">
                    Your Results
                  </h3>
                </div>
                <div className="grid grid-cols-3 gap-4 text-center">
                  <div className="bg-green-100 p-3 rounded-lg">
                    <div className="text-2xl font-bold text-green-700">
                      {checkResults.correct}
                    </div>
                    <div className="text-sm text-green-600">Correct</div>
                  </div>
                  <div className="bg-red-100 p-3 rounded-lg">
                    <div className="text-2xl font-bold text-red-700">
                      {checkResults.wrong}
                    </div>
                    <div className="text-sm text-red-600">Wrong</div>
                  </div>
                  <div className="bg-gray-100 p-3 rounded-lg">
                    <div className="text-2xl font-bold text-gray-700">
                      {checkResults.total_answers}
                    </div>
                    <div className="text-sm text-gray-600">Total</div>
                  </div>
                </div>
              </div>
            )}
            <div className="text-center">
              {checkResults?.next_grammar_point_id ? (
                <Link
                  to={`/stories/${id}/grammar?id=${checkResults.next_grammar_point_id}`}
                  className="inline-flex items-center px-6 py-3 bg-blue-500 text-white rounded-lg hover:bg-blue-600"
                >
                  <span>Next Grammar Exercise</span>
                  <span className="material-icons ml-2">arrow_forward</span>
                </Link>
              ) : (
                <Link
                  to={`/stories/${id}/score`}
                  className="inline-flex items-center px-6 py-3 bg-green-500 text-white rounded-lg hover:bg-green-600"
                >
                  <span>Continue to Score</span>
                  <span className="material-icons ml-2">arrow_forward</span>
                </Link>
              )}
            </div>
          </div>
        )}
      </header>
      <div className="max-w-4xl mx-auto px-5">
        <div className="story-lines text-2xl max-w-3xl mx-auto">
          {pageData.lines.length > 0 &&
            (() => {
              const RTL_LANGUAGES = ["he", "ar", "fa", "ur"];
              const languageCode = pageData.languageCode;
              const isRTL =
                languageCode && RTL_LANGUAGES.includes(languageCode);

              return (
                <div
                  className={isRTL ? "text-right" : "text-left"}
                  dir={isRTL ? "rtl" : "ltr"}
                >
                  {pageData.lines.map((line, lineIndex) => (
                    <div key={lineIndex} className="story-line inline">
                      <div className="line-content text-3xl inline">
                        {line.text.split("").map((char, charIndex) => {
                          const isSelected = isPositionSelected(
                            lineIndex,
                            charIndex,
                          );

                          let className = "cursor-pointer select-none";

                          if (!isSubmitted) {
                            if (isSelected) {
                              className += " bg-blue-200";
                            } else {
                              className += " hover:bg-gray-100";
                            }
                          } else {
                            // Show correct answers in light green
                            if (isCorrectAnswerPosition(lineIndex, charIndex)) {
                              className += " bg-green-200";
                            }

                            // Overlay user selections with their result
                            const userResult = getUserSelectionResult(
                              lineIndex,
                              charIndex,
                            );
                            if (userResult) {
                              className += userResult.correct
                                ? " bg-green-600 text-white" // Dark green for correct selection
                                : " bg-red-500 text-white"; // Red for incorrect selection
                            }
                          }

                          return (
                            <span
                              key={charIndex}
                              className={className}
                              onClick={() =>
                                handleTextClick(lineIndex, charIndex)
                              }
                            >
                              {char}
                            </span>
                          );
                        })}
                      </div>
                    </div>
                  ))}
                </div>
              );
            })()}
        </div>
      </div>
    </>
  );
}
