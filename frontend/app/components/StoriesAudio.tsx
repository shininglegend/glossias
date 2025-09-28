import React, { useState, useEffect } from "react";
import { useParams, Link } from "react-router";
import { useApiService } from "../services/api";
import type { PageData, AudioFile } from "../services/api";

export function StoriesAudio() {
  const { id } = useParams<{ id: string }>();
  const api = useApiService();
  const [pageData, setPageData] = useState<PageData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [currentAudio, setCurrentAudio] = useState<HTMLAudioElement | null>(
    null,
  );
  const [signedAudioURLs, setSignedAudioURLs] = useState<{ [key: number]: string }>({});
  const [selectedAudioLabel, setSelectedAudioLabel] = useState<string>("complete");

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

          // Fetch signed URLs for selected audio label
          const audioResponse = await api.getSignedAudioURLs(id, selectedAudioLabel);
          if (audioResponse.success && audioResponse.data) {
            setSignedAudioURLs(audioResponse.data);
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

    fetchPageData();
  }, [id, selectedAudioLabel]);

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

  // Get available audio labels from the data
  const getAvailableLabels = () => {
    if (!pageData) return [];
    const labels = new Set<string>();
    pageData.lines.forEach(line => {
      line.audio_files.forEach(audio => labels.add(audio.label));
    });
    return Array.from(labels);
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
        <h2>Step 1: Listen to the entire story.</h2>
        <p>
          Click the first play button to listen to the story. It will play from
          that point onward.
        </p>
        <hr />
        <p>Click any play button to restart the story from that point.</p>

        {getAvailableLabels().length > 1 && (
          <div className="audio-controls">
            <label htmlFor="audio-label-select">Audio Type: </label>
            <select
              id="audio-label-select"
              value={selectedAudioLabel}
              onChange={(e) => setSelectedAudioLabel(e.target.value)}
            >
              {getAvailableLabels().map(label => (
                <option key={label} value={label}>
                  {label.replace('_', ' ')}
                </option>
              ))}
            </select>
          </div>
        )}
      </header>
      <div className="container">
        {pageData.lines.length > 0 ? (
          pageData.lines.map((line, index) => {
            // Find audio file for selected label
            const selectedAudio = line.audio_files.find(audio => audio.label === selectedAudioLabel);
            const audioURL = selectedAudio ? signedAudioURLs[selectedAudio.id] : null;

            return (
              <div key={index} className="line">
                <span className="story-text">{line.text.join("")}</span>
                {audioURL && (
                  <button
                    onClick={() => playAudio(audioURL)}
                    className="audio-button"
                    type="button"
                  >
                    <span className="material-icons">play_arrow</span>
                  </button>
                )}
              </div>
            );
          })
        ) : (
          <p>This story has no text associated with it yet.</p>
        )}

        <div className="next-button">
          <Link to={`/stories/${id}/vocab`} className="button-link">
            <span>Next Step</span>
            <span className="material-icons">arrow_forward</span>
          </Link>
        </div>
      </div>
    </>
  );
}
