import { useNavigate } from "react-router";
import Button from "~/components/ui/Button";

interface RedoStoryButtonProps {
  storyId: string;
}

/**
 * Opens Identify to start the next attempt. Finishing a story already
 * archived the score and cleared the exercises server-side, so this is a
 * plain navigation with nothing to confirm.
 */
export function RedoStoryButton({ storyId }: RedoStoryButtonProps) {
  const navigate = useNavigate();
  return (
    <Button
      type="button"
      variant="outline"
      size="lg"
      onClick={() => navigate(`/stories/${storyId}/identify`)}
      className="h-auto px-8 py-4 text-lg font-semibold text-primary-700 border-2 border-primary-500 hover:bg-primary-50"
    >
      <span className="material-icons">replay</span>
      <span>Do this story again</span>
    </Button>
  );
}
