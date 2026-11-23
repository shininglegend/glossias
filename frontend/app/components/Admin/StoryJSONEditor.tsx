import React from "react";
import type { Story } from "../../types/admin";
import Textarea from "~/components/ui/Textarea";
import Button from "~/components/ui/Button";

type Props = {
  value: Story;
  // A rejected promise is shown under the editor (e.g. the 409 the server
  // returns for a story that already has lines).
  onSubmit: (story: Story) => void | Promise<void>;
};

export default function StoryJSONEditor({ value, onSubmit }: Props) {
  const [text, setText] = React.useState<string>(
    JSON.stringify(value, null, 2),
  );
  const [error, setError] = React.useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    let parsed: Story;
    try {
      parsed = JSON.parse(text) as Story;
    } catch (e) {
      setError(e instanceof Error ? e.message : "Invalid JSON");
      return;
    }
    setError(null);
    try {
      await onSubmit(parsed);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to save story");
    }
  };

  return (
    <form onSubmit={handleSubmit}>
      <Textarea
        className="font-mono text-sm"
        value={text}
        onChange={(e) => setText(e.target.value)}
        rows={24}
      />
      {error && <div className="text-red-600 text-sm mt-2">{error}</div>}
      <Button className="mt-3">Save</Button>
    </form>
  );
}
