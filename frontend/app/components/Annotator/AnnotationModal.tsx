// [moved from annotator/src/components/AnnotationModal.tsx]
import { useState } from "react";
import Button from "~/components/ui/Button";
import Input from "~/components/ui/Input";
import Label from "~/components/ui/Label";
import type { GrammarPoint } from "../../types/admin";

interface Props {
  type: "vocab" | "grammar" | "footnote";
  selectedText: string;
  onSave: (data: {
    text?: string;
    lexicalForm?: string;
    grammarPointId?: number;
  }) => void;
  onClose: () => void;
  storyGrammarPoints?: GrammarPoint[];
}

export default function AnnotationModal({
  type,
  selectedText,
  onSave,
  onClose,
  storyGrammarPoints = [],
}: Props) {
  const [input, setInput] = useState("");
  const [selectedGrammarPointId, setSelectedGrammarPointId] = useState<
    number | undefined
  >();

  const handleSave = () => {
    if (type === "grammar") {
      onSave({ text: selectedText, grammarPointId: selectedGrammarPointId });
    } else if (type === "vocab") {
      onSave({ lexicalForm: input });
    } else {
      onSave({ text: input });
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/30 p-4">
      <div className="w-full max-w-md rounded-lg border border-slate-200 bg-white p-4 shadow-xl">
        <h3 className="text-lg font-semibold">Add {type}</h3>
        <p className="mt-1 text-sm text-slate-600">
          Selected: <span className="font-medium">{selectedText}</span>
        </p>
        {type === "grammar" && (
          <div className="mt-3">
            <Label>Grammar Point</Label>
            {storyGrammarPoints.length === 0 ? (
              <div className="w-full px-3 py-2 border border-red-300 rounded-md bg-red-50 text-red-700 text-sm">
                No grammar points available for this story. Please add grammar
                points in the metadata first.
              </div>
            ) : (
              <select
                value={selectedGrammarPointId || ""}
                onChange={(e) =>
                  setSelectedGrammarPointId(
                    e.target.value ? Number(e.target.value) : undefined,
                  )
                }
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
              >
                <option value="">Select a grammar point...</option>
                {storyGrammarPoints.map((gp) => (
                  <option key={gp.id} value={gp.id}>
                    {gp.name} {gp.description && `- ${gp.description}`}
                  </option>
                ))}
              </select>
            )}
          </div>
        )}
        {type !== "grammar" && (
          <div className="mt-3">
            <Label>{type === "vocab" ? "Lexical form" : "Note"}</Label>
            <Input
              value={input}
              onChange={(e) => setInput(e.target.value)}
              placeholder={type === "vocab" ? "e.g. lemma" : "Enter note"}
              autoFocus={type === "vocab"}
            />
          </div>
        )}
        <div className="mt-4 flex justify-end gap-2">
          <Button variant="ghost" onClick={onClose}>
            Cancel
          </Button>
          <Button
            onClick={handleSave}
            disabled={
              type === "grammar" &&
              (!selectedGrammarPointId || storyGrammarPoints.length === 0)
            }
          >
            Save
          </Button>
        </div>
      </div>
    </div>
  );
}
