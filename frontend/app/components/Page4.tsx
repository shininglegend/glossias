import React, { useState, useEffect } from "react";
import { useParams, Link } from "react-router";
import { useApiService } from "../services/api";
import type { PageData } from "../services/api";

export function Page4() {
  const { id } = useParams<{ id: string }>();
  const api = useApiService();
  const [pageData, setPageData] = useState<PageData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [currentAudio, setCurrentAudio] = useState<HTMLAudioElement | null>(
    null
  );

  useEffect(() => {
    const fetchPageData = async () => {
      if (!id) {
        setError("Story ID is required");
        setLoading(false);
        return;
      }

      try {
        const response = await api.getStoryPage4(id);
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
        <h2>Step 4: Translation</h2>
        <p>Listen to the story and practice translating it.</p>
      </header>
      <div className="container">
        {pageData.lines.map((line, lineIndex) => (
          <div key={lineIndex} className="line">
            <div className="story-text">{line.text.join("")}</div>
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
          <Link to="/" className="button-link">
            <span>Back to Stories</span>
            <span className="material-icons">home</span>
          </Link>
        </div>
      </div>
    </>
  );
}
