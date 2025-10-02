import { useState, useEffect } from "react";
import { useParams, useNavigate, Link } from "react-router";
import { useApiService } from "../services/api";
import { useNavigationGuidance } from "../hooks/useNavigationGuidance";
import type { PageData, TranslateData } from "../services/api";

export function StoriesTranslate() {
  const { id } = useParams<{ id: string }>();
  const api = useApiService();
  const navigate = useNavigate();
  const { getNavigationGuidance } = useNavigationGuidance();
  const [pageData, setPageData] = useState<PageData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [selectedLines, setSelectedLines] = useState<number[]>([]);
  const [translations, setTranslations] = useState<TranslateData | null>(null);
  const [translationLoading, setTranslationLoading] = useState(false);
  const [nextStepName, setNextStepName] = useState<string>("Next Step");
  const [showLinesMismatchWarning, setShowLinesMismatchWarning] =
    useState(false);

  useEffect(() => {
    const fetchPageData = async () => {
      if (!id) {
        setError("Story ID is required");
        setLoading(false);
        return;
      }

      try {
        const response = await api.getStoryWithAudio(id);
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

    fetchPageData();
  }, [id]);

  useEffect(() => {
    const fetchNextStep = async () => {
      if (!id) return;
      try {
        const guidance = await getNavigationGuidance(id, "translate");
        if (guidance) {
          setNextStepName(guidance.displayName);
        }
      } catch (error) {
        console.error("Failed to get navigation guidance:", error);
      }
    };

    fetchNextStep();
  }, [id, getNavigationGuidance]);

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

  const handleLineSelect = (lineIndex: number) => {
    if (translations) return; // Can't change selection after translation

    if (selectedLines.includes(lineIndex)) {
      setSelectedLines(selectedLines.filter((index) => index !== lineIndex));
    } else if (selectedLines.length < 5) {
      setSelectedLines([...selectedLines, lineIndex]);
    }
  };

  const handleGetTranslation = async () => {
    if (selectedLines.length !== 5 || !id) return;

    setTranslationLoading(true);
    try {
      const response = await api.getTranslations(id, selectedLines);
      if (response.success && response.data) {
        setTranslations(response.data);

        // Check if returned lines match requested lines
        const requestedIndices = selectedLines.map((i) => i + 1).sort(); // Convert to 1-indexed
        const returnedIndicesSorted = [...response.data.returned_lines].sort();

        const linesDiffer = !requestedIndices.every(
          (line, index) => line === returnedIndicesSorted[index],
        );
        setShowLinesMismatchWarning(linesDiffer);
      }
    } catch (err) {
      console.error("Failed to get translations:", err);
    } finally {
      setTranslationLoading(false);
    }
  };

  return (
    <>
      <header>
        <h1>{pageData.story_title}</h1>
        <h2>Translation</h2>

        <div className="bg-gray-50 border border-gray-300 p-4 mb-4 rounded-lg text-center">
          <div className="flex items-start justify-center">
            <span className="material-icons text-gray-600 mr-2 mt-1">info</span>
            <div>
              <p className="text-gray-700 mb-2">
                Select exactly 5 lines from the text below that you would like
                to see translated.
              </p>
              <p className="text-gray-700">
                Selected:{" "}
                <span className="font-semibold">{selectedLines.length}/5</span>
              </p>
            </div>
          </div>
        </div>

        {!translations && (
          <div className="mt-4">
            <button
              onClick={handleGetTranslation}
              disabled={selectedLines.length !== 5 || translationLoading}
              className={`px-6 py-3 rounded-lg font-medium ${
                selectedLines.length === 5 && !translationLoading
                  ? "bg-blue-500 text-white hover:bg-blue-600"
                  : "bg-gray-300 text-gray-500 cursor-not-allowed"
              }`}
            >
              {translationLoading ? (
                <>
                  <span className="material-icons animate-spin mr-2">
                    refresh
                  </span>
                  Getting Translations...
                </>
              ) : (
                `Get Translations (${selectedLines.length}/5)`
              )}
            </button>
          </div>
        )}

        {showLinesMismatchWarning && translations && (
          <div className="bg-yellow-50 border-l-4 border-yellow-400 p-6 rounded-lg mb-4">
            <div className="flex items-center mb-3">
              <span className="material-icons text-yellow-600 mr-2">
                warning
              </span>
              <h3 className="text-xl font-bold text-yellow-900">
                Different Lines Returned
              </h3>
            </div>
            <p className="text-yellow-800">
              You have already requested translations for this story, so your
              previously requested lines are being shown instead of your current
              selection.
            </p>
          </div>
        )}

        {translations && (
          <div className="text-center mt-4">
            <button
              onClick={async () => {
                try {
                  const guidance = await getNavigationGuidance(
                    id!,
                    "translate",
                  );
                  if (guidance) {
                    navigate(`/stories/${id}/${guidance.nextPage}`);
                  }
                } catch (error) {
                  console.error("Failed to get navigation guidance:", error);
                }
              }}
              className="inline-flex items-center px-8 py-4 bg-green-500 text-white rounded-lg hover:bg-green-600 text-lg font-semibold transition-all duration-200 shadow-lg"
            >
              <span>Continue to {nextStepName}</span>
              <span className="material-icons ml-2">arrow_forward</span>
            </button>
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

              // Helper function for RTL indentation
              const processTextForRTL = (text: string) => {
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

              return (
                <div
                  className={isRTL ? "text-right" : "text-left"}
                  dir={isRTL ? "rtl" : "ltr"}
                >
                  {pageData.lines.map((line, lineIndex) => {
                    const isSelected = selectedLines.includes(lineIndex);
                    const currentLineNumber = lineIndex + 1; // Convert to 1-indexed
                    const translationData = translations?.lines.find(
                      (t) => t.line_number === currentLineNumber,
                    );
                    const lineText = line.text.join("");
                    const { displayText, indentLevel } =
                      processTextForRTL(lineText);

                    return (
                      <div key={lineIndex} className="story-line inline">
                        <div
                          className={`line-content text-3xl rounded-lg cursor-pointer transition-colors duration-150 ${
                            translations
                              ? translationData
                                ? "bg-blue-100 border-2 border-blue-300"
                                : isSelected
                                  ? "bg-orange-200 border-2 border-orange-400"
                                  : "bg-gray-50"
                              : isSelected
                                ? "bg-blue-200 border-2 border-blue-400"
                                : "hover:bg-gray-100 border-2 border-transparent"
                          } ${isRTL ? "text-right" : "text-left"}`}
                          onClick={() => handleLineSelect(lineIndex)}
                          dir={isRTL ? "rtl" : "ltr"}
                        >
                          <span
                            style={
                              indentLevel > 0
                                ? { paddingRight: `${indentLevel * 2}em` }
                                : {}
                            }
                          >
                            {displayText}
                          </span>
                        </div>

                        {translationData && (
                          <div
                            className="mt-2 bg-yellow-50 border-l-4 border-yellow-400 rounded-r-lg text-left"
                            dir="ltr"
                          >
                            <p className="text-lg text-yellow-800">
                              {translationData.translation}
                            </p>
                          </div>
                        )}
                      </div>
                    );
                  })}
                </div>
              );
            })()}
        </div>
      </div>
    </>
  );
}
