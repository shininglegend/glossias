// [moved from annotator/src/components/Line.tsx]
import React, { useState, useEffect, useCallback } from "react";
import AnnotatedText from "./AnnotatedText";

interface VocabularyItem {
  lexicalForm: string;
}

interface GrammarItem {
  text: string;
}

interface StoryLine {
  lineNumber: number;
  text: string;
  vocabulary: VocabularyItem[];
  grammar: GrammarItem[];
}

interface Props {
  line: StoryLine;
  onSelect: (
    lineNumber: number,
    text: string,
    type: "vocab" | "grammar" | "footnote",
    start: number,
    end: number,
    data?: { text?: string; lexicalForm?: string },
  ) => void;
}

export default function Line({ line, onSelect }: Props) {
  const [menu, setMenu] = useState<{ x: number; y: number } | null>(null);
  const [modal, setModal] = useState<{ type: "vocab" | "grammar" | "footnote"; text: string } | null>(null);
  const [selection, setSelection] = useState<{ start: number; end: number; text: string } | null>(null);

  const handleClickAway = useCallback((event: MouseEvent) => {
    const target = event.target as HTMLElement;
    if (!target.closest(".annotation-menu") && !target.closest(".annotated-text")) {
      setMenu(null);
    }
  }, []);

  useEffect(() => {
    document.addEventListener("mousedown", handleClickAway);
    return () => document.removeEventListener("mousedown", handleClickAway);
  }, [handleClickAway]);

  const handleSelect = (start: number, end: number, text: string) => {
    const sel = window.getSelection();
    if (!sel?.toString()) {
      setMenu(null);
      setSelection(null);
      return;
    }
    const range = sel.getRangeAt(0);
    const rect = range.getBoundingClientRect();
    setMenu({ x: rect.left, y: rect.bottom + 5 });
    setSelection({ start, end, text });
  };

  const handleAnnotate = (type: "vocab" | "grammar" | "footnote") => {
    if (!selection) return;
    setModal({ type, text: selection.text });
    setMenu(null);
  };

  const handleSave = (data: { text?: string; lexicalForm?: string }) => {
    if (!modal || !selection) return;
    onSelect(line.lineNumber, selection.text, modal.type, selection.start, selection.end, data);
    setModal(null);
    setSelection(null);
    window.getSelection()?.removeAllRanges();
  };

  return (
    <div className="story-line">
      <span className="line-number">{line.lineNumber}</span>
      <AnnotatedText text={line.text} vocabulary={line.vocabulary} grammar={line.grammar} onSelect={handleSelect} />
      {menu && (
        <div className="annotation-menu" style={{ position: "fixed", left: menu.x, top: menu.y }}>
          <button onClick={() => handleAnnotate("vocab")}>Add Vocabulary</button>
          <button onClick={() => handleAnnotate("grammar")}>Add Grammar Note</button>
          <button onClick={() => handleAnnotate("footnote")}>Add Footnote</button>
        </div>
      )}
      {modal && (
        <div className="modal">
          <h3>Add {modal.type}</h3>
          <p>Selected text: {modal.text}</p>
          {/* Use the shared modal in real UI; simplified here to minimize dependencies */}
          <button onClick={() => handleSave({})}>Save</button>
          <button onClick={() => setModal(null)}>Cancel</button>
        </div>
      )}
    </div>
  );
}


