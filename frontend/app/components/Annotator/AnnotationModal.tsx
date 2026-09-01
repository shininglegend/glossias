// [moved from annotator/src/components/AnnotationModal.tsx]
import { useState } from "react";
import Button from "~/components/ui/Button";
import Input from "~/components/ui/Input";
import Label from "~/components/ui/Label";
import Modal from "~/components/ui/Modal";
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

  const canSave =
    type !== "grammar" ||
    (!!selectedGrammarPointId && storyGrammarPoints.length > 0);

  const handleSave = () => {
    if (!canSave) return;
    if (type === "grammar") {
      onSave({ text: selectedText, grammarPointId: selectedGrammarPointId });
    } else if (type === "vocab") {
      onSave({ lexicalForm: input });
    } else {
      onSave({ text: input });
    }
  };

  // Escape, focus trapping and initial focus (first control) are handled by
  // Modal's native <dialog>.
  return (
    <Modal
      isOpen
      onClose={onClose}
      title={`Add ${type}`}
      description={
        <>
          Selected: <span className="font-medium">{selectedText}</span>
        </>
      }
    >
      <form
        onSubmit={(e) => {
          e.preventDefault();
          handleSave();
        }}
      >
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
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-primary-500 focus:border-primary-500"
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
            />
          </div>
        )}
        <div className="mt-4 flex justify-end gap-2">
          <Button type="button" variant="ghost" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" disabled={!canSave}>
            Save
          </Button>
        </div>
      </form>
    </Modal>
  );
}
