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
  // [+] Adjust menu position if it would appear off-screen
  const menuRef = React.useRef<HTMLDivElement>(null);
  const [adjustedPosition, setAdjustedPosition] = React.useState({ x, y });

  React.useEffect(() => {
    if (menuRef.current) {
      const rect = menuRef.current.getBoundingClientRect();
      const viewportHeight = window.innerHeight;
      const viewportWidth = window.innerWidth;

      let newY = y;
      let newX = x;

      // Adjust vertical position if menu would appear off-screen
      if (rect.bottom > viewportHeight) {
        newY = y - rect.height - 10; // Position above selection
      }

      // Adjust horizontal position if menu would appear off-screen
      if (rect.right > viewportWidth) {
        newX = viewportWidth - rect.width - 10;
      }

      setAdjustedPosition({ x: newX, y: newY });
    }
  }, [x, y]);

  return (
    <div
      ref={menuRef}
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
