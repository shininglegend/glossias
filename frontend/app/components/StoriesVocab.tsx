import React, { useState, useEffect } from "react";
import { useParams, Link } from "react-router";
import { useApiService } from "../services/api";
import type { VocabData } from "../services/api";
import "./StoriesVocab.css";

export function StoriesVocab() {
  const { id } = useParams<{ id: string }>();
  const api = useApiService();
  const [pageData, setPageData] = useState<VocabData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [currentAudio, setCurrentAudio] = useState<HTMLAudioElement | null>(
    null,
  );
  const [selectedAnswers, setSelectedAnswers] = useState<{
    [key: number]: string;
  }>({});
  const [checkResults, setCheckResults] = useState<{
    [key: number]: boolean;
  } | null>(null);

  useEffect(() => {
    const fetchPageData = async () => {
      if (!id) {
        setError("Story ID is required");
        setLoading(false);
        return;
      }

      try {
        const response = await api.getStoryVocab(id);
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

  const playAudio = (audioUrl: string) => {
    if (currentAudio) {
      currentAudio.pause();
      currentAudio.currentTime = 0;
    }

    const audio = new Audio(audioUrl);
    setCurrentAudio(audio);

    audio.play().catch((err) => {
      console.error("Failed to play audio:", err);
    });

    audio.addEventListener("ended", () => {
      setCurrentAudio(null);
    });
  };

  const handleSelectChange = (lineIndex: number, value: string) => {
    setSelectedAnswers((prev) => ({
      ...prev,
      [lineIndex]: value,
    }));
    // Reset check results when answer changes
    setCheckResults(null);
  };

  const checkAnswers = async () => {
    if (!id || !pageData) return;

    // Collect answers in the format the API expects
    const answers = pageData.lines
      .map((line, index) => {
        if (line.has_vocab_or_grammar) {
          return {
            line_number: index,
            answers: [selectedAnswers[index] || ""],
          };
        }
        return null;
      })
      .filter((answer) => answer !== null);

    try {
      const response = await api.checkVocab(id, answers);
      if (response.success && response.data?.answers) {
        // Convert API response to indexed object
        const results: { [key: number]: boolean } = {};
        response.data.answers.forEach((answer: any) => {
          results[answer.line] = answer.correct;
        });
        setCheckResults(results);
      }
    } catch (err) {
      console.error("Failed to check answers:", err);
    }
  };

  const allBlanksFilledIn = () => {
    if (!pageData) return false;
    return pageData.lines.every((line, index) => {
      if (line.has_vocab_or_grammar) {
        return selectedAnswers[index] && selectedAnswers[index] !== "";
      }
      return true;
    });
  };

  useEffect(() => {
    return () => {
      if (currentAudio) {
        currentAudio.pause();
        currentAudio.currentTime = 0;
      }
    };
  }, [currentAudio]);

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
        <h2>Step 2: Vocabulary Practice</h2>
        <p>Fill in the blanks with the correct vocabulary words:</p>
      </header>
      <div className="container">
        {pageData.lines.map((line, lineIndex) => (
          <div
            key={lineIndex}
            className={`line ${line.has_vocab_or_grammar ? "has-vocab" : ""}`}
          >
            <div className="story-text">
              {line.text.map((text, textIndex) => {
                if (text === "%") {
                  return (
                    <select
                      key={textIndex}
                      className={`vocab-select ${
                        checkResults && checkResults[lineIndex] !== undefined
                          ? checkResults[lineIndex]
                            ? "correct"
                            : "incorrect"
                          : ""
                      }`}
                      value={selectedAnswers[lineIndex] || ""}
                      onChange={(e) =>
                        handleSelectChange(lineIndex, e.target.value)
                      }
                    >
                      <option value="">Choose...</option>
                      {pageData.vocab_bank.map((word, wordIndex) => (
                        <option key={wordIndex} value={word}>
                          {word}
                        </option>
                      ))}
                    </select>
                  );
                } else {
                  return <span key={textIndex}>{text}</span>;
                }
              })}
            </div>
            {/* TODO: Fix audio handling - line.audio_url may not exist in current Line interface */}
            {(line as any).audio_url && (
              <button
                onClick={() => playAudio((line as any).audio_url!)}
                className="audio-button"
                type="button"
              >
                <span className="material-icons">play_arrow</span>
              </button>
            )}
          </div>
        ))}

        <div className="button-group">
          <button
            type="button"
            id="checkAnswers"
            className={`btn ${!allBlanksFilledIn() ? "btn-disabled" : ""}`}
            onClick={checkAnswers}
            disabled={!allBlanksFilledIn()}
          >
            Check Answers
          </button>
        </div>

        {checkResults && (
          <div className="next-button">
            <Link to={`/stories/${id}/translate`} className="button-link">
              <span>Next Step</span>
              <span className="material-icons">arrow_forward</span>
            </Link>
          </div>
        )}
      </div>
    </>
  );
}
