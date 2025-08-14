// [moved from annotator/src/components/AnnotationModal.tsx]
import React, { useState } from "react";
import Button from "~/components/ui/Button";
import Input from "~/components/ui/Input";
import Label from "~/components/ui/Label";

interface Props {
  type: "vocab" | "grammar" | "footnote";
  selectedText: string;
  onSave: (data: { text?: string; lexicalForm?: string }) => void;
  onClose: () => void;
}

export default function AnnotationModal({
  type,
  selectedText,
  onSave,
  onClose,
}: Props) {
  const [input, setInput] = useState("");

  const handleSave = () => {
    if (type === "grammar") {
      onSave({ text: selectedText });
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
          <Button variant="ghost" onClick={onClose}>
            Cancel
          </Button>
          <Button onClick={handleSave}>Save</Button>
        </div>
      </div>
    </div>
  );
}
