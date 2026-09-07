import type { ReactNode } from "react";
import { Link, useNavigate, useParams } from "react-router";
import { cn } from "~/lib/cn";
import {
  STORY_FLOW_PHASES,
  phaseByPath,
  type StoryPhasePath,
} from "~/lib/storyPhases";
import Button from "../ui/Button";

export function StoryShell({
  phasePath,
  storyTitle,
  loading = false,
  error = null,
  navError = null,
  children,
}: {
  phasePath: StoryPhasePath;
  storyTitle?: string;
  loading?: boolean;
  error?: string | null;
  navError?: string | null;
  children?: ReactNode;
}) {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const phase = phaseByPath(phasePath);

  return (
    <div className="w-full">
      <div className="flex items-center gap-3 mb-4">
        <Link
          to="/"
          className="inline-flex items-center gap-1 text-sm font-medium text-slate-600 hover:text-slate-900"
        >
          <span className="material-icons text-base" aria-hidden="true">
            home
          </span>
          Stories
        </Link>
      </div>

      <nav aria-label="Story phases" className="mb-6">
        <ol className="flex gap-2 overflow-x-auto pb-1">
          {STORY_FLOW_PHASES.map((item) => {
            const current = item.path === phasePath;
            return (
              <li key={item.path} className="shrink-0">
                <Link
                  to={id ? `/stories/${id}/${item.path}` : "/"}
                  aria-current={current ? "page" : undefined}
                  className={cn(
                    "inline-flex items-center gap-1.5 rounded-full px-3 py-1.5 text-sm font-medium ring-1 transition-colors",
                    current
                      ? cn(
                          item.theme.barMuted,
                          item.theme.title,
                          item.theme.ring,
                        )
                      : "bg-slate-50 text-slate-500 ring-slate-200 hover:bg-slate-100 hover:text-slate-800",
                  )}
                >
                  <span
                    className={cn(
                      "material-icons text-base",
                      current ? item.theme.icon : "text-slate-400",
                    )}
                    aria-hidden="true"
                  >
                    {item.icon}
                  </span>
                  {item.title}
                </Link>
              </li>
            );
          })}
        </ol>
      </nav>

      {loading ? (
        <div className="flex justify-center items-center min-h-[40vh]">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-500" />
        </div>
      ) : error ? (
        <div className="max-w-xl mx-auto mt-6 p-6 bg-red-50 border border-red-200 rounded-lg text-center">
          <p className="text-red-700 font-bold mb-2">Error Loading Phase</p>
          <p className="text-red-600 mb-4">{error}</p>
          <Button variant="outline" onClick={() => navigate("/")}>
            Back to Stories
          </Button>
        </div>
      ) : (
        <>
          {(storyTitle || phase) && (
            <header>
              {storyTitle ? <h1>{storyTitle}</h1> : null}
              {phase ? (
                <h2 className={phase.theme.title}>{phase.title}</h2>
              ) : null}
            </header>
          )}
          {navError ? (
            <p className="text-center text-red-600 mb-4" role="alert">
              {navError}
            </p>
          ) : null}
          {children}
        </>
      )}
    </div>
  );
}
