import { useState, useEffect, useMemo } from "react";
import { useNavigate } from "react-router";
import { useApiService } from "../services/api";
import { useNavigationGuidance } from "../hooks/useNavigationGuidance";
import { useUserContext } from "../contexts/UserContext";
import type { Story } from "../services/api";
import "./StoryList.css";
import "./StoryList-sections.css";

export function StoryList() {
  const api = useApiService();
  const navigate = useNavigate();
  const { getNavigationGuidance } = useNavigationGuidance();
  const { userInfo } = useUserContext();
  const [stories, setStories] = useState<Story[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [loadingStory, setLoadingStory] = useState<number | null>(null);
  const [showPast, setShowPast] = useState(false);
  const [showFuture, setShowFuture] = useState(false);

  // Group stories by course status
  const groupedStories = useMemo(() => {
    if (!userInfo?.enrolled_courses) {
      return { active: stories, past: [], future: [] };
    }

    const courseStatusMap = new Map(
      userInfo.enrolled_courses.map(c => [c.course_id, c.status])
    );

    const active: Story[] = [];
    const past: Story[] = [];
    const future: Story[] = [];
    console.log("Course Status Map:", courseStatusMap);
    console.log("Stories:", stories);

    stories.forEach(story => {
      const status = story.course_id ? courseStatusMap.get(story.course_id) : 'past';
      if (status === 'past') {
        past.push(story);
      } else if (status === 'future') {
        future.push(story);
      } else {
        active.push(story);
      }
    });

    return { active, past, future };
  }, [stories, userInfo]);

  useEffect(() => {
    const fetchStories = async () => {
      try {
        const response = await api.getStories();
        if (response.success && response.data) {
          setStories(response.data.stories);
          // Preload navigation guidance for first story only
          response.data.stories.slice(0, 1).forEach((story) => {
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
        {stories.length === 0 ? (
          <div className="stories-list">
            <div className="story-item">
              <h2>Welcome!</h2>
              <p>
                You're in! Please wait to be registered for a course so you can
                access some stories.
              </p>
            </div>
          </div>
        ) : (
          <>
            {/* Active Stories */}
            {groupedStories.active.length > 0 && (
              <section className="story-section">
                <h2 className="section-title">Current Stories</h2>
                <div className="stories-list">
                  {groupedStories.active.map((story) => (
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
                  ))}
                </div>
              </section>
            )}

            {/* Future Stories */}
            {groupedStories.future.length > 0 && (
              <section className="story-section">
                <button
                  onClick={() => setShowFuture(!showFuture)}
                  className="section-toggle"
                >
                  <span>Upcoming Stories ({groupedStories.future.length})</span>
                  <span className="toggle-icon">{showFuture ? '▼' : '▶'}</span>
                </button>
                {showFuture && (
                  <div className="stories-list">
                    {groupedStories.future.map((story) => (
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
                    ))}
                  </div>
                )}
              </section>
            )}

            {/* Past Stories */}
            {groupedStories.past.length > 0 && (
              <section className="story-section">
                <button
                  onClick={() => setShowPast(!showPast)}
                  className="section-toggle"
                >
                  <span>Archived Stories ({groupedStories.past.length})</span>
                  <span className="toggle-icon">{showPast ? '▼' : '▶'}</span>
                </button>
                {showPast && (
                  <div className="stories-list">
                    {groupedStories.past.map((story) => (
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
                    ))}
                  </div>
                )}
              </section>
            )}
          </>
        )}
      </main>
    </>
  );
}
