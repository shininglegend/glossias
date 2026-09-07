import Button from "~/components/ui/Button";

interface RedoStoryButtonProps {
  onRedo: () => void;
  redoing: boolean;
  error?: string | null;
}

export function RedoStoryButton({
  onRedo,
  redoing,
  error,
}: RedoStoryButtonProps) {
  return (
    <div>
      <Button
        type="button"
        variant="outline"
        size="lg"
        onClick={onRedo}
        disabled={redoing}
        className="h-auto px-8 py-4 text-lg font-semibold text-primary-700 border-2 border-primary-500 hover:bg-primary-50"
      >
        <span className="material-icons">replay</span>
        <span>{redoing ? "Starting over..." : "Do this story again"}</span>
      </Button>
      {error && <p className="text-red-600 text-sm mt-2">{error}</p>}
    </div>
  );
}
