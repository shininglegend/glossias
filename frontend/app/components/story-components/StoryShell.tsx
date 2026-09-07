import type { ReactNode } from "react";
import { Link, useNavigate, useParams } from "react-router";
import { cn } from "~/lib/cn";
import {
  STORY_BAR_PHASES,
  phaseByPath,
  phaseHref,
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
    <div className="w-full min-w-0">
      <nav aria-label="Story phases" className="mb-5 w-full">
        <ol className="flex w-full overflow-x-auto rounded-lg border border-slate-200 bg-white shadow-sm">
          {STORY_BAR_PHASES.map((item, i) => {
            const current = item.path === phasePath;
            return (
              <li
                key={item.path}
                className={cn(
                  "flex min-w-9 flex-1 overflow-hidden",
                  i > 0 && "border-l border-slate-200",
                  i === 0 && "rounded-l-lg",
                  i === STORY_BAR_PHASES.length - 1 && "rounded-r-lg",
                )}
              >
                <Link
                  to={id ? phaseHref(id, item.path) : "/"}
                  aria-current={current ? "page" : undefined}
                  className={cn(
                    "relative flex flex-1 items-center justify-center gap-1 px-1.5 py-2 text-xs font-medium transition-colors",
                    current
                      ? cn(item.theme.barMuted, item.theme.title)
                      : "text-slate-500 hover:bg-slate-50 hover:text-slate-800",
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
                  <span
                    className={cn(
                      "truncate",
                      current ? "inline" : "hidden xl:inline",
                    )}
                  >
                    {item.title}
                  </span>
                  {current ? (
                    <span
                      className={cn(
                        "absolute inset-x-0 bottom-0 h-0.5",
                        item.theme.bar,
                      )}
                    />
                  ) : null}
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
