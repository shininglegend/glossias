import { useEffect, useState, type ReactNode } from "react";
import AdminStoryNavigation from "./AdminStoryNavigation";
import { useAdminApi } from "~/services/adminApi";
import type { StoryContentReadiness } from "~/types/admin";

interface AdminStoryPageProps {
  storyId: string | number;
  title: string;
  /** One line explaining what this editor controls. Always rendered so the
   *  header keeps the same height across editors and the nav doesn't jump. */
  description: string;
  /** Buttons or status text shown to the right of the title. */
  actions?: ReactNode;
  children: ReactNode;
}

// Shared chrome for every /admin/stories/:id/* editor: identical container,
// header block, and phase navigation so switching editors keeps the content
// anchored in the same place.
export default function AdminStoryPage({
  storyId,
  title,
  description,
  actions,
  children,
}: AdminStoryPageProps) {
  const adminApi = useAdminApi();
  const [storyTitle, setStoryTitle] = useState<string | null>(null);
  const [readiness, setReadiness] = useState<StoryContentReadiness | null>(
    null,
  );

  useEffect(() => {
    let cancelled = false;
    adminApi
      .getMetadata(Number(storyId))
      .then((data) => {
        if (!cancelled) setStoryTitle(data.story.metadata.title?.en ?? null);
      })
      .catch(() => {
        if (!cancelled) setStoryTitle(null);
      });
    adminApi
      .getContentReadiness(Number(storyId))
      .then((r) => {
        if (!cancelled) setReadiness(r);
      })
      .catch(() => {
        if (!cancelled) setReadiness(null);
      });
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [storyId]);

  return (
    <main className="container mx-auto p-6">
      <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <div className="mb-1 flex flex-wrap items-baseline gap-x-3">
            <h1 className="text-2xl font-bold">{title}</h1>
            <span className="text-lg text-slate-500 min-h-7">
              {storyTitle && `"${storyTitle}"`}
            </span>
          </div>
          <p className="text-sm text-slate-600 min-h-5">{description}</p>
        </div>
        {actions && (
          <div className="flex items-center gap-2 sm:shrink-0">{actions}</div>
        )}
      </div>
      <AdminStoryNavigation storyId={storyId} readiness={readiness} />
      {children}
    </main>
  );
}
