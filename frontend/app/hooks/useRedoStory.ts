import { useCallback, useState } from "react";
import { useNavigate } from "react-router";
import { useApiService } from "../services/api";
import { useNavigationGuidance } from "./useNavigationGuidance";

/**
 * Clears the student's exercise answers for a story and opens Identify, the
 * start of the sequence. Video time is left on the server.
 */
export function useRedoStory(storyId: string | undefined) {
  const api = useApiService();
  const navigate = useNavigate();
  const { clearCache } = useNavigationGuidance();
  const [redoing, setRedoing] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const redo = useCallback(async () => {
    if (!storyId) return;
    const confirmed = window.confirm(
      "Start the exercises over from Identify? Your first-attempt score is kept. You can still watch the video.",
    );
    if (!confirmed) return;

    setRedoing(true);
    setError(null);
    try {
      const response = await api.resetOwnStoryProgress(storyId);
      if (!response.success) {
        setError(response.error || "Failed to restart the story");
        return;
      }
      clearCache();
      navigate(`/stories/${storyId}/identify`);
    } catch {
      setError("Failed to restart the story");
    } finally {
      setRedoing(false);
    }
  }, [api, clearCache, navigate, storyId]);

  return { redo, redoing, error };
}
