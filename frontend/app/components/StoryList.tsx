import { useState, useEffect, useMemo } from "react";
import { useNavigate } from "react-router";
import { useApiService } from "../services/api";
import { useNavigationGuidance } from "../hooks/useNavigationGuidance";
import { useUserContext } from "../contexts/UserContext";
import type { Story } from "../services/api";
import Button from "./ui/Button";
import { Card, CardContent } from "./ui/Card";

function StoryCard({
  story,
  loading,
  onOpen,
}: {
  story: Story;
  loading: boolean;
  onOpen: () => void;
}) {
  return (
    <Card className="hover:shadow-md transition-shadow">
      <CardContent className="p-5 flex flex-col gap-4">
        <div>
          <h3 className="text-lg font-semibold text-slate-900 leading-snug">
            {story.title}
          </h3>
          <p className="mt-1 text-sm text-slate-500">
            Week {story.week_number}
            {story.day_letter}
          </p>
        </div>
        <Button
          className="self-start"
          onClick={onOpen}
          disabled={loading}
          icon={
            loading ? (
              <span
                className="animate-spin w-4 h-4 border-2 border-white border-t-transparent rounded-full"
                aria-hidden="true"
              />
            ) : undefined
          }
        >
          {loading ? "Loading..." : "Start Reading"}
        </Button>
      </CardContent>
    </Card>
  );
}

function StoryGrid({
  stories,
  loadingStory,
  onOpen,
}: {
  stories: Story[];
  loadingStory: number | null;
  onOpen: (id: number) => void;
}) {
  return (
    <div className="grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
      {stories.map((story) => (
        <StoryCard
          key={story.id}
          story={story}
          loading={loadingStory === story.id}
          onOpen={() => onOpen(story.id)}
        />
      ))}
    </div>
  );
}

/** Collapsed group of stories (upcoming / archived) using a native <details>. */
function CollapsedSection({
  title,
  stories,
  loadingStory,
  onOpen,
}: {
  title: string;
  stories: Story[];
  loadingStory: number | null;
  onOpen: (id: number) => void;
}) {
  if (stories.length === 0) return null;
  return (
    <details className="group mb-8">
      <summary className="list-none cursor-pointer select-none flex items-center justify-between rounded-lg border border-slate-200 bg-white px-4 py-3 text-slate-700 font-medium hover:bg-slate-50 transition-colors [&::-webkit-details-marker]:hidden">
        <span>
          {title} ({stories.length})
        </span>
        <span
          className="material-icons text-slate-400 transition-transform group-open:rotate-180"
          aria-hidden="true"
        >
          expand_more
        </span>
      </summary>
      <div className="mt-4">
        <StoryGrid
          stories={stories}
          loadingStory={loadingStory}
          onOpen={onOpen}
        />
      </div>
    </details>
  );
}

export function StoryList() {
  const api = useApiService();
  const navigate = useNavigate();
  const { getNavigationGuidance } = useNavigationGuidance();
  const { userInfo } = useUserContext();
  const [stories, setStories] = useState<Story[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [loadingStory, setLoadingStory] = useState<number | null>(null);

  // Group stories by course status
  const groupedStories = useMemo(() => {
    if (!userInfo?.enrolled_courses) {
      return { active: stories, past: [], future: [] };
    }

    const courseStatusMap = new Map(
      userInfo.enrolled_courses.map((c) => [c.course_id, c.status]),
    );

    const active: Story[] = [];
    const past: Story[] = [];
    const future: Story[] = [];

    stories.forEach((story) => {
      const status = story.course_id
        ? courseStatusMap.get(story.course_id)
        : "active";
      if (status === "past") {
        past.push(story);
      } else if (status === "future") {
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
      } catch {
        setError("Failed to fetch stories");
      } finally {
        setLoading(false);
      }
    };

    fetchStories();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

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

  return (
    <div className="w-full max-w-5xl mx-auto px-4 py-10">
      <div className="mb-8">
        <h1 className="text-3xl font-bold tracking-tight text-slate-900">
          Your Stories
        </h1>
        <p className="mt-1 text-slate-600">Select a story to begin reading</p>
      </div>

      {loading ? (
        <p className="text-slate-600">Loading stories...</p>
      ) : error ? (
        <p className="text-rose-700">Error: {error}</p>
      ) : stories.length === 0 ? (
        <Card className="max-w-xl">
          <CardContent className="p-6">
            <h2 className="text-xl font-semibold text-slate-900 mb-2">
              Welcome!
            </h2>
            <p className="text-slate-600">
              You're in! Please wait to be registered for a course so you can
              access some stories.
            </p>
          </CardContent>
        </Card>
      ) : (
        <>
          {groupedStories.active.length > 0 && (
            <section className="mb-10">
              <h2 className="text-xl font-semibold text-slate-900 mb-4">
                Current Stories
              </h2>
              <StoryGrid
                stories={groupedStories.active}
                loadingStory={loadingStory}
                onOpen={handleStoryClick}
              />
            </section>
          )}

          <CollapsedSection
            title="Upcoming Stories"
            stories={groupedStories.future}
            loadingStory={loadingStory}
            onOpen={handleStoryClick}
          />

          <CollapsedSection
            title="Archived Stories"
            stories={groupedStories.past}
            loadingStory={loadingStory}
            onOpen={handleStoryClick}
          />
        </>
      )}
    </div>
  );
}
