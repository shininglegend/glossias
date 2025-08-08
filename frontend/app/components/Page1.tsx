import React, { useState, useEffect } from "react";
import { useParams, Link } from "react-router";
import { api } from "../services/api";
import type { PageData } from "../services/api";

export function Page1() {
  const { id } = useParams<{ id: string }>();
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
        const response = await api.getStoryPage1(id);
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
    // Stop current audio if playing
    if (currentAudio) {
      currentAudio.pause();
      currentAudio.currentTime = 0;
    }

    // Create and play new audio
    const audio = new Audio(audioUrl);
    setCurrentAudio(audio);

    audio.play().catch((err) => {
      console.error("Failed to play audio:", err);
    });

    // Clean up when audio ends
    audio.addEventListener("ended", () => {
      setCurrentAudio(null);
    });
  };

  useEffect(() => {
    // Cleanup audio on unmount
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
        <h2>Step 1: Listen to the entire story.</h2>
        <p>
          Click the first play button to listen to the story. It will play from
          that point onward. <hr />
          Click any play button to restart the story from that point.
        </p>
      </header>
      <div className="container">
        {pageData.lines.length > 0 ? (
          pageData.lines.map((line, index) => (
            <div key={index} className="line">
              <span className="story-text">{line.text.join("")}</span>
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
          ))
        ) : (
          <p>This story has no text associated with it yet.</p>
        )}

        <div className="next-button">
          <Link to={`/stories/${id}/page2`} className="button-link">
            <span>Next Step</span>
            <span className="material-icons">arrow_forward</span>
          </Link>
        </div>
      </div>
    </>
  );
}
