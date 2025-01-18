// src/components/AnnotationMenu.tsx
import React from "react";

interface Props {
  x: number;
  y: number;
  onVocab: () => void;
  onGrammar: () => void;
  onFootnote: () => void;
  className?: string;
}

export default function AnnotationMenu({
  x,
  y,
  onVocab,
  onGrammar,
  onFootnote,
  className = "",
}: Props) {
  return (
    <div
      className={`annotation-menu ${className}`.trim()}
      style={{
        position: "fixed",
        left: x,
        top: y,
        zIndex: 1000,
        background: "white",
        boxShadow: "0 2px 10px rgba(0,0,0,0.1)",
        borderRadius: "4px",
        padding: "0.5rem",
      }}
    >
      <button onClick={onVocab}>Add Vocabulary</button>
      <button onClick={onGrammar}>Add Grammar Note</button>
      <button onClick={onFootnote}>Add Footnote</button>
    </div>
  );
}
