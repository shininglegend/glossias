// [moved from annotator/src/components/Line.tsx]
import React, { useState, useEffect, useCallback } from "react";
import AnnotatedText from "./AnnotatedText";
import AnnotationMenu from "./AnnotationMenu";
import AnnotationModal from "./AnnotationModal";
import type { StoryLine } from "../../types/api";

interface Props {
  line: StoryLine;
  onSelect: (
    lineNumber: number,
    text: string,
    type: "vocab" | "grammar" | "footnote",
    start: number,
    end: number,
    data?: { text?: string; lexicalForm?: string }
  ) => void;
}

export default function Line({ line, onSelect }: Props) {
  const [menu, setMenu] = useState<{ x: number; y: number } | null>(null);
  const [modal, setModal] = useState<{
    type: "vocab" | "grammar" | "footnote";
    text: string;
  } | null>(null);
  const [selection, setSelection] = useState<{
    start: number;
    end: number;
    text: string;
  } | null>(null);

  const handleClickAway = useCallback((event: MouseEvent) => {
    const target = event.target as HTMLElement;
    if (
      !target.closest(".annotation-menu") &&
      !target.closest(".annotated-text")
    ) {
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
    onSelect(
      line.lineNumber,
      selection.text,
      modal.type,
      selection.start,
      selection.end,
      data
    );
    setModal(null);
    setSelection(null);
    window.getSelection()?.removeAllRanges();
  };

  return (
    <div className="story-line">
      <span className="line-number text-slate-500 mr-1">{line.lineNumber}</span>
      <AnnotatedText
        text={line.text}
        vocabulary={line.vocabulary}
        grammar={line.grammar}
        onSelect={handleSelect}
      />
      {menu && (
        <AnnotationMenu
          x={menu.x}
          y={menu.y}
          onVocab={() => handleAnnotate("vocab")}
          onGrammar={() => handleAnnotate("grammar")}
          onFootnote={() => handleAnnotate("footnote")}
          className="annotation-menu"
        />
      )}
      {modal && (
        <AnnotationModal
          type={modal.type}
          selectedText={modal.text}
          onSave={handleSave}
          onClose={() => setModal(null)}
        />
      )}
    </div>
  );
}
