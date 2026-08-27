import React from "react";
import Button from "~/components/ui/Button";
import Label from "~/components/ui/Label";
import Textarea from "~/components/ui/Textarea";
import { Card, CardContent } from "~/components/ui/Card";
import { useAdminApi } from "../../services/adminApi";
import ReadinessPanel from "./ReadinessPanel";
import type { ProducePage, GrammarPoint } from "../../types/admin";

interface ProduceEditorProps {
  storyId: number;
}

/**
 * Authors the Produce phase: two English prompts with their reference Hebrew and
 * grammar point, plus the contrastive explanation shown after both attempts.
 *
 * The two segments are addressed by position rather than by row ID, so the slots
 * stay put across saves and either can be filled in first.
 */
export default function ProduceEditor({ storyId }: ProduceEditorProps) {
  const adminApi = useAdminApi();
  const [page, setPage] = React.useState<ProducePage | null>(null);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);
  const [explanation, setExplanation] = React.useState("");
  const [savingExplanation, setSavingExplanation] = React.useState(false);

  const load = React.useCallback(async () => {
    try {
      const data = await adminApi.getProduce(storyId);
      setPage(data);
      setExplanation(data.explanation);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load");
    } finally {
      setLoading(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [storyId]);

  React.useEffect(() => {
    load();
  }, [load]);

  const saveExplanation = async () => {
    setSavingExplanation(true);
    setError(null);
    try {
      if (explanation.trim() === "") {
        await adminApi.deleteProduceExplanation(storyId);
      } else {
        await adminApi.saveProduceExplanation(storyId, explanation);
      }
      await load();
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to save explanation",
      );
    } finally {
      setSavingExplanation(false);
    }
  };

  if (loading) {
    return <div className="text-center py-8">Loading produce content...</div>;
  }

  if (!page) {
    return (
      <div className="text-center py-8 text-rose-700">
        {error ?? "Failed to load produce content"}
      </div>
    );
  }

  const slots = Array.from({ length: page.required }, (_, index) => index + 1);
  const explanationChanged = explanation !== page.explanation;

  return (
    <div>
      <ReadinessPanel
        readiness={page.readiness}
        requirement={`Both segments need an English prompt, a reference translation, and a grammar point, and the story needs one contrastive explanation.`}
      />

      {error && (
        <div className="bg-rose-50 border-l-4 border-rose-400 p-3 mb-4 rounded text-sm text-rose-800">
          {error}
        </div>
      )}

      {page.grammarPoints.length === 0 && (
        <div className="bg-blue-50 border-l-4 border-blue-400 p-3 mb-4 rounded text-sm text-blue-900">
          This story has no grammar points yet. Add them in the Metadata editor
          — a segment needs one so AI grading and the explanation popup have
          context.
        </div>
      )}

      <div className="space-y-4 mb-8">
        {slots.map((order) => (
          <SegmentCard
            key={order}
            storyId={storyId}
            order={order}
            segment={page.segments.find((s) => s.segmentOrder === order)}
            grammarPoints={page.grammarPoints}
            onChanged={load}
          />
        ))}
      </div>

      <Card>
        <CardContent>
          <Label className="mb-1">Contrastive grammar explanation</Label>
          <p className="text-xs text-slate-500 mb-2">
            Shown in a popup after both segments: how the grammar point works in
            each one, what it contributes, and how the two compare.
          </p>
          <Textarea
            value={explanation}
            onChange={(event) => setExplanation(event.target.value)}
            rows={6}
            placeholder="Explain how the grammar point works in each segment..."
          />
          <div className="flex items-center gap-2 mt-2">
            <Button
              onClick={saveExplanation}
              disabled={!explanationChanged || savingExplanation}
            >
              {savingExplanation ? "Saving..." : "Save explanation"}
            </Button>
            {explanation.trim() === "" && page.explanation !== "" && (
              <span className="text-xs text-slate-500">
                Saving an empty explanation removes it.
              </span>
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

interface SegmentCardProps {
  storyId: number;
  order: number;
  segment: ProducePage["segments"][number] | undefined;
  grammarPoints: GrammarPoint[];
  onChanged: () => Promise<void>;
}

function SegmentCard({
  storyId,
  order,
  segment,
  grammarPoints,
  onChanged,
}: SegmentCardProps) {
  const adminApi = useAdminApi();
  const [englishText, setEnglishText] = React.useState(
    segment?.englishText ?? "",
  );
  const [referenceHebrew, setReferenceHebrew] = React.useState(
    segment?.referenceHebrew ?? "",
  );
  const [grammarPointId, setGrammarPointId] = React.useState<number | "">(
    segment?.grammarPointId ?? "",
  );
  const [saving, setSaving] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  // Re-sync when a reload brings different content for this slot.
  React.useEffect(() => {
    setEnglishText(segment?.englishText ?? "");
    setReferenceHebrew(segment?.referenceHebrew ?? "");
    setGrammarPointId(segment?.grammarPointId ?? "");
  }, [segment?.englishText, segment?.referenceHebrew, segment?.grammarPointId]);

  const changed =
    englishText !== (segment?.englishText ?? "") ||
    referenceHebrew !== (segment?.referenceHebrew ?? "") ||
    grammarPointId !== (segment?.grammarPointId ?? "");

  const save = async () => {
    setSaving(true);
    setError(null);
    try {
      await adminApi.saveProduceSegment(storyId, order, {
        englishText,
        referenceHebrew,
        grammarPointId: grammarPointId === "" ? undefined : grammarPointId,
      });
      await onChanged();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to save segment");
    } finally {
      setSaving(false);
    }
  };

  const remove = async () => {
    if (
      !window.confirm(
        `Delete segment ${order}? Any student attempts at it are deleted too.`,
      )
    ) {
      return;
    }

    setSaving(true);
    setError(null);
    try {
      await adminApi.deleteProduceSegment(storyId, order);
      await onChanged();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to delete segment");
    } finally {
      setSaving(false);
    }
  };

  return (
    <Card>
      <CardContent>
        <div className="flex items-center justify-between mb-3">
          <h3 className="font-semibold">Segment {order}</h3>
          {segment && (
            <Button
              variant="danger"
              size="sm"
              disabled={saving}
              onClick={remove}
            >
              Delete
            </Button>
          )}
        </div>

        {error && <p className="text-sm text-rose-700 mb-2">{error}</p>}

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <Label className="mb-1">English prompt</Label>
            <Textarea
              value={englishText}
              onChange={(event) => setEnglishText(event.target.value)}
              rows={3}
              placeholder="The English the student renders into Hebrew..."
            />
          </div>

          <div>
            <Label className="mb-1">Reference Hebrew</Label>
            <Textarea
              value={referenceHebrew}
              onChange={(event) => setReferenceHebrew(event.target.value)}
              rows={3}
              dir="rtl"
              className="text-right"
              placeholder="התרגום לדוגמה..."
            />
          </div>
        </div>

        <div className="mt-4 flex flex-col md:flex-row md:items-end gap-3">
          <div className="flex-1">
            <Label className="mb-1">Grammar point</Label>
            <select
              value={grammarPointId}
              onChange={(event) =>
                setGrammarPointId(
                  event.target.value === "" ? "" : Number(event.target.value),
                )
              }
              className="w-full rounded-md border border-slate-300 bg-white py-2 px-3 text-sm shadow-sm outline-none focus:border-primary-500 focus:ring-2 focus:ring-primary-200"
            >
              <option value="">No grammar point</option>
              {grammarPoints.map((point) => (
                <option key={point.id} value={point.id}>
                  {point.name}
                </option>
              ))}
            </select>
          </div>

          <Button onClick={save} disabled={!changed || saving}>
            {saving ? "Saving..." : "Save segment"}
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}
