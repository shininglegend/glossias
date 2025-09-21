import { Link, useLocation } from "react-router";
import Button from "~/components/ui/Button";

interface AdminStoryNavigationProps {
  storyId: string | number;
}

export default function AdminStoryNavigation({
  storyId,
}: AdminStoryNavigationProps) {
  const location = useLocation();
  const basePath = `/admin/stories/${storyId}`;

  const isActive = (path: string) => location.pathname === path;

  return (
    <div className="flex gap-2 mb-4">
      <Link to={basePath}>
        <Button variant={isActive(basePath) ? "primary" : "outline"} size="sm">
          Edit Story
        </Button>
      </Link>
      <Link to={`${basePath}/annotate`}>
        <Button
          variant={isActive(`${basePath}/annotate`) ? "primary" : "outline"}
          size="sm"
        >
          Annotate
        </Button>
      </Link>
      <Link to={`${basePath}/metadata`}>
        <Button
          variant={isActive(`${basePath}/metadata`) ? "primary" : "outline"}
          size="sm"
        >
          Metadata
        </Button>
      </Link>
      <Link to={`${basePath}/translate`}>
        <Button
          variant={isActive(`${basePath}/translate`) ? "primary" : "outline"}
          size="sm"
        >
          Translate
        </Button>
      </Link>
    </div>
  );
}
