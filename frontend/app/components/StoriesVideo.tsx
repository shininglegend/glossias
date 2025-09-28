import { useState, useEffect } from "react";
import { useParams, Link } from "react-router";
import { useApiService } from "../services/api";
import type { StoryMetadata } from "../services/api";

function getYouTubeEmbedUrl(url: string): string | null {
  const regex = /(?:youtube\.com\/watch\?v=|youtu\.be\/)([a-zA-Z0-9_-]{11})/;
  const match = url.match(regex);
  return match ? `https://www.youtube.com/embed/${match[1]}` : null;
}

function isYouTubeUrl(url: string): boolean {
  return url.includes("youtube.com") || url.includes("youtu.be");
}

export function StoriesVideo() {
  const { id } = useParams<{ id: string }>();
  const api = useApiService();
  const [metadata, setMetadata] = useState<StoryMetadata | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [videoWatched, setVideoWatched] = useState(false);

  useEffect(() => {
    const fetchMetadata = async () => {
      if (!id) {
        setError("Story ID is required");
        setLoading(false);
        return;
      }

      try {
        const response = await api.getStoryMetadata(id);
        if (response.success && response.data) {
          setMetadata(response.data);
        } else {
          setError(response.error || "Failed to fetch story metadata");
        }
      } catch (err) {
        setError("Failed to fetch story metadata");
      } finally {
        setLoading(false);
      }
    };

    fetchMetadata();
  }, [id]);

  if (loading) {
    return (
      <div className="container">
        <p>Loading video...</p>
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

  if (!metadata) {
    return (
      <div className="container">
        <p>No story found</p>
        <Link to="/">Back to Stories</Link>
      </div>
    );
  }

  if (!metadata.videoUrl) {
    return (
      <div className="container">
        <h1>
          {typeof metadata.title === "string"
            ? metadata.title
            : metadata.title?.en || "Story"}
        </h1>
        <p>No video available for this story</p>
        <div className="text-center">
          <Link
            to={`/stories/${id}/vocab`}
            className="inline-flex items-center px-8 py-4 bg-green-500 text-white rounded-lg hover:bg-green-600 text-lg font-semibold transition-all duration-200 shadow-lg"
          >
            <span>Skip to Vocabulary</span>
            <span className="material-icons ml-2">arrow_forward</span>
          </Link>
        </div>
      </div>
    );
  }

  return (
    <>
      <header>
        <h1>
          {typeof metadata.title === "string"
            ? metadata.title
            : metadata.title?.en || "Story"}
        </h1>
        <h2>Step 0: Watch the story video</h2>

        <div className="bg-gray-50 border border-gray-300 p-4 mb-4 rounded-lg text-center">
          <div className="flex items-start justify-center">
            <span className="material-icons text-gray-600 mr-2 mt-1">info</span>
            <div>
              <p className="text-gray-700">
                {metadata.description?.text ||
                  "Watch the video to get familiar with the story before listening."}
              </p>
            </div>
          </div>
        </div>
      </header>
      <div className="max-w-4xl mx-auto px-5">
        {videoWatched && (
          <div className="text-center mb-8">
            <Link
              to={`/stories/${id}/vocab`}
              className="inline-flex items-center px-8 py-4 bg-green-500 text-white rounded-lg hover:bg-green-600 text-lg font-semibold transition-all duration-200 shadow-lg"
            >
              <span>Continue to Vocabulary</span>
              <span className="material-icons ml-2">arrow_forward</span>
            </Link>
          </div>
        )}
        <div
          className="video-container"
          style={{
            width: "100%",
            maxWidth: "800px",
            margin: "0 auto",
            aspectRatio: "16/9",
          }}
        >
          {isYouTubeUrl(metadata.videoUrl) ? (
            <iframe
              src={getYouTubeEmbedUrl(metadata.videoUrl) || ""}
              allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
              allowFullScreen
              onLoad={() => {
                setTimeout(() => setVideoWatched(true), 30000);
              }}
              style={{
                width: "100%",
                height: "100%",
                borderRadius: "8px",
                border: "none",
              }}
            />
          ) : (
            <video
              src={metadata.videoUrl}
              controls
              onEnded={() => setVideoWatched(true)}
              onTimeUpdate={(e) => {
                const video = e.target as HTMLVideoElement;
                if (video.currentTime / video.duration > 0.8) {
                  setVideoWatched(true);
                }
              }}
              style={{
                width: "100%",
                height: "100%",
                borderRadius: "8px",
              }}
            >
              Your browser does not support the video tag.
            </video>
          )}
        </div>
      </div>
    </>
  );
}
