import React from "react";
import Button from "~/components/ui/Button";
import Label from "~/components/ui/Label";
import Textarea from "~/components/ui/Textarea";
import { Card, CardContent } from "~/components/ui/Card";
import { useAdminApi } from "../../services/adminApi";
import ReadinessPanel from "./ReadinessPanel";
import type { ProducePage, GrammarPoint, StoryLine } from "../../types/admin";

interface ProduceEditorProps {
  storyId: number;
}

/**
 * Authors the Produce phase: two Hebrew passages drawn from the story, each
 * with its reference English and grammar point, plus the contrastive
 * explanation shown after both attempts. Students see the Hebrew highlighted
 * in the story and write the English.
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
  const [storyLines, setStoryLines] = React.useState<StoryLine[] | null>(null);
  const [translations, setTranslations] = React.useState<Map<
    number,
    string
  > | null>(null);

  // Story text feeds the line picker and the Hebrew passage; the English
  // translations seed the reference English. A failure in either shouldn't
  // block editing, so both are loaded separately and the picker/seeding falls back gracefully.
  React.useEffect(() => {
    let cancelled = false;
    adminApi
      .getStoryContent(storyId)
      .then((res) => {
        if (!cancelled) setStoryLines(res.story.content.lines);
      })
      .catch(() => {
        if (!cancelled) setStoryLines(null);
      });
    adminApi
      .getTranslations(storyId, "en")
      .then((rows) => {
        if (!cancelled) {
          setTranslations(
            new Map(rows.map((r) => [r.lineNumber, r.translationText])),
          );
        }
      })
      .catch(() => {
        if (!cancelled) setTranslations(null);
      });
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [storyId]);

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
        requirement={`Both segments need a Hebrew passage, a reference English translation, and a grammar point, and the story needs one contrastive explanation.`}
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
            storyLines={storyLines}
            translations={translations}
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
  /** Story text for the line picker; null while loading or if it failed. */
  storyLines: StoryLine[] | null;
  /** English line translations for seeding the prompt; null if unavailable. */
  translations: Map<number, string> | null;
  onChanged: () => Promise<void>;
}

/** Shortens a story line for the picker's option label. */
function lineLabel(line: StoryLine): string {
  const text = line.text.length > 60 ? `${line.text.slice(0, 60)}…` : line.text;
  return `${line.lineNumber}. ${text}`;
}

/** Joins a story's Hebrew text for lines start..end (inclusive). */
function hebrewForRange(
  storyLines: StoryLine[],
  start: number,
  end: number,
): string {
  return storyLines
    .filter((l) => l.lineNumber >= start && l.lineNumber <= end)
    .map((l) => l.text)
    .join("\n");
}

/**
 * Joins the English translations for lines start..end. Lines with no stored
 * translation are skipped rather than left as gaps.
 */
function englishForRange(
  translations: Map<number, string>,
  start: number,
  end: number,
): string {
  const parts: string[] = [];
  for (let n = start; n <= end; n++) {
    const t = translations.get(n);
    if (t) parts.push(t);
  }
  return parts.join("\n");
}

function SegmentCard({
  storyId,
  order,
  segment,
  grammarPoints,
  storyLines,
  translations,
  onChanged,
}: SegmentCardProps) {
  const adminApi = useAdminApi();
  const [referenceEnglish, setReferenceEnglish] = React.useState(
    segment?.referenceEnglish ?? "",
  );
  const [hebrewText, setHebrewText] = React.useState(segment?.hebrewText ?? "");
  const [grammarPointId, setGrammarPointId] = React.useState<number | "">(
    segment?.grammarPointId ?? "",
  );
  // The line range picker: what the author is currently choosing, which may
  // differ from the segment's saved range until "Replace" is clicked.
  const [lineStart, setLineStart] = React.useState<number | "">(
    segment?.lineStart ?? "",
  );
  const [lineEnd, setLineEnd] = React.useState<number | "">(
    segment?.lineEnd ?? "",
  );
  const [saving, setSaving] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  // Re-sync when a reload brings different content for this slot.
  React.useEffect(() => {
    setReferenceEnglish(segment?.referenceEnglish ?? "");
    setHebrewText(segment?.hebrewText ?? "");
    setGrammarPointId(segment?.grammarPointId ?? "");
    setLineStart(segment?.lineStart ?? "");
    setLineEnd(segment?.lineEnd ?? "");
  }, [
    segment?.referenceEnglish,
    segment?.hebrewText,
    segment?.grammarPointId,
    segment?.lineStart,
    segment?.lineEnd,
  ]);

  const changed =
    referenceEnglish !== (segment?.referenceEnglish ?? "") ||
    hebrewText !== (segment?.hebrewText ?? "") ||
    grammarPointId !== (segment?.grammarPointId ?? "") ||
    lineStart !== (segment?.lineStart ?? "") ||
    lineEnd !== (segment?.lineEnd ?? "");

  const rangeValid = lineStart !== "" && lineEnd !== "" && lineStart <= lineEnd;

  // Warn when a single-line passage isn't in that line verbatim: the student
  // page then highlights the whole line instead of just the passage. A
  // multi-line passage is expected not to match any one line, so the whole
  // range is highlighted by design and no warning applies.
  const chosenLine =
    rangeValid && lineStart === lineEnd
      ? storyLines?.find((l) => l.lineNumber === lineStart)
      : undefined;
  const referenceNotInLine =
    chosenLine !== undefined &&
    hebrewText.trim() !== "" &&
    !chosenLine.text.includes(hebrewText.trim());

  // Replace: (re)derive the Hebrew passage and reference English from the
  // range currently picked above, discarding whatever was there before.
  const replace = () => {
    if (!storyLines || !rangeValid) return;
    setHebrewText(hebrewForRange(storyLines, lineStart, lineEnd));
    if (translations) {
      setReferenceEnglish(englishForRange(translations, lineStart, lineEnd));
    }
  };

  // Sync: re-derive from the segment's already-saved line range, refreshing
  // stale text (e.g. the story or its translation changed since) without
  // requiring the author to re-pick lines.
  const sync = () => {
    if (!storyLines || segment?.lineStart == null || segment?.lineEnd == null)
      return;
    setLineStart(segment.lineStart);
    setLineEnd(segment.lineEnd);
    setHebrewText(
      hebrewForRange(storyLines, segment.lineStart, segment.lineEnd),
    );
    if (translations) {
      setReferenceEnglish(
        englishForRange(translations, segment.lineStart, segment.lineEnd),
      );
    }
  };

  const save = async () => {
    setSaving(true);
    setError(null);
    try {
      await adminApi.saveProduceSegment(storyId, order, {
        referenceEnglish,
        hebrewText,
        grammarPointId: grammarPointId === "" ? undefined : grammarPointId,
        lineStart: lineStart === "" ? undefined : lineStart,
        lineEnd: lineEnd === "" ? undefined : lineEnd,
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

        <div className="mt-4">
          <Label className="mb-1">Story lines</Label>
          <p className="text-xs text-slate-500 mb-1">
            Where this passage sits in the story. Pick the line range it spans,
            then Replace to fill in the Hebrew passage and reference English
            below from it; students see the passage highlighted (or the whole
            range, if it's more than a single line) and write the English right
            under it.
          </p>
          <div className="flex flex-wrap items-center gap-2">
            <select
              value={lineStart}
              onChange={(event) =>
                setLineStart(
                  event.target.value === "" ? "" : Number(event.target.value),
                )
              }
              dir="auto"
              className="flex-1 min-w-[10rem] rounded-md border border-slate-300 bg-white py-2 px-3 text-sm shadow-sm outline-none focus:border-primary-500 focus:ring-2 focus:ring-primary-200"
            >
              <option value="">From line…</option>
              {storyLines
                ? storyLines.map((line) => (
                    <option key={line.lineNumber} value={line.lineNumber}>
                      {lineLabel(line)}
                    </option>
                  ))
                : lineStart !== "" && (
                    <option value={lineStart}>Line {lineStart}</option>
                  )}
            </select>
            <span className="text-sm text-slate-500">to</span>
            <select
              value={lineEnd}
              onChange={(event) =>
                setLineEnd(
                  event.target.value === "" ? "" : Number(event.target.value),
                )
              }
              dir="auto"
              className="flex-1 min-w-[10rem] rounded-md border border-slate-300 bg-white py-2 px-3 text-sm shadow-sm outline-none focus:border-primary-500 focus:ring-2 focus:ring-primary-200"
            >
              <option value="">To line…</option>
              {storyLines
                ? storyLines.map((line) => (
                    <option key={line.lineNumber} value={line.lineNumber}>
                      {lineLabel(line)}
                    </option>
                  ))
                : lineEnd !== "" && (
                    <option value={lineEnd}>Line {lineEnd}</option>
                  )}
            </select>
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={!storyLines || !rangeValid}
              onClick={replace}
            >
              Replace
            </Button>
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={
                !storyLines ||
                segment?.lineStart == null ||
                segment?.lineEnd == null
              }
              onClick={sync}
            >
              Sync
            </Button>
          </div>
          {lineStart !== "" && lineEnd !== "" && !rangeValid && (
            <p className="text-xs text-rose-700 mt-1">
              "To line" must be the same as or after "From line".
            </p>
          )}
        </div>

        <div className="mt-4">
          <Label className="mb-1">Hebrew passage</Label>
          <p className="text-xs text-slate-500 mb-1">
            What the student translates. Trim to the passage itself if the lines
            hold more than that.
          </p>
          <Textarea
            value={hebrewText}
            onChange={(event) => setHebrewText(event.target.value)}
            rows={3}
            dir="rtl"
            className="text-right"
            placeholder={
              storyLines ? "Pick story lines above..." : "הקטע מהסיפור..."
            }
          />
          {referenceNotInLine && (
            <p className="text-xs text-amber-700 mt-1">
              This passage doesn't appear word-for-word in line {lineStart}, so
              students will see the whole line highlighted rather than just the
              passage.
            </p>
          )}
        </div>

        <div className="mt-4">
          <Label className="mb-1">Reference English</Label>
          <p className="text-xs text-slate-500 mb-1">
            Revealed beside the student's answer after they submit, and used by
            the AI grader as one good translation.
          </p>
          <Textarea
            value={referenceEnglish}
            onChange={(event) => setReferenceEnglish(event.target.value)}
            rows={3}
            placeholder="The English the passage means..."
          />
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
