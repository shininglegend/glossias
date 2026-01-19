import React from "react";
import type { StoryMetadata, GrammarPoint } from "../../types/admin";
import Input from "~/components/ui/Input";
import Textarea from "~/components/ui/Textarea";
import Label from "~/components/ui/Label";
import Button from "~/components/ui/Button";
import CourseSelector from "~/components/ui/CourseSelector";
type Props = {
  value: StoryMetadata;
  onSubmit: (metadata: StoryMetadata) => void;
  onHasChanges?: (hasChanges: boolean) => void;
  onResetSaveStatus?: () => void;
};

export default function MetadataForm({
  value,
  onSubmit,
  onHasChanges,
  onResetSaveStatus,
}: Props) {
  const [meta, setMeta] = React.useState<StoryMetadata>({
    ...value,
    grammarPoints: value.grammarPoints || [],
  });

  const nameInputRef = React.useRef<HTMLInputElement>(null);
  const descInputRef = React.useRef<HTMLInputElement>(null);

  const update = <K extends keyof StoryMetadata>(
    key: K,
    val: StoryMetadata[K],
  ) => {
    setMeta((m) => ({ ...m, [key]: val }));
    onHasChanges?.(true);
    onResetSaveStatus?.();
  };

  const updateTitle = (lang: string, val: string) => {
    setMeta((m) => ({ ...m, title: { ...m.title, [lang]: val } }));
    onHasChanges?.(true);
    onResetSaveStatus?.();
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSubmit(meta);
  };

  return (
    <form
      onSubmit={handleSubmit}
      className="grid grid-cols-1 md:grid-cols-2 gap-4"
    >
      <div>
        <Label>Week</Label>
        <Input
          type="number"
          value={meta.weekNumber}
          onChange={(e) => update("weekNumber", Number(e.target.value))}
        />
      </div>
      <div>
        <Label>Day Letter</Label>
        <Input
          value={meta.dayLetter}
          onChange={(e) => update("dayLetter", e.target.value)}
          maxLength={1}
        />
      </div>
      <div>
        <Label>Title (en)</Label>
        <Input
          value={meta.title["en"] || ""}
          onChange={(e) => updateTitle("en", e.target.value)}
        />
      </div>
      <div className="md:col-span-2">
        <Label>Grammar Points</Label>

        {/* Selected grammar points display */}
        <div className="flex flex-wrap gap-2 mb-3">
          {(meta.grammarPoints || []).map((gp, index) => (
            <div
              key={index}
              className="inline-flex items-center gap-2 px-3 py-1 bg-primary-100 text-primary-800 rounded-full text-sm"
            >
              <span>{gp.name}</span>
              <button
                type="button"
                onClick={() => {
                  const newGrammarPoints = (meta.grammarPoints || []).filter(
                    (_, i) => i !== index,
                  );
                  update("grammarPoints", newGrammarPoints);
                }}
                className="text-primary-600 hover:text-primary-800 font-bold"
              >
                ×
              </button>
            </div>
          ))}
        </div>

        {/* Add new grammar point form */}
        <div className="border rounded p-3 bg-gray-50">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-2 items-end">
            <div>
              <Label>Grammar Point Name</Label>
              <Input
                type="text"
                placeholder="e.g., Present Tense"
                ref={nameInputRef}
              />
            </div>
            <div>
              <Label>Description (Optional)</Label>
              <Input
                type="text"
                placeholder="Brief description"
                ref={descInputRef}
              />
            </div>
            <div>
              <Button
                type="button"
                onClick={() => {
                  const name = nameInputRef.current?.value?.trim();
                  if (!name) {
                    alert("Grammar point name is required");
                    return;
                  }

                  const newGrammarPoint = {
                    id: Date.now(), // Temporary ID for frontend
                    name,
                    description: descInputRef.current?.value?.trim() || "",
                  };

                  update("grammarPoints", [
                    ...(meta.grammarPoints || []),
                    newGrammarPoint,
                  ]);

                  // Clear inputs
                  if (nameInputRef.current) nameInputRef.current.value = "";
                  if (descInputRef.current) descInputRef.current.value = "";
                }}
                className="bg-green-600 hover:bg-green-700"
              >
                Add Grammar Point
              </Button>
            </div>
          </div>
        </div>

        {(!meta.grammarPoints || meta.grammarPoints.length === 0) && (
          <div className="text-sm text-red-500 mt-2">
            At least one grammar point is required
          </div>
        )}
      </div>
      <div>
        <Label>Author ID</Label>
        <Input
          value={meta.author.id}
          onChange={(e) =>
            update("author", { ...meta.author, id: e.target.value })
          }
        />
      </div>
      <div>
        <Label>Author Name</Label>
        <Input
          value={meta.author.name}
          onChange={(e) =>
            update("author", { ...meta.author, name: e.target.value })
          }
        />
      </div>
      <div>
        <CourseSelector
          value={meta.courseId}
          onChange={(courseId) => update("courseId", courseId)}
        />
      </div>
      <div>
        <Label>Video URL</Label>
        <Input
          value={meta.videoUrl || ""}
          onChange={(e) => update("videoUrl", e.target.value)}
          placeholder="https://..."
        />
      </div>
      <div className="md:col-span-2">
        <Label>Description Language</Label>
        <Input
          value={meta.languageCode}
          onChange={(e) => update("languageCode", e.target.value)}
        />
      </div>
      <div className="md:col-span-2">
        <Label>Description Text</Label>
        <Textarea
          rows={4}
          value={meta.description.text}
          onChange={(e) =>
            update("description", { ...meta.description, text: e.target.value })
          }
        />
      </div>
      <div className="md:col-span-2">
        <Button type="submit">Save Metadata</Button>
      </div>
    </form>
  );
}
