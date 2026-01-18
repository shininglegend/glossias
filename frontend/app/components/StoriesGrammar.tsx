import { useState, useEffect } from "react";
import { useParams, Link, useSearchParams, useNavigate } from "react-router";
import { useApiService } from "../services/api";
import { useNavigationGuidance } from "../hooks/useNavigationGuidance";
import { CompletionMessage } from "./story-components/CompletionMessage";
import type { GrammarData } from "../services/api";

interface ClickPosition {
  lineNumber: number;
  position: number;
}

interface CheckGrammarResponse {
  correct: boolean;
  matched_position?: [number, number];
  total_instances: number;
  next_grammar_point: number | null;
}

export function StoriesGrammar() {
  const { id } = useParams<{ id: string }>();
  const [searchParams] = useSearchParams();
  const grammarPointId = searchParams.get("id");
  const api = useApiService();
  const navigate = useNavigate();
  const { getNavigationGuidance } = useNavigationGuidance();
  const [pageData, setPageData] = useState<GrammarData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [clickedPositions, setClickedPositions] = useState<Set<string>>(
    new Set()
  );
  const [correctPositions, setCorrectPositions] = useState<Set<string>>(
    new Set()
  );
  const [incorrectPositions, setIncorrectPositions] = useState<Set<string>>(
    new Set()
  );
  const [isSubmittingAnswer, setIsSubmittingAnswer] = useState(false);
  const [nextGrammarPoint, setNextGrammarPoint] = useState<number | null>(null);
  const [foundInstances, setFoundInstances] = useState(0);
  const [nextStepName, setNextStepName] = useState<string>("Next Step");

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

          // Initialize state from existing data
          const newCorrect = new Set<string>();
          const newClicked = new Set<string>();
          let foundCount = 0;

          if (response.data.found_instances) {
            response.data.found_instances.forEach((instance) => {
              const lineIndex = instance.line_number - 1;
              const [start, end] = instance.position;
              for (let i = start; i <= end; i++) {
                newCorrect.add(`${lineIndex}-${i}`);
                newClicked.add(`${lineIndex}-${i}`);
              }
              foundCount++;
            });
          }

          if (response.data.incorrect_instances) {
            response.data.incorrect_instances.forEach((instance) => {
              const lineIndex = instance.line_number - 1;
              const [start, end] = instance.position;
              const positionKey = `${lineIndex}-${start}`;
              setIncorrectPositions((prev) => new Set([...prev, positionKey]));
              newClicked.add(positionKey);
            });
          }

          setCorrectPositions(newCorrect);
          setClickedPositions(newClicked);
          setFoundInstances(foundCount);

          if (response.data.next_grammar_point) {
            setNextGrammarPoint(response.data.next_grammar_point);
          }
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
    setClickedPositions(new Set());
    setCorrectPositions(new Set());
    setIncorrectPositions(new Set());
    setFoundInstances(0);
    setNextGrammarPoint(null);

    fetchPageData();
  }, [id, grammarPointId]);

  useEffect(() => {
    const fetchNextStep = async () => {
      if (!id) return;
      try {
        const guidance = await getNavigationGuidance(id, "grammar");
        if (guidance) {
          setNextStepName(guidance.displayName);
        }
      } catch (error) {
        console.error("Failed to fetch next step name:", error);
      }
    };

    fetchNextStep();
  }, [id, getNavigationGuidance]);

  useEffect(() => {
    // Warm up serverless function to avoid cold start delay
    if (!pageData || !id) return;

    const warmUpApi = async () => {
      try {
        // Make a dummy call to warm up the serverless environment
        await fetch("/api/health", { method: "GET" }).catch(() => {});
      } catch {
        // Silently fail warming
      }
    };

    warmUpApi();
  }, [pageData, id]);

  const handleTextClick = async (lineIndex: number, charIndex: number) => {
    if (isSubmittingAnswer || !pageData || !id) return;

    const positionKey = `${lineIndex}-${charIndex}`;

    // Skip if already clicked or already processed
    if (
      clickedPositions.has(positionKey) ||
      correctPositions.has(positionKey) ||
      incorrectPositions.has(positionKey)
    )
      return;

    setIsSubmittingAnswer(true);

    try {
      const result = await api.checkGrammar(id, pageData.grammar_point_id, {
        grammar_point_id: pageData.grammar_point_id,
        line_number: lineIndex + 1,
        position: charIndex,
      });

      if (result.success && result.data) {
        const newClicked = new Set(clickedPositions);
        newClicked.add(positionKey);
        setClickedPositions(newClicked);

        if (result.data.correct) {
          // Mark the full matched position as correct
          if (result.data.matched_position) {
            const [start, end] = result.data.matched_position;
            const newCorrect = new Set(correctPositions);
            for (let i = start; i <= end; i++) {
              newCorrect.add(`${lineIndex}-${i}`);
            }
            setCorrectPositions(newCorrect);
            setFoundInstances((prev) => prev + 1);
          }
        } else {
          // Mark this position as incorrect
          const newIncorrect = new Set(incorrectPositions);
          newIncorrect.add(positionKey);
          setIncorrectPositions(newIncorrect);
        }

        if (result.data.next_grammar_point) {
          setNextGrammarPoint(result.data.next_grammar_point);
        }
      }
    } catch (err) {
      console.error("Failed to check grammar:", err);
    } finally {
      setIsSubmittingAnswer(false);
    }
  };

  const isPositionCorrect = (lineIndex: number, charIndex: number) => {
    const positionKey = `${lineIndex}-${charIndex}`;
    return correctPositions.has(positionKey);
  };

  const isPositionIncorrect = (lineIndex: number, charIndex: number) => {
    const positionKey = `${lineIndex}-${charIndex}`;
    return incorrectPositions.has(positionKey);
  };

  const isPositionClicked = (lineIndex: number, charIndex: number) => {
    const positionKey = `${lineIndex}-${charIndex}`;
    return clickedPositions.has(positionKey);
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
        <h2>Grammar Focus</h2>
        {foundInstances === pageData.instances_count && (
          <div className="m-4 text-center">
            {nextGrammarPoint ? (
              <Link
                to={`/stories/${id}/grammar?id=${nextGrammarPoint}`}
                className="inline-flex items-center px-8 py-4 bg-primary-500 text-white rounded-lg hover:bg-primary-600 text-lg font-semibold transition-all duration-200 shadow-lg"
              >
                <span>Next Grammar Exercise</span>
                <span className="material-icons ml-2">arrow_forward</span>
              </Link>
            ) : (
              <CompletionMessage
                currentStepName="grammar"
                nextStepName={nextStepName}
                onContinue={async () => {
                  try {
                    const guidance = await getNavigationGuidance(
                      id!,
                      "grammar"
                    );
                    if (guidance) {
                      navigate(`/stories/${id}/${guidance.nextPage}`);
                    }
                  } catch (error) {
                    console.error("Failed to get navigation guidance:", error);
                  }
                }}
              />
            )}
          </div>
        )}
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
                Click on any character within each occurrence. Find exactly{" "}
                {pageData.instances_count} instances. Found:{" "}
                <span className="font-semibold">
                  {foundInstances}/{pageData.instances_count}
                </span>
              </p>
            </div>
          </div>
        </div>

        <div className="bg-primary-50 border border-primary-200 p-3 rounded-lg mb-4 text-center">
          <div className="flex items-center justify-center text-primary-700">
            <span className="material-icons mr-2 text-sm">touch_app</span>
            <span className="text-sm">
              {isSubmittingAnswer
                ? "Checking your answer..."
                : "Click characters to check immediately. Green = correct, red = incorrect."}
            </span>
          </div>
        </div>

        <div className="bg-gray-50 border border-gray-200 p-3 rounded-lg mb-4 text-center">
          <h5 className="font-medium text-gray-700 mb-2 text-sm">Legend:</h5>
          <div className="flex flex-wrap gap-4 text-sm justify-center">
            <div className="flex items-center">
              <span className="w-4 h-4 bg-secondary-100 border border-secondary-200 rounded-sm mr-2"></span>
              <span className="text-gray-600">Hover to click</span>
            </div>
            <div className="flex items-center">
              <span className="w-4 h-4 bg-green-500 bg-opacity-70 rounded-sm mr-2"></span>
              <span className="text-gray-600">Correct</span>
            </div>
            <div className="flex items-center">
              <span className="w-4 h-4 bg-red-500 bg-opacity-70 rounded-sm mr-2"></span>
              <span className="text-gray-600">Incorrect</span>
            </div>
          </div>
        </div>
      </header>
      <div className="max-w-4xl mx-auto px-5">
        <div className="story-lines text-2xl max-w-3xl mx-auto">
          {pageData.lines.length > 0 &&
            (() => {
              const RTL_LANGUAGES = ["he", "ar", "fa", "ur"];
              const languageCode = pageData.language;
              const isRTL =
                languageCode && RTL_LANGUAGES.includes(languageCode);

              // Helper function for RTL indentation - keep original positions intact
              const getIndentLevel = (text: string) => {
                if (!isRTL || typeof text !== "string") {
                  return 0;
                }

                const leadingTabs = text.match(/^\t*/)?.[0] || "";
                return leadingTabs.length;
              };

              return (
                <div
                  className={isRTL ? "text-right" : "text-left"}
                  dir={isRTL ? "rtl" : "ltr"}
                >
                  {pageData.lines.map((line, lineIndex) => (
                    <div key={lineIndex} className="story-line inline">
                      <div className="line-content text-3xl inline">
                        {(() => {
                          const indentLevel = getIndentLevel(line.text);
                          const firstNonTabIndex = Math.min(
                            indentLevel,
                            line.text.length - 1
                          );
                          return line.text.split("").map((char, charIndex) => {
                            let className = isSubmittingAnswer
                              ? "select-none transition-colors duration-150 cursor-wait opacity-50"
                              : "cursor-pointer select-none transition-colors duration-150";

                            if (isPositionCorrect(lineIndex, charIndex)) {
                              className +=
                                " bg-green-500 bg-opacity-70 text-white rounded-sm shadow-sm";
                            } else if (
                              isPositionIncorrect(lineIndex, charIndex)
                            ) {
                              className +=
                                " bg-red-500 bg-opacity-70 text-white rounded-sm shadow-sm";
                            } else if (!isSubmittingAnswer) {
                              className +=
                                " hover:bg-secondary-100 hover:shadow-sm rounded-sm";
                            }

                            return (
                              <span
                                key={charIndex}
                                className={className}
                                style={
                                  indentLevel > 0 &&
                                  charIndex === firstNonTabIndex
                                    ? { paddingRight: `${indentLevel * 2}em` }
                                    : {}
                                }
                                onClick={() => {
                                  if (!isSubmittingAnswer) {
                                    handleTextClick(lineIndex, charIndex);
                                  }
                                }}
                              >
                                {char === "\t" && charIndex < indentLevel
                                  ? ""
                                  : char}
                              </span>
                            );
                          });
                        })()}
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
