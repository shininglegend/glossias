import React from "react";
import Button from "~/components/ui/Button";
import Label from "~/components/ui/Label";
import Textarea from "~/components/ui/Textarea";
import { Card, CardContent } from "~/components/ui/Card";
import { useAdminApi } from "../../services/adminApi";
import { usePhaseAssetUploader } from "../../lib/phaseAssets";
import ReadinessPanel from "./ReadinessPanel";
import AssetSlot from "./AssetSlot";
import RecallSentencePicker, {
  splitStoryIntoSentences,
  type StorySentence,
} from "./RecallSentencePicker";
import type {
  RecallPage,
  RecallSentence,
  StoryContent,
  TargetVocabulary,
} from "../../types/admin";

interface RecallEditorProps {
  storyId: number;
}

/**
 * Authors the five Recall sentences in story order, one per target word.
 *
 * The order shown here is the correct sequence; the student endpoint shuffles it
 * and withholds the positions. Each sentence is addressed by its position, so the
 * five slots stay put across saves.
 */
export default function RecallEditor({ storyId }: RecallEditorProps) {
  const adminApi = useAdminApi();
  const [page, setPage] = React.useState<RecallPage | null>(null);
  const [storyContent, setStoryContent] = React.useState<StoryContent | null>(
    null,
  );
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);

  const load = React.useCallback(async () => {
    try {
      setPage(await adminApi.getRecall(storyId));
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

  // Story text only feeds the sentence picker; a failure here shouldn't block
  // editing, so it's loaded separately and silently falls back to typing.
  React.useEffect(() => {
    let cancelled = false;
    adminApi
      .getStoryContent(storyId)
      .then((res) => {
        if (!cancelled) setStoryContent(res.story.content);
      })
      .catch(() => {
        if (!cancelled) setStoryContent(null);
      });
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [storyId]);

  const storySentences = React.useMemo(
    () => splitStoryIntoSentences(storyContent),
    [storyContent],
  );

  if (loading) {
    return <div className="text-center py-8">Loading recall sentences...</div>;
  }

  if (!page) {
    return (
      <div className="text-center py-8 text-rose-700">
        {error ?? "Failed to load recall sentences"}
      </div>
    );
  }

  const slots = Array.from({ length: page.required }, (_, index) => index + 1);

  // Which position already uses each sentence, so the picker can flag repeats.
  const usedByPosition = new Map<string, number>();
  for (const sentence of page.sentences) {
    usedByPosition.set(sentence.hebrewText.trim(), sentence.sequenceOrder);
  }

  return (
    <div>
      <ReadinessPanel
        readiness={page.readiness}
        requirement={`All ${page.required} positions need a sentence, a picture, and its own target word — the order below is the correct one students have to reconstruct.`}
      />

      {error && (
        <div className="bg-rose-50 border-l-4 border-rose-400 p-3 mb-4 rounded text-sm text-rose-800">
          {error}
        </div>
      )}

      {page.targetVocabulary.length === 0 && (
        <div className="bg-blue-50 border-l-4 border-blue-400 p-3 mb-4 rounded text-sm text-blue-900">
          This story has no target words yet. Choose them in the Target Vocab
          editor first — each recall sentence links to one of them.
        </div>
      )}

      <div className="space-y-4">
        {slots.map((order) => (
          <RecallSentenceCard
            key={order}
            storyId={storyId}
            order={order}
            sentence={page.sentences.find((s) => s.sequenceOrder === order)}
            targetVocabulary={page.targetVocabulary}
            storySentences={storySentences}
            usedByPosition={usedByPosition}
            usedTargetVocabIds={page.sentences
              .filter((s) => s.sequenceOrder !== order && s.targetVocabId)
              .map((s) => s.targetVocabId as number)}
            onChanged={load}
          />
        ))}
      </div>
    </div>
  );
}

interface RecallSentenceCardProps {
  storyId: number;
  order: number;
  sentence: RecallSentence | undefined;
  targetVocabulary: TargetVocabulary[];
  /** The story split into sentences, for the picker. */
  storySentences: StorySentence[];
  /** Sentence text -> position already using it. */
  usedByPosition: Map<string, number>;
  /** Target words already spoken for by another position. */
  usedTargetVocabIds: number[];
  onChanged: () => Promise<void>;
}

function RecallSentenceCard({
  storyId,
  order,
  sentence,
  targetVocabulary,
  storySentences,
  usedByPosition,
  usedTargetVocabIds,
  onChanged,
}: RecallSentenceCardProps) {
  const adminApi = useAdminApi();
  const uploadAsset = usePhaseAssetUploader();
  const [hebrewText, setHebrewText] = React.useState(
    sentence?.hebrewText ?? "",
  );
  const [targetVocabId, setTargetVocabId] = React.useState<number | "">(
    sentence?.targetVocabId ?? "",
  );
  const [saving, setSaving] = React.useState(false);
  const [uploading, setUploading] = React.useState(false);
  const [pickerOpen, setPickerOpen] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  // Re-sync when a reload brings different content for this slot.
  React.useEffect(() => {
    setHebrewText(sentence?.hebrewText ?? "");
    setTargetVocabId(sentence?.targetVocabId ?? "");
  }, [sentence?.hebrewText, sentence?.targetVocabId]);

  const changed =
    hebrewText !== (sentence?.hebrewText ?? "") ||
    targetVocabId !== (sentence?.targetVocabId ?? "");

  const save = async () => {
    setSaving(true);
    setError(null);
    try {
      await adminApi.saveRecallSentence(storyId, order, {
        hebrewText,
        targetVocabId: targetVocabId === "" ? undefined : targetVocabId,
      });
      await onChanged();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to save sentence");
    } finally {
      setSaving(false);
    }
  };

  const remove = async () => {
    if (!window.confirm(`Delete the sentence at position ${order}?`)) return;

    setSaving(true);
    setError(null);
    try {
      await adminApi.deleteRecallSentence(storyId, order);
      await onChanged();
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to delete sentence",
      );
    } finally {
      setSaving(false);
    }
  };

  const attachImage = async (file: File | null) => {
    if (!sentence) {
      setError("Save the sentence before uploading its picture.");
      return;
    }

    setUploading(true);
    setError(null);
    try {
      // An empty path clears the slot; a real upload returns the path to attach.
      const imagePath = file
        ? await uploadAsset(file, storyId, "recall_image", sentence.id)
        : "";

      await adminApi.saveRecallSentence(storyId, order, {
        hebrewText: sentence.hebrewText,
        targetVocabId: sentence.targetVocabId,
        imagePath,
      });
      await onChanged();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Upload failed");
    } finally {
      setUploading(false);
    }
  };

  return (
    <Card>
      <CardContent>
        <div className="flex items-center justify-between mb-3">
          <h3 className="font-semibold">Position {order}</h3>
          {sentence && (
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
            <div className="flex items-center justify-between mb-1">
              <Label>Sentence (Hebrew)</Label>
              <Button
                variant="secondary"
                size="sm"
                type="button"
                disabled={storySentences.length === 0}
                onClick={() => setPickerOpen(true)}
              >
                Pick from story
              </Button>
            </div>
            <RecallSentencePicker
              isOpen={pickerOpen}
              onClose={() => setPickerOpen(false)}
              sentences={storySentences}
              usedByPosition={usedByPosition}
              currentOrder={order}
              onPick={setHebrewText}
            />
            <Textarea
              value={hebrewText}
              onChange={(event) => setHebrewText(event.target.value)}
              rows={3}
              dir="rtl"
              className="text-right"
              placeholder="המשפט..."
            />

            <Label className="mb-1 mt-3">Target word</Label>
            <select
              value={targetVocabId}
              onChange={(event) =>
                setTargetVocabId(
                  event.target.value === "" ? "" : Number(event.target.value),
                )
              }
              className="w-full rounded-md border border-slate-300 bg-white py-2 px-3 text-sm shadow-sm outline-none focus:border-primary-500 focus:ring-2 focus:ring-primary-200"
            >
              <option value="">No target word</option>
              {targetVocabulary.map((word) => (
                <option key={word.id} value={word.id}>
                  {word.lexicalForm}
                  {usedTargetVocabIds.includes(word.id)
                    ? " (used elsewhere)"
                    : ""}
                </option>
              ))}
            </select>

            <Button
              className="mt-3"
              onClick={save}
              disabled={!changed || saving}
            >
              {saving ? "Saving..." : sentence ? "Save sentence" : "Create"}
            </Button>
          </div>

          <div>
            {sentence ? (
              <AssetSlot
                label="Picture"
                accept="image/*"
                uploading={uploading}
                hasAsset={Boolean(sentence.imagePath)}
                onSelect={(file) => attachImage(file)}
                onClear={() => attachImage(null)}
                preview={
                  sentence.imageUrl ? (
                    <img
                      src={sentence.imageUrl}
                      alt={`Picture for recall position ${order}`}
                      className="max-h-32 rounded border border-slate-200"
                    />
                  ) : null
                }
              />
            ) : (
              <div>
                <Label className="mb-1">Picture</Label>
                <p className="text-xs text-slate-400">
                  Save the sentence first, then upload its picture.
                </p>
              </div>
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
