// src/components/AnnotationModal.tsx
import React, { useState } from "react";

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
    <div className="modal-overlay">
      <div className="modal">
        <h3>Add {type}</h3>
        <p>Selected text: {selectedText}</p>
        {type !== "grammar" && (
          <input
            type="text"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            placeholder={type === "vocab" ? "Enter lexical form" : "Enter note"}
          />
        )}
        <div>
          <button onClick={handleSave}>Save</button>
          <button onClick={onClose}>Cancel</button>
        </div>
      </div>
    </div>
  );
}
