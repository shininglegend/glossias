import React from "react";
import Button from "~/components/ui/Button";
import Badge from "~/components/ui/Badge";
import Label from "~/components/ui/Label";
import { Card, CardContent } from "~/components/ui/Card";
import { useAdminApi } from "../../services/adminApi";
import { usePhaseAssetUploader } from "../../lib/phaseAssets";
import ReadinessPanel from "./ReadinessPanel";
import AssetSlot from "./AssetSlot";
import type { TargetVocabulary, TargetVocabularyPage } from "../../types/admin";

interface TargetVocabEditorProps {
  storyId: number;
}

/**
 * Authors the five target words the Identify and Recall phases are built on.
 *
 * A word can only be chosen from lexical forms already annotated in the story
 * text, and only when it appears often enough for Identify to pause on it more
 * than once — the backend enforces both, and the candidate list shows the counts
 * so the author does not have to guess.
 */
export default function TargetVocabEditor({ storyId }: TargetVocabEditorProps) {
  const adminApi = useAdminApi();
  const [page, setPage] = React.useState<TargetVocabularyPage | null>(null);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);
  const [busyWordId, setBusyWordId] = React.useState<number | null>(null);
  const [adding, setAdding] = React.useState(false);

  const load = React.useCallback(async () => {
    try {
      setPage(await adminApi.getTargetVocabulary(storyId));
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

  const addWord = async (lexicalForm: string) => {
    setAdding(true);
    setError(null);
    try {
      await adminApi.addTargetWord(storyId, lexicalForm);
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to add word");
    } finally {
      setAdding(false);
    }
  };

  const removeWord = async (word: TargetVocabulary) => {
    if (
      !window.confirm(
        `Remove "${word.lexicalForm}" as a target word? Its audio and picture ` +
          `will be deleted, and any recall sentence linked to it will lose that link.`,
      )
    ) {
      return;
    }

    setBusyWordId(word.id);
    setError(null);
    try {
      await adminApi.deleteTargetWord(storyId, word.id);
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to remove word");
    } finally {
      setBusyWordId(null);
    }
  };

  if (loading) {
    return <div className="text-center py-8">Loading target vocabulary...</div>;
  }

  if (!page) {
    return (
      <div className="text-center py-8 text-rose-700">
        {error ?? "Failed to load target vocabulary"}
      </div>
    );
  }

  const chosen = new Set(page.words.map((word) => word.lexicalForm));
  const eligible = page.candidates.filter(
    (candidate) =>
      candidate.occurrences >= page.minOccurrences &&
      !chosen.has(candidate.lexicalForm),
  );
  const atCapacity = page.words.length >= page.required;

  return (
    <div>
      <ReadinessPanel
        readiness={page.readiness}
        requirement={`Each of the ${page.required} target words needs a pronunciation recording, a picture, and at least ${page.minOccurrences} annotated occurrences in the story text.`}
      />

      {error && (
        <div className="bg-rose-50 border-l-4 border-rose-400 p-3 mb-4 rounded text-sm text-rose-800">
          {error}
        </div>
      )}

      <div className="space-y-4 mb-8">
        {page.words.map((word) => (
          <TargetWordCard
            key={word.id}
            storyId={storyId}
            word={word}
            occurrences={
              page.candidates.find((c) => c.lexicalForm === word.lexicalForm)
                ?.occurrences ?? 0
            }
            minOccurrences={page.minOccurrences}
            busy={busyWordId === word.id}
            onChanged={load}
            onRemove={() => removeWord(word)}
          />
        ))}

        {page.words.length === 0 && (
          <p className="text-sm text-slate-500">
            No target words yet. Pick {page.required} from the annotated words
            below.
          </p>
        )}
      </div>

      <Card>
        <CardContent>
          <Label className="mb-2">
            Add a target word ({page.words.length} of {page.required} chosen)
          </Label>

          {atCapacity ? (
            <p className="text-sm text-slate-500">
              All {page.required} target words are chosen. Remove one to swap it
              out.
            </p>
          ) : eligible.length === 0 ? (
            <p className="text-sm text-slate-500">
              No eligible words left. A word must be annotated as vocabulary in
              at least {page.minOccurrences} places in the story text before it
              can be a target word — add those annotations in the Annotate
              editor first.
            </p>
          ) : (
            <div className="flex flex-wrap gap-2">
              {eligible.map((candidate) => (
                <Button
                  key={candidate.lexicalForm}
                  variant="outline"
                  size="sm"
                  disabled={adding}
                  onClick={() => addWord(candidate.lexicalForm)}
                >
                  <span dir="rtl">{candidate.lexicalForm}</span>
                  <span className="ml-2 text-slate-400">
                    ×{candidate.occurrences}
                  </span>
                </Button>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

interface TargetWordCardProps {
  storyId: number;
  word: TargetVocabulary;
  occurrences: number;
  minOccurrences: number;
  busy: boolean;
  onChanged: () => Promise<void>;
  onRemove: () => void;
}

function TargetWordCard({
  storyId,
  word,
  occurrences,
  minOccurrences,
  busy,
  onChanged,
  onRemove,
}: TargetWordCardProps) {
  const adminApi = useAdminApi();
  const uploadAsset = usePhaseAssetUploader();
  const [assetError, setAssetError] = React.useState<string | null>(null);
  const [uploading, setUploading] = React.useState<"audio" | "image" | null>(
    null,
  );

  const attach = async (slot: "audio" | "image", file: File | null) => {
    setAssetError(null);
    setUploading(slot);
    try {
      // An empty path clears the slot; a real upload returns the path to attach.
      const filePath = file
        ? await uploadAsset(
            file,
            storyId,
            slot === "audio" ? "target_vocab_audio" : "target_vocab_image",
            word.id,
          )
        : "";

      await adminApi.saveTargetWord(
        storyId,
        word.id,
        slot === "audio" ? { audioPath: filePath } : { imagePath: filePath },
      );
      await onChanged();
    } catch (err) {
      setAssetError(err instanceof Error ? err.message : "Upload failed");
    } finally {
      setUploading(null);
    }
  };

  return (
    <Card>
      <CardContent>
        <div className="flex items-start justify-between gap-4 mb-3">
          <div>
            <div className="text-2xl" dir="rtl">
              {word.lexicalForm}
            </div>
            <div className="mt-1">
              {occurrences >= minOccurrences ? (
                <Badge variant="success">
                  appears {occurrences}× in the story
                </Badge>
              ) : (
                <Badge variant="danger">
                  appears {occurrences}× — needs at least {minOccurrences}
                </Badge>
              )}
            </div>
          </div>
          <Button variant="danger" size="sm" disabled={busy} onClick={onRemove}>
            {busy ? "Removing..." : "Remove"}
          </Button>
        </div>

        {assetError && (
          <p className="text-sm text-rose-700 mb-2">{assetError}</p>
        )}

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <AssetSlot
            label="Pronunciation"
            accept="audio/*"
            uploading={uploading === "audio"}
            hasAsset={Boolean(word.audioPath)}
            onSelect={(file) => attach("audio", file)}
            onClear={() => attach("audio", null)}
            preview={
              word.audioUrl ? (
                <audio controls src={word.audioUrl} className="w-full" />
              ) : null
            }
          />

          <AssetSlot
            label="Picture"
            accept="image/*"
            uploading={uploading === "image"}
            hasAsset={Boolean(word.correctImagePath)}
            onSelect={(file) => attach("image", file)}
            onClear={() => attach("image", null)}
            preview={
              word.imageUrl ? (
                <img
                  src={word.imageUrl}
                  alt={`Picture for ${word.lexicalForm}`}
                  className="max-h-32 rounded border border-slate-200"
                />
              ) : null
            }
          />
        </div>
      </CardContent>
    </Card>
  );
}
