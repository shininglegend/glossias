import { Link } from "react-router";
import React from "react";
import Button from "~/components/ui/Button";
import Input from "~/components/ui/Input";
import { Card } from "~/components/ui/Card";
import Badge from "~/components/ui/Badge";

import { useAdminApi } from "../services/adminApi";
import { useAuthenticatedFetch } from "../lib/authFetch";

type StoryListItem = {
  id: number;
  title: string;
  week_number: number; // keeping backend field name
  day_letter: string;
};

export default function AdminHome() {
  const adminApi = useAdminApi();
  const authenticatedFetch = useAuthenticatedFetch();
  const [stories, setStories] = React.useState<StoryListItem[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [processing, setProcessing] = React.useState(false);
  const [query, setQuery] = React.useState("");
  React.useEffect(() => {
    async function fetchStories() {
      try {
        const res = await authenticatedFetch("/api/stories", {
          headers: { Accept: "application/json" },
        });
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const json = await res.json();
        setStories(json.data.stories);
      } catch (error) {
        console.error("Failed to fetch stories:", error);
      } finally {
        setLoading(false);
      }
    }
    fetchStories();
  }, []);

  const handleAction = async (intent: string, storyId: number) => {
    setProcessing(true);
    try {
      if (intent === "clear-annotations") {
        await adminApi.clearAnnotations(storyId);
      } else if (intent === "delete") {
        await adminApi.deleteStory(storyId);
        setStories((prev) => prev.filter((s) => s.id !== storyId));
      }
    } catch (error) {
      console.error("Action failed:", error);
    } finally {
      setProcessing(false);
    }
  };

  const filtered = React.useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return stories;
    return stories.filter(
      (s) =>
        (s.title || "").toLowerCase().includes(q) || String(s.id).includes(q),
    );
  }, [stories, query]);

  return (
    <main className="mx-auto max-w-6xl px-4 py-6">
      <div className="flex flex-col gap-6">
        <div className="flex items-end justify-between gap-4">
          <div>
            <h1 className="text-3xl font-semibold tracking-tight">
              Admin Dashboard
            </h1>
            <p className="text-sm text-slate-500">
              Manage stories, metadata, and annotations
            </p>
          </div>
          <Link to="/admin/stories/add">
            <Button
              icon={<span className="material-icons text-base">add</span>}
            >
              Add New Story
            </Button>
          </Link>
        </div>

        <div className="flex items-center justify-between gap-4">
          <div className="relative w-full sm:w-96">
            <span className="material-icons pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-slate-400">
              search
            </span>
            <Input
              placeholder="Search by title or ID…"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              className="pl-10"
            />
          </div>
          <div className="text-xs text-slate-500">
            {filtered.length} of {stories.length}
          </div>
        </div>

        {loading ? (
          <div className="text-center py-8">Loading stories...</div>
        ) : (
          <ul className="grid gap-4 sm:grid-cols-2">
            {filtered.map((s) => (
              <li key={s.id}>
                <Card className="p-4">
                  <div className="mb-3 flex items-start justify-between gap-3">
                    <div className="min-w-0">
                      <Link
                        to={`/admin/stories/${s.id}`}
                        className="block truncate font-medium text-slate-900 hover:underline"
                      >
                        {s.title || `Story #${s.id}`}
                      </Link>
                      <div className="mt-1 text-xs text-slate-500">
                        Week {s.week_number}
                        {(s.day_letter || "").toUpperCase()}
                      </div>
                    </div>
                    <Badge>#{s.id}</Badge>
                  </div>

                  <div className="flex flex-wrap gap-2">
                    <Link to={`/admin/stories/${s.id}`}>
                      <Button
                        variant="outline"
                        size="sm"
                        icon={
                          <span className="material-icons text-sm">
                            data_object
                          </span>
                        }
                      >
                        Edit JSON
                      </Button>
                    </Link>
                    <Link to={`/admin/stories/${s.id}/metadata`}>
                      <Button
                        variant="outline"
                        size="sm"
                        icon={
                          <span className="material-icons text-sm">
                            description
                          </span>
                        }
                      >
                        Metadata
                      </Button>
                    </Link>
                    <Link to={`/admin/stories/${s.id}/annotate`}>
                      <Button
                        variant="outline"
                        size="sm"
                        icon={
                          <span className="material-icons text-sm">edit</span>
                        }
                      >
                        Annotate
                      </Button>
                    </Link>
                    <Button
                      onClick={() => {
                        if (confirm("Clear all annotations?")) {
                          handleAction("clear-annotations", s.id);
                        }
                      }}
                      variant="warning"
                      size="sm"
                      icon={
                        <span className="material-icons text-sm">
                          layers_clear
                        </span>
                      }
                      disabled={processing}
                    >
                      {processing ? "Clearing…" : "Clear"}
                    </Button>
                    <Button
                      onClick={() => {
                        if (
                          confirm("Delete this story? This cannot be undone.")
                        ) {
                          handleAction("delete", s.id);
                        }
                      }}
                      variant="danger"
                      size="sm"
                      icon={
                        <span className="material-icons text-sm">delete</span>
                      }
                      disabled={processing}
                    >
                      {processing ? "Deleting…" : "Delete"}
                    </Button>
                  </div>
                </Card>
              </li>
            ))}
          </ul>
        )}
      </div>
    </main>
  );
}
