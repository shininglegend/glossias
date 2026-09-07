import { useState, useEffect } from "react";
import { useParams, useNavigate, useSearchParams } from "react-router";
import { useApiService } from "../services/api";
import { useNavigationGuidance } from "../hooks/useNavigationGuidance";
import { CompletionMessage } from "./story-components/CompletionMessage";
import { StoryShell } from "./story-components/StoryShell";
import Button from "./ui/Button";
import type { StoryMetadata } from "../services/api";
import type { NavigationGuidanceResponse } from "../types/api";

/** Shown once the student has an archived attempt, since finishing clears the live progress. */
function ScoresButton({ onClick }: { onClick: () => void }) {
  return (
    <Button
      type="button"
      variant="outline"
      size="lg"
      onClick={onClick}
      className="h-auto px-8 py-4 text-lg font-semibold text-primary-700 border-2 border-primary-500 hover:bg-primary-50"
    >
      <span className="material-icons">emoji_events</span>
      <span>View your scores</span>
    </Button>
  );
}

function getYouTubeEmbedUrl(url: string): string | null {
  const regex = /(?:youtube\.com\/watch\?v=|youtu\.be\/)([a-zA-Z0-9_-]{11})/;
  const match = url.match(regex);
  // autoplay=1 is honored because the embed is only mounted after the
  // student clicks "Start video", which counts as a user gesture.
  return match ? `https://www.youtube.com/embed/${match[1]}?autoplay=1` : null;
}

function isYouTubeUrl(url: string): boolean {
  return url.includes("youtube.com") || url.includes("youtu.be");
}

function storyTitle(metadata: StoryMetadata | null): string | undefined {
  if (!metadata) return undefined;
  return typeof metadata.title === "string"
    ? metadata.title
    : metadata.title?.en || "Story";
}

export function StoriesVideo() {
  const { id } = useParams<{ id: string }>();
  const [searchParams, setSearchParams] = useSearchParams();
  const api = useApiService();
  const navigate = useNavigate();
  const { getNavigationGuidance } = useNavigationGuidance();
  const [metadata, setMetadata] = useState<StoryMetadata | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [navError, setNavError] = useState<string | null>(null);
  const videoStarted = searchParams.has("watch");
  const [videoWatched, setVideoWatched] = useState(false);
  const [nextStepName, setNextStepName] = useState<string>("Next Step");
  const [guidanceCache, setGuidanceCache] =
    useState<NavigationGuidanceResponse | null>(null);
  const [iframeLoaded, setIframeLoaded] = useState(false);
  const hasScores = (guidanceCache?.completedAttempts ?? 0) > 0;
  const goToScores = () => navigate(`/stories/${id}/score`);

  useEffect(() => {
    const fetchMetadataAndGuidance = async () => {
      if (!id) {
        setError("Story ID is required");
        setLoading(false);
        return;
      }

      try {
        const [metadataResponse, guidance] = await Promise.all([
          api.getStoryMetadata(id),
          getNavigationGuidance(id, "video"),
        ]);

        if (metadataResponse.success && metadataResponse.data) {
          setMetadata(metadataResponse.data);
        } else {
          setError(metadataResponse.error || "Failed to fetch story metadata");
        }

        if (guidance) {
          setNextStepName(guidance.displayName);
          setGuidanceCache(guidance);
        }
      } catch {
        setError("Failed to fetch story metadata");
      } finally {
        setLoading(false);
      }
    };

    fetchMetadataAndGuidance();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id]);

  const goToNextStep = async () => {
    try {
      const guidance =
        guidanceCache || (await getNavigationGuidance(id!, "video"));
      if (guidance) {
        navigate(`/stories/${id}/${guidance.nextPage}`);
        return;
      }
      setNavError("Couldn't open the next phase.");
    } catch (err) {
      console.error("Failed to get navigation guidance:", err);
      setNavError("Couldn't open the next phase.");
    }
  };

  const title = storyTitle(metadata);
  const summary = metadata?.description?.text || "";
  const videoUrl = metadata?.videoUrl;
  const isYouTube = videoUrl ? isYouTubeUrl(videoUrl) : false;
  // Direct <video> reports progress, so Continue is gated on it. YouTube
  // embeds can't report progress without the IFrame API, so leave ungated.
  const canContinue = isYouTube || videoWatched;

  return (
    <StoryShell
      phasePath={videoStarted ? "video" : "intro"}
      storyTitle={title}
      loading={loading}
      error={error || (!loading && !metadata ? "No story found" : null)}
      navError={navError}
    >
      {metadata && !videoUrl ? (
        <div className="text-center flex flex-col items-center gap-4">
          <p>No video available for this story</p>
          <Button
            type="button"
            size="lg"
            onClick={goToNextStep}
            className="h-auto px-8 py-4 text-lg font-semibold"
          >
            Skip to {nextStepName}
            <span className="material-icons">arrow_forward</span>
          </Button>
          {hasScores && <ScoresButton onClick={goToScores} />}
        </div>
      ) : metadata && !videoStarted ? (
        <div className="max-w-2xl mx-auto px-5 text-center">
          {summary ? (
            <p className="text-xl leading-relaxed text-gray-800 mb-8">
              {summary}
            </p>
          ) : (
            <p className="text-lg text-gray-600 mb-8">
              Watch the video to get familiar with the story before the other
              exercises.
            </p>
          )}
          <div className="flex flex-col items-center gap-4">
            <Button
              type="button"
              size="lg"
              onClick={() => setSearchParams({ watch: "1" }, { replace: true })}
              className="h-auto px-8 py-4 text-lg font-semibold"
              icon={<span className="material-icons">play_arrow</span>}
            >
              Start video
            </Button>
            {hasScores && <ScoresButton onClick={goToScores} />}
          </div>
        </div>
      ) : metadata && videoUrl ? (
        <div className="max-w-4xl mx-auto px-5">
          <div
            className="video-container"
            style={{
              width: "100%",
              maxWidth: "800px",
              margin: "0 auto",
              aspectRatio: "16/9",
              position: "relative",
            }}
          >
            {isYouTube ? (
              <>
                {!iframeLoaded && (
                  <div
                    className="absolute inset-0 flex items-center justify-center bg-gray-100 rounded-lg"
                    style={{
                      borderRadius: "8px",
                    }}
                  >
                    <div className="text-center">
                      <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-gray-900 mx-auto mb-2"></div>
                      <p className="text-gray-600">Loading video...</p>
                    </div>
                  </div>
                )}
                <iframe
                  src={getYouTubeEmbedUrl(videoUrl) || ""}
                  allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
                  allowFullScreen
                  onLoad={() => setIframeLoaded(true)}
                  style={{
                    width: "100%",
                    height: "100%",
                    borderRadius: "8px",
                    border: "none",
                    opacity: iframeLoaded ? 1 : 0,
                    transition: "opacity 0.3s ease-in-out",
                  }}
                />
              </>
            ) : (
              <video
                src={videoUrl}
                controls
                autoPlay
                onEnded={() => setVideoWatched(true)}
                onTimeUpdate={(e) => {
                  const video = e.target as HTMLVideoElement;
                  if (
                    video.duration &&
                    video.currentTime / video.duration > 0.8
                  ) {
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
          {canContinue ? (
            <>
              <CompletionMessage
                currentStepName="video"
                nextStepName={nextStepName}
                onContinue={goToNextStep}
              />
              {hasScores && (
                <div className="text-center -mt-4 mb-10">
                  <ScoresButton onClick={goToScores} />
                </div>
              )}
            </>
          ) : (
            <div className="text-center m-10 p-6 bg-gray-50 rounded-xl border-2 border-yellow-400">
              <div className="flex items-start justify-center">
                <span className="material-icons text-gray-600 mr-2 mt-1">
                  info
                </span>
                <p className="text-gray-700 m-0">
                  <strong>
                    Please watch the entire video before continuing.
                  </strong>{" "}
                  The continue button will appear once the video is nearly
                  finished.
                </p>
              </div>
            </div>
          )}
        </div>
      ) : null}
    </StoryShell>
  );
}
