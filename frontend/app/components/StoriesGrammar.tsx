import { useState, useEffect } from "react";
import { useParams, Link, useSearchParams, useNavigate } from "react-router";
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
  const navigate = useNavigate();
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

    const newPosition = { lineNumber: lineIndex + 1, position: charIndex };
    const existingIndex = selectedPositions.findIndex(
      (pos) => pos.lineNumber === lineIndex + 1 && pos.position === charIndex,
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
      (pos) => pos.lineNumber === lineIndex + 1 && pos.position === charIndex,
    );
  };

  const isCorrectAnswerPosition = (lineIndex: number, charIndex: number) => {
    if (!checkResults) return false;
    return checkResults.grammar_instances.some(
      (instance) =>
        instance.line_number === lineIndex + 1 &&
        charIndex >= instance.position[0] &&
        charIndex <= instance.position[1],
    );
  };

  const getUserSelectionResult = (lineIndex: number, charIndex: number) => {
    if (!checkResults || !isPositionSelected(lineIndex, charIndex)) return null;
    return checkResults.user_selections.find(
      (selection) =>
        selection.line_number === lineIndex + 1 &&
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
        <div className="bg-gray-50 border border-gray-300 p-4 mb-4 rounded-lg text-center">
          <div className="flex items-start justify-center">
            <span className="material-icons text-gray-600 mr-2 mt-1">info</span>
            <div>
              <p className="text-gray-700 mb-2">
                <strong>Grammar Point:</strong> {pageData.grammar_point}
                {pageData.grammar_description && (
                  <span> - {pageData.grammar_description}</span>
                )}
              </p>
              <p className="text-gray-700">
                Find each occurrence of this grammar point in the text below.
                Click on any character within each occurrence - you only need
                one selection per occurrence. Find exactly{" "}
                {pageData.instances_count} instances. Selected:{" "}
                <span className="font-semibold">
                  {selectedPositions.length}/{pageData.instances_count}
                </span>
              </p>
            </div>
          </div>
        </div>

        <div className="bg-blue-50 border border-blue-200 p-3 rounded-lg mb-4 text-center">
          <div className="flex items-center justify-center text-blue-700">
            <span className="material-icons mr-2 text-sm">touch_app</span>
            <span className="text-sm">
              Click individual characters in the text below. Selected characters
              will be highlighted in blue.
            </span>
          </div>
        </div>

        <div className="bg-gray-50 border border-gray-200 p-3 rounded-lg mb-4 text-center">
          <h5 className="font-medium text-gray-700 mb-2 text-sm">Legend:</h5>
          <div className="flex flex-wrap gap-4 text-sm justify-center">
            {!isSubmitted ? (
              <>
                <div className="flex items-center">
                  <span className="w-4 h-4 bg-yellow-100 border border-yellow-200 rounded-sm mr-2"></span>
                  <span className="text-gray-600">Hover to select</span>
                </div>
                <div className="flex items-center">
                  <span className="w-4 h-4 bg-blue-400 rounded-sm mr-2"></span>
                  <span className="text-gray-600">Selected</span>
                </div>
              </>
            ) : (
              <>
                <div className="flex items-center">
                  <span className="w-4 h-4 bg-green-200 rounded-sm mr-2"></span>
                  <span className="text-gray-600">Correct answer</span>
                </div>
                <div className="flex items-center">
                  <span className="w-4 h-4 bg-green-600 rounded-sm mr-2"></span>
                  <span className="text-gray-600">Your correct selection</span>
                </div>
                <div className="flex items-center">
                  <span className="w-4 h-4 bg-red-500 rounded-sm mr-2"></span>
                  <span className="text-gray-600">
                    Your incorrect selection
                  </span>
                </div>
              </>
            )}
          </div>
        </div>

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
                  className="inline-flex items-center px-8 py-4 bg-blue-500 text-white rounded-lg hover:bg-blue-600 text-lg font-semibold transition-all duration-200 shadow-lg"
                >
                  <span>Next Grammar Exercise</span>
                  <span className="material-icons ml-2">arrow_forward</span>
                </Link>
              ) : (
                <button
                  onClick={async () => {
                    try {
                      const response = await api.getNavigationGuidance(
                        id!,
                        "placeholder-user-id",
                        "grammar",
                      );
                      if (response.success && response.data) {
                        navigate(`/stories/${id}/${response.data.nextPage}`);
                      }
                    } catch (error) {
                      console.error(
                        "Failed to get navigation guidance:",
                        error,
                      );
                    }
                  }}
                  className="inline-flex items-center px-8 py-4 bg-green-500 text-white rounded-lg hover:bg-green-600 text-lg font-semibold transition-all duration-200 shadow-lg"
                >
                  <span>Continue to Score</span>
                  <span className="material-icons ml-2">arrow_forward</span>
                </button>
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
              const languageCode = pageData.language;
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

                          let className =
                            "cursor-pointer select-none transition-colors duration-150";

                          if (!isSubmitted) {
                            if (isSelected) {
                              className +=
                                " bg-blue-400 text-white rounded-sm shadow-sm";
                            } else {
                              className +=
                                " hover:bg-yellow-100 hover:shadow-sm rounded-sm";
                            }
                          } else {
                            // Show correct answers in light green
                            if (isCorrectAnswerPosition(lineIndex, charIndex)) {
                              className += " bg-green-200 rounded-sm";
                            }

                            // Overlay user selections with their result
                            const userResult = getUserSelectionResult(
                              lineIndex,
                              charIndex,
                            );
                            if (userResult) {
                              className += userResult.correct
                                ? " bg-green-600 text-white rounded-sm shadow-sm" // Dark green for correct selection
                                : " bg-red-500 text-white rounded-sm shadow-sm"; // Red for incorrect selection
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
