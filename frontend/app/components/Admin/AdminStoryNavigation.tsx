import { Link, useLocation } from "react-router";
import Button from "~/components/ui/Button";
import { useStoryReadiness } from "~/contexts/StoryReadinessContext";

interface AdminStoryNavigationProps {
  storyId: string | number;
}

// The first three editors cover a story's shared content; the last three author
// the Summer 2026 phases (target vocabulary for Identify, produce segments, and
// recall sentences).
const EDITORS = [
  { path: "annotate", label: "Annotate" },
  { path: "metadata", label: "Metadata", phase: "video" },
  { path: "translate", label: "Translate" },
  { path: "target-vocab", label: "Target Vocab", phase: "identify" },
  { path: "produce", label: "Produce", phase: "produce" },
  { path: "recall", label: "Recall", phase: "recall" },
] as const;

export default function AdminStoryNavigation({
  storyId,
}: AdminStoryNavigationProps) {
  const location = useLocation();
  const { readiness } = useStoryReadiness();
  const basePath = `/admin/stories/${storyId}`;

  return (
    <div className="flex flex-wrap gap-2 mb-4">
      {EDITORS.map((editor) => {
        const { path, label } = editor;
        const to = `${basePath}/${path}`;
        const phase =
          "phase" in editor && readiness ? readiness[editor.phase] : null;
        const incomplete = phase ? !phase.ready : false;
        return (
          <Link key={path} to={to}>
            <Button
              variant={location.pathname === to ? "primary" : "outline"}
              size="sm"
              title={
                incomplete
                  ? `Incomplete: ${phase!.issues.map((i) => i.message).join("; ")}`
                  : undefined
              }
              icon={
                incomplete ? (
                  <span
                    className="material-icons text-sm text-amber-500"
                    aria-label="Incomplete"
                  >
                    warning
                  </span>
                ) : undefined
              }
            >
              {label}
            </Button>
          </Link>
        );
      })}
    </div>
  );
}
