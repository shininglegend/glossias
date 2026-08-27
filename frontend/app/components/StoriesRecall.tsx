import { useState, useEffect } from "react";
import { useParams, useNavigate, Link } from "react-router";
import { useApiService } from "../services/api";
import { useNavigationGuidance } from "../hooks/useNavigationGuidance";
import { CompletionMessage } from "./story-components/CompletionMessage";
import type { StoryMetadata } from "../types/api";

export function StoriesRecall() {
  const { id } = useParams<{ id: string }>();
  const api = useApiService();
  const navigate = useNavigate();
  const { getNavigationGuidance } = useNavigationGuidance();
  const [metadata, setMetadata] = useState<StoryMetadata | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [nextStepName, setNextStepName] = useState<string>("Next Step");

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

        const guidance = await getNavigationGuidance(id, "recall");
        if (guidance) {
          setNextStepName(guidance.displayName);
        }
      } catch (err) {
        setError("Failed to fetch story metadata");
      } finally {
        setLoading(false);
      }
    };

    fetchMetadata();
  }, [id, getNavigationGuidance, api]);

  const handleContinue = async () => {
    if (!id) return;
    try {
      const guidance = await getNavigationGuidance(id, "recall");
      if (guidance) {
        navigate(`/stories/${id}/${guidance.nextPage}`);
      }
    } catch (err) {
      console.error("Failed to navigate to next phase:", err);
    }
  };

  if (loading) {
    return (
      <div className="flex justify-center items-center min-h-[50vh]">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-500"></div>
      </div>
    );
  }

  if (error || !metadata) {
    return (
      <div className="container max-w-xl mx-auto mt-10 p-6 bg-red-50 border border-red-200 rounded-lg text-center">
        <h2 className="text-red-700 font-bold mb-2">Error Loading Phase</h2>
        <p className="text-red-600 mb-4">{error || "Could not retrieve story details."}</p>
        <Link to="/" className="text-primary-600 hover:text-primary-700 underline font-medium">
          Back to Stories
        </Link>
      </div>
    );
  }

  const storyTitle = typeof metadata.title === "string" 
    ? metadata.title 
    : metadata.title?.en || "Story";

  return (
    <div className="max-w-4xl mx-auto px-5 py-8">
      <header className="mb-8 text-center">
        <span className="inline-block px-3 py-1 bg-primary-50 text-primary-700 rounded-full text-xs font-semibold uppercase tracking-wider mb-3">
          Phase 5 of 5
        </span>
        <h1 className="text-3xl font-extrabold text-gray-900 tracking-tight sm:text-4xl mb-2">
          {storyTitle}
        </h1>
        <h2 className="text-lg font-medium text-gray-500">Recall Phase</h2>
      </header>

      <div className="bg-white shadow-xl rounded-2xl overflow-hidden border border-gray-100 max-w-2xl mx-auto">
        <div className="p-8">
          <div className="flex items-center justify-center w-16 h-16 bg-orange-50 text-orange-600 rounded-2xl mx-auto mb-6">
            <span className="material-icons text-3xl">history</span>
          </div>

          <div className="text-center mb-8">
            <h3 className="text-xl font-bold text-gray-900 mb-3">
              Story Sequencing & Recall
            </h3>
            <p className="text-gray-600 leading-relaxed max-w-md mx-auto">
              In this phase, you will listen to the story narration with no text shown. 
              Once the audio completes, you will drag-and-drop the key Hebrew sentences to place them back into their correct chronological order.
            </p>
          </div>

          <div className="bg-slate-50 rounded-xl p-5 border border-slate-100 mb-8">
            <div className="flex items-center gap-3 mb-2 text-slate-700 font-semibold text-sm">
              <span className="material-icons text-slate-500 text-lg">build</span>
              <span>Scaffolding Mode Active</span>
            </div>
            <p className="text-slate-600 text-sm leading-relaxed">
              This phase is currently in placeholder/scaffolding mode. Click the button below to record completion and advance to the next step.
            </p>
          </div>

          <CompletionMessage
            currentStepName="recall"
            nextStepName={nextStepName}
            onContinue={handleContinue}
          />
        </div>
      </div>
    </div>
  );
}
