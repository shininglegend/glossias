import { useState, useEffect } from "react";
import { useNavigate } from "react-router";
import { useApiService } from "../services/api";
import type { Story } from "../services/api";
import "./StoryList.css";

export function StoryList() {
  const api = useApiService();
  const navigate = useNavigate();
  const [stories, setStories] = useState<Story[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchStories = async () => {
      try {
        const response = await api.getStories();
        if (response.success && response.data) {
          setStories(response.data.stories);
        } else {
          setError(response.error || "Failed to fetch stories");
        }
      } catch (err) {
        setError("Failed to fetch stories");
      } finally {
        setLoading(false);
      }
    };

    fetchStories();
  }, []);

  const handleStartReading = async (storyId: number) => {
    try {
      const response = await api.getNavigationGuidance(
        storyId.toString(),
        "placeholder-user-id",
        "list",
      );
      if (response.success && response.data) {
        navigate(`/stories/${storyId}/${response.data.nextPage}`);
      }
    } catch (error) {
      console.error("Failed to get navigation guidance:", error);
    }
  };

  if (loading) {
    return (
      <div className="container">
        <p>Loading stories...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="container">
        <p>Error: {error}</p>
      </div>
    );
  }

  return (
    <>
      <header>
        <h1>Glossias</h1>
        <p>Select a story to begin reading</p>
        <hr />
      </header>
      <main className="container">
        <div className="stories-list">
          {stories.map((story) => (
            <div key={story.id} className="story-item">
              <h2>{story.title}</h2>
              <p>
                Week {story.week_number}
                {story.day_letter}
              </p>
              <button
                onClick={() => handleStartReading(story.id)}
                className="start-reading-button"
              >
                Start Reading
              </button>
            </div>
          ))}
        </div>
      </main>
    </>
  );
}
