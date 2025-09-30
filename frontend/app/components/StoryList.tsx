import { useState, useEffect } from "react";
import { useNavigate } from "react-router";
import { useApiService } from "../services/api";
import { useNavigationGuidance } from "../hooks/useNavigationGuidance";
import type { Story } from "../services/api";
import "./StoryList.css";

export function StoryList() {
  const api = useApiService();
  const navigate = useNavigate();
  const { getNavigationGuidance } = useNavigationGuidance();
  const [stories, setStories] = useState<Story[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [loadingStory, setLoadingStory] = useState<number | null>(null);

  useEffect(() => {
    const fetchStories = async () => {
      try {
        const response = await api.getStories();
        if (response.success && response.data) {
          setStories(response.data.stories);
          // Preload navigation guidance to avoid cold starts
          response.data.stories.forEach((story) => {
            getNavigationGuidance(story.id.toString(), "list").catch(() => {
              // Silently fail preloading
            });
          });
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
  }, [getNavigationGuidance]);

  const handleStoryClick = async (storyId: number) => {
    setLoadingStory(storyId);
    try {
      const guidance = await getNavigationGuidance(storyId.toString(), "list");
      if (guidance) {
        navigate(`/stories/${storyId}/${guidance.nextPage}`);
      }
    } catch (error) {
      console.error("Failed to get navigation guidance:", error);
    } finally {
      setLoadingStory(null);
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
          {stories.length === 0 ? (
            <div className="story-item">
              <h2>Welcome!</h2>
              <p>
                You're in! Please wait to be registered for a course so you can
                access some stories.
              </p>
            </div>
          ) : (
            stories.map((story) => (
              <div key={story.id} className="story-item">
                <h2>{story.title}</h2>
                <p>
                  Week {story.week_number}
                  {story.day_letter}
                </p>
                <button
                  onClick={() => handleStoryClick(story.id)}
                  className="start-reading-button"
                  disabled={loadingStory === story.id}
                >
                  {loadingStory === story.id ? (
                    <div className="flex items-center gap-2">
                      <div className="animate-spin w-4 h-4 border border-white border-t-transparent rounded-full"></div>
                      Loading...
                    </div>
                  ) : (
                    "Start Reading"
                  )}
                </button>
              </div>
            ))
          )}
        </div>
      </main>
    </>
  );
}
