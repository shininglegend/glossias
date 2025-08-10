// [moved from annotator/src/components/AnnotationMenu.tsx]
import React from "react";
import Button from "~/components/ui/Button";

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
  const menuRef = React.useRef<HTMLDivElement>(null);
  const [adjustedPosition, setAdjustedPosition] = React.useState({ x, y });

  React.useEffect(() => {
    if (menuRef.current) {
      const rect = menuRef.current.getBoundingClientRect();
      const viewportHeight = window.innerHeight;
      const viewportWidth = window.innerWidth;

      let newY = y;
      let newX = x;

      if (rect.bottom > viewportHeight) newY = y - rect.height - 10;
      if (rect.right > viewportWidth) newX = viewportWidth - rect.width - 10;

      setAdjustedPosition({ x: newX, y: newY });
    }
  }, [x, y]);

  return (
    <div
      ref={menuRef}
      className={`annotation-menu ${className}`.trim()}
      style={{
        position: "fixed",
        left: adjustedPosition.x,
        top: adjustedPosition.y,
        zIndex: 1000,
      }}
    >
      <div className="rounded-md border border-slate-200 bg-white p-2 shadow-lg">
        <div className="grid gap-2">
          <Button size="sm" variant="ghost" onClick={onVocab}>
            Add Vocabulary
          </Button>
          <Button size="sm" variant="ghost" onClick={onGrammar}>
            Add Grammar Note
          </Button>
          <Button size="sm" variant="ghost" onClick={onFootnote}>
            Add Footnote
          </Button>
        </div>
      </div>
    </div>
  );
}
