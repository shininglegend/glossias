import { Link, useLocation } from "react-router";
import Button from "~/components/ui/Button";

interface AdminStoryNavigationProps {
  storyId: string | number;
}

// The first three editors cover a story's shared content; the last three author
// the Summer 2026 phases (target vocabulary for Identify, produce segments, and
// recall sentences).
const EDITORS = [
  { path: "annotate", label: "Annotate" },
  { path: "metadata", label: "Metadata" },
  { path: "translate", label: "Translate" },
  { path: "target-vocab", label: "Target Vocab" },
  { path: "produce", label: "Produce" },
  { path: "recall", label: "Recall" },
] as const;

export default function AdminStoryNavigation({
  storyId,
}: AdminStoryNavigationProps) {
  const location = useLocation();
  const basePath = `/admin/stories/${storyId}`;

  return (
    <div className="flex flex-wrap gap-2 mb-4">
      {EDITORS.map(({ path, label }) => {
        const to = `${basePath}/${path}`;
        return (
          <Link key={path} to={to}>
            <Button
              variant={location.pathname === to ? "primary" : "outline"}
              size="sm"
            >
              {label}
            </Button>
          </Link>
        );
      })}
    </div>
  );
}
