import React, { useState, useEffect } from "react";
import { Link } from "react-router";
import { api } from "../services/api";
import type { Story } from "../services/api";

export function StoryList() {
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
        <h1>Logos Stories</h1>
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
              <Link to={`/stories/${story.id}/page1`}>Start Reading</Link>
            </div>
          ))}
        </div>
      </main>
    </>
  );
}
