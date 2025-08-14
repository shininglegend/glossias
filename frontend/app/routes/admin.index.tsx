import { useLoaderData, Link, Form, useNavigation } from "react-router";
import React from "react";
import Button from "~/components/ui/Button";
import Input from "~/components/ui/Input";
import { Card } from "~/components/ui/Card";
import Badge from "~/components/ui/Badge";

import { type ActionFunctionArgs, type LoaderFunctionArgs } from "react-router";
import { clearAnnotations, deleteStory } from "../services/adminApi";

type StoryListItem = {
  id: number;
  title: string;
  week_number: number; // keeping backend field name
  day_letter: string;
};

export async function loader({ request }: LoaderFunctionArgs) {
  const normalize = (b: string) => b.replace(/\/$/, "");
  const toAbs = (b: string) =>
    b.startsWith("http")
      ? b
      : new URL(normalize(b), new URL(request.url)).toString();

  async function fetchJSON(u: string) {
    const url = new URL(request.url);
    const baseUrl = `${url.protocol}//${url.host}`;
    const fullUrl = u.startsWith("http") ? u : `${baseUrl}${u}`;
    const res = await fetch(fullUrl, {
      headers: { Accept: "application/json" },
    });
    if (!res.ok) throw new Error(`HTTP ${res.status} from ${u}`);
    const ct = res.headers.get("content-type") || "";
    const body = await res.text();
    if (!ct.includes("application/json")) {
      if (body.startsWith("<!DOCTYPE") || body.includes("<html")) {
        throw new Error(
          `Received HTML from ${fullUrl}. Check that the backend is running on localhost:8080.`
        );
      }
      throw new Error(
        `Unexpected content-type: ${ct} from ${fullUrl}: ${body.slice(0, 120)}…`
      );
    }
    return JSON.parse(body);
  }

  const storiesUrl = "/api/stories";
  const json = await fetchJSON(storiesUrl);
  return json.data.stories as StoryListItem[];
}

export async function action({ request }: ActionFunctionArgs) {
  const data = await request.formData();
  const intent = data.get("intent");
  const storyId = Number(data.get("storyId"));
  const url = new URL(request.url);
  const baseUrl = `${url.protocol}//${url.host}`;
  if (intent === "clear-annotations" && storyId) {
    await clearAnnotations(storyId, baseUrl);
    return { success: true } as const;
  }
  if (intent === "delete" && storyId) {
    await deleteStory(storyId, baseUrl);
    return { success: true } as const;
  }
  return {} as const;
}

export default function AdminHome() {
  const stories = useLoaderData() as StoryListItem[];
  const nav = useNavigation();
  const [query, setQuery] = React.useState("");
  const filtered = React.useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return stories;
    return stories.filter(
      (s) =>
        (s.title || "").toLowerCase().includes(q) || String(s.id).includes(q)
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
                  <Form
                    method="post"
                    replace
                    className="inline-block"
                    onSubmit={(e) => {
                      if (!confirm("Clear all annotations?")) {
                        e.preventDefault();
                      }
                    }}
                  >
                    <input type="hidden" name="storyId" value={s.id} />
                    <Button
                      name="intent"
                      value="clear-annotations"
                      variant="warning"
                      size="sm"
                      icon={
                        <span className="material-icons text-sm">
                          layers_clear
                        </span>
                      }
                      disabled={nav.state === "submitting"}
                    >
                      {nav.state === "submitting" ? "Clearing…" : "Clear"}
                    </Button>
                  </Form>
                  <Form
                    method="post"
                    replace
                    className="inline-block"
                    onSubmit={(e) => {
                      if (
                        !confirm("Delete this story? This cannot be undone.")
                      ) {
                        e.preventDefault();
                      }
                    }}
                  >
                    <input type="hidden" name="storyId" value={s.id} />
                    <Button
                      name="intent"
                      value="delete"
                      variant="danger"
                      size="sm"
                      icon={
                        <span className="material-icons text-sm">delete</span>
                      }
                      disabled={nav.state === "submitting"}
                    >
                      {nav.state === "submitting" ? "Deleting…" : "Delete"}
                    </Button>
                  </Form>
                </div>
              </Card>
            </li>
          ))}
        </ul>
      </div>
    </main>
  );
}
