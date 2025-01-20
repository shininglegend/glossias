// src/components/Line.tsx
import React, { useState, useEffect, useCallback } from "react";
import AnnotatedText, { Props } from "./AnnotatedText.tsx";
import AnnotationMenu from "./AnnotationMenu.tsx";
import AnnotationModal from "./AnnotationModal.tsx";

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

  // [+] Add click-away handler
  const handleClickAway = useCallback((event: MouseEvent) => {
    const target = event.target as HTMLElement;
    if (
      !target.closest(".annotation-menu") &&
      !target.closest(".annotated-text")
    ) {
      setMenu(null);
    }
  }, []);

  // [+] Add and remove click listener
  useEffect(() => {
    document.addEventListener("mousedown", handleClickAway);
    return () => {
      document.removeEventListener("mousedown", handleClickAway);
    };
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

    setMenu({
      x: rect.left,
      y: rect.bottom + 5,
    });
    setSelection({ start, end, text }); // Save selection info
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
      data,
    );
    setModal(null);
    setSelection(null);

    // Clear selection
    window.getSelection()?.removeAllRanges();
  };

  return (
    <div className="story-line">
      <span className="line-number">{line.lineNumber}</span>
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
