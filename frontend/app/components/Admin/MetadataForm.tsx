import React from "react";
import type { StoryMetadata } from "../../types/admin";
import Input from "~/components/ui/Input";
import Textarea from "~/components/ui/Textarea";
import Label from "~/components/ui/Label";
import Button from "~/components/ui/Button";
import CourseSelector from "~/components/ui/CourseSelector";

type Props = {
  value: StoryMetadata;
  onSubmit: (metadata: StoryMetadata) => void;
};

export default function MetadataForm({ value, onSubmit }: Props) {
  const [meta, setMeta] = React.useState<StoryMetadata>(value);

  const update = <K extends keyof StoryMetadata>(
    key: K,
    val: StoryMetadata[K],
  ) => setMeta((m) => ({ ...m, [key]: val }));

  const updateTitle = (lang: string, val: string) =>
    setMeta((m) => ({ ...m, title: { ...m.title, [lang]: val } }));

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
          required
        />
      </div>
      <div>
        <Label>Day Letter</Label>
        <Input
          value={meta.dayLetter}
          onChange={(e) => update("dayLetter", e.target.value)}
          maxLength={1}
          required
        />
      </div>
      <div>
        <Label>Title (en)</Label>
        <Input
          value={meta.title["en"] || ""}
          onChange={(e) => updateTitle("en", e.target.value)}
        />
      </div>
      <div>
        <Label>Grammar Point</Label>
        <Input
          value={meta.grammarPoint}
          onChange={(e) => update("grammarPoint", e.target.value)}
        />
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
      <div className="md:col-span-2">
        <Label>Description Language</Label>
        <Input
          value={meta.description.language}
          onChange={(e) =>
            update("description", {
              ...meta.description,
              language: e.target.value,
            })
          }
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
