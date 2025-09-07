import React, { useState, useEffect } from "react";
import { useParams, Link } from "react-router";
import { useApiService } from "../services/api";
import type { Page3Data } from "../services/api";

export function Page3() {
  const { id } = useParams<{ id: string }>();
  const api = useApiService();
  const [pageData, setPageData] = useState<Page3Data | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [currentAudio, setCurrentAudio] = useState<HTMLAudioElement | null>(
    null,
  );

  useEffect(() => {
    const fetchPageData = async () => {
      if (!id) {
        setError("Story ID is required");
        setLoading(false);
        return;
      }

      try {
        const response = await api.getStoryPage3(id);
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
        <h2>Step 3: Grammar Focus</h2>
        <p>Grammar Point: {pageData.grammar_point}</p>
      </header>
      <div className="container">
        {pageData.lines.map((line, lineIndex) => (
          <div
            key={lineIndex}
            className={`line ${line.has_vocab_or_grammar ? "has-grammar" : ""}`}
          >
            <div className="story-text">
              {line.text.map((text, textIndex) => {
                if (text === "%") {
                  return (
                    <span key={textIndex} className="grammar-highlight-start" />
                  );
                } else if (text === "&") {
                  return (
                    <span key={textIndex} className="grammar-highlight-end" />
                  );
                } else {
                  return <span key={textIndex}>{text}</span>;
                }
              })}
            </div>
            {line.audio_url && (
              <button
                onClick={() => playAudio(line.audio_url!)}
                className="audio-button"
                type="button"
              >
                <span className="material-icons">play_arrow</span>
              </button>
            )}
          </div>
        ))}

        <div className="next-button">
          <Link to={`/stories/${id}/page4`} className="button-link">
            <span>Next Step</span>
            <span className="material-icons">arrow_forward</span>
          </Link>
        </div>
      </div>
    </>
  );
}
