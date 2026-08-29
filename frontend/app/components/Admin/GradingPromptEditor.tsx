import React from "react";
import Button from "~/components/ui/Button";
import { Card } from "~/components/ui/Card";
import Label from "~/components/ui/Label";
import Textarea from "~/components/ui/Textarea";
import { useAuthenticatedFetch } from "../../lib/authFetch";

interface GradingPrompt {
  id: number;
  text: string;
  note?: string;
  createdBy?: string;
  createdAt: string;
  isDefault?: boolean;
}

interface GradingPromptPage {
  active: GradingPrompt;
  history: GradingPrompt[];
  default: string;
}

/**
 * Super-admin editor for the Produce AI grader's system prompt. Saving
 * appends a new version (the table is append-only); the newest version is
 * what every grading run uses from then on, and each row in
 * produce_grading_log records which version it ran with.
 */
export default function GradingPromptEditor() {
  const authenticatedFetch = useAuthenticatedFetch();
  const [page, setPage] = React.useState<GradingPromptPage | null>(null);
  const [text, setText] = React.useState("");
  const [note, setNote] = React.useState("");
  const [loading, setLoading] = React.useState(true);
  const [saving, setSaving] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);
  const [success, setSuccess] = React.useState<string | null>(null);
  const [showHistory, setShowHistory] = React.useState(false);

  const load = React.useCallback(async () => {
    try {
      const response = await authenticatedFetch(
        "/api/admin/system/grading-prompt",
      );
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }
      const data = (await response.json()) as GradingPromptPage;
      setPage(data);
      setText(data.active.text);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load prompt");
    } finally {
      setLoading(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  React.useEffect(() => {
    load();
  }, [load]);

  const changed = page !== null && text.trim() !== page.active.text.trim();

  const activate = async (id: number) => {
    setSaving(true);
    setError(null);
    setSuccess(null);
    try {
      const response = await authenticatedFetch(
        "/api/admin/system/grading-prompt/active",
        {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ id }),
        },
      );
      if (!response.ok) {
        const body = await response.text();
        throw new Error(body || `HTTP ${response.status}`);
      }
      const data = (await response.json()) as GradingPromptPage;
      setPage(data);
      setText(data.active.text);
      setSuccess(
        `Version ${data.active.id} is now active for new grading runs.`,
      );
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to activate");
    } finally {
      setSaving(false);
    }
  };

  const save = async () => {
    if (!changed) return;
    setSaving(true);
    setError(null);
    setSuccess(null);
    try {
      const response = await authenticatedFetch(
        "/api/admin/system/grading-prompt",
        {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ text, note }),
        },
      );
      if (!response.ok) {
        const body = await response.text();
        throw new Error(body || `HTTP ${response.status}`);
      }
      const data = (await response.json()) as GradingPromptPage;
      setPage(data);
      setText(data.active.text);
      setNote("");
      setSuccess(`Version  is now active for new grading runs.`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to save prompt");
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <Card className="p-6">
        <div className="text-sm text-slate-500">Loading grading prompt...</div>
      </Card>
    );
  }

  return (
    <Card className="p-6">
      <div className="flex items-start justify-between gap-4 mb-4">
        <div className="min-w-0 flex-1">
          <h2 className="text-xl font-semibold mb-2">Produce grading prompt</h2>
          <p className="text-slate-600 text-sm">
            The system prompt sent to the AI grader with every Produce
            submission (Hebrew passage, reference English, grammar point and the
            student's English go in the user message). Saving creates a new
            version — or, if the text matches an earlier version, makes that one
            active again. Every version is kept so each row in{" "}
            <code className="text-xs">produce_grading_log</code> can be traced
            to the exact prompt it ran with.
          </p>
        </div>
        {page && !page.active.isDefault && (
          <div className="text-xs text-slate-500 text-right shrink-0">
            <div>
              Active: <strong>v{page.active.id}</strong>
            </div>
            <div>{new Date(page.active.createdAt).toLocaleString()}</div>
            {page.active.note && (
              <div className="italic">{page.active.note}</div>
            )}
          </div>
        )}
      </div>

      {error && (
        <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded text-sm mb-4">
          {error}
        </div>
      )}
      {success && (
        <div className="bg-green-50 border border-green-200 text-green-700 px-4 py-3 rounded text-sm mb-4">
          {success}
        </div>
      )}

      <Textarea
        value={text}
        onChange={(event) => setText(event.target.value)}
        rows={18}
        className="font-mono text-xs leading-relaxed"
        spellCheck={false}
        aria-label="Grading system prompt"
      />

      <div className="mt-3 flex flex-col md:flex-row md:items-end gap-3">
        <div className="flex-1">
          <Label className="mb-1">Change note (optional)</Label>
          <input
            type="text"
            value={note}
            onChange={(event) => setNote(event.target.value)}
            placeholder="What changed and why"
            className="w-full rounded-md border border-slate-300 bg-white py-2 px-3 text-sm shadow-sm outline-none focus:border-primary-500 focus:ring-2 focus:ring-primary-200"
          />
        </div>
        <div className="flex gap-2">
          {page && text.trim() !== page.default.trim() && (
            <Button
              variant="outline"
              type="button"
              onClick={() => setText(page.default)}
              disabled={saving}
            >
              Load built-in default
            </Button>
          )}
          {changed && page && (
            <Button
              variant="outline"
              type="button"
              onClick={() => setText(page.active.text)}
              disabled={saving}
            >
              Discard changes
            </Button>
          )}
          <Button onClick={save} disabled={!changed || saving}>
            {saving ? "Saving..." : "Save and activate"}
          </Button>
        </div>
      </div>

      {page && page.history.length > 0 && (
        <div className="mt-6 border-t border-slate-200 pt-4">
          <button
            type="button"
            className="text-sm text-primary-600 hover:underline"
            onClick={() => setShowHistory((v) => !v)}
          >
            {showHistory ? "Hide" : "Show"} version history (
            {page.history.length})
          </button>
          {showHistory && (
            <ul className="mt-3 space-y-3">
              {page.history.map((version) => (
                <li
                  key={version.id}
                  className="rounded-md border border-slate-200 bg-slate-50 p-3 text-sm"
                >
                  <div className="flex flex-wrap items-center justify-between gap-2 mb-2 text-xs text-slate-500">
                    <span>
                      <strong className="text-slate-700">v{version.id}</strong>{" "}
                      · {new Date(version.createdAt).toLocaleString()}
                      {version.createdBy && ` · ${version.createdBy}`}
                      {version.note && (
                        <span className="italic"> · {version.note}</span>
                      )}
                    </span>
                    <span className="flex gap-3">
                      {version.id === page.active.id ? (
                        <span className="font-semibold text-green-700">
                          Active
                        </span>
                      ) : (
                        <button
                          type="button"
                          className="text-primary-600 hover:underline"
                          disabled={saving}
                          onClick={() => activate(version.id)}
                        >
                          Make active
                        </button>
                      )}
                      {version.text.trim() !== text.trim() && (
                        <button
                          type="button"
                          className="text-primary-600 hover:underline"
                          onClick={() => setText(version.text)}
                        >
                          Load into editor
                        </button>
                      )}
                    </span>
                  </div>
                  <pre className="whitespace-pre-wrap font-mono text-xs text-slate-700 max-h-48 overflow-y-auto">
                    {version.text}
                  </pre>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </Card>
  );
}
