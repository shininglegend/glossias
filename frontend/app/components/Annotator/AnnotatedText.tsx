// [moved from annotator/src/components/AnnotatedText.tsx]
import React, { useMemo, useState } from "react";
import { createPortal } from "react-dom";
import type { VocabularyItem as VocabItem, GrammarItem } from "../../types/api";

export interface Props {
  text: string;
  vocabulary: VocabItem[];
  grammar: GrammarItem[];
  onSelect?: (start: number, end: number, text: string) => void;
}

interface Annotation {
  start: number;
  end: number;
  type: "vocab" | "grammar";
  data: VocabItem | GrammarItem;
}

export default function AnnotatedText({
  text,
  vocabulary,
  grammar,
  onSelect,
}: Props) {
  const segments = useMemo(() => {
    const annotations: Annotation[] = [
      ...vocabulary.map((v) => ({
        start: v.position[0],
        end: v.position[1],
        type: "vocab" as const,
        data: v,
      })),
      ...grammar.map((g) => ({
        start: g.position[0],
        end: g.position[1],
        type: "grammar" as const,
        data: g,
      })),
    ].sort((a, b) => a.start - b.start);

    return createTextSegments(text, annotations);
  }, [text, vocabulary, grammar]);

  const handleMouseUp = () => {
    const selection = window.getSelection();
    if (!selection || !onSelect) return;

    const range = selection.getRangeAt(0);
    const container = (range.commonAncestorContainer as HTMLElement)
      .parentElement;
    if (!container?.closest(".annotated-text")) return;

    const textNodes: Node[] = [];
    const walker = document.createTreeWalker(
      container.closest(".annotated-text")!,
      NodeFilter.SHOW_TEXT,
      null
    );
    let node: Node | null;
    while ((node = walker.nextNode())) textNodes.push(node);

    let absoluteStart = 0;
    let absoluteEnd = 0;
    let foundStart = false;
    let currentPosition = 0;

    for (const n of textNodes) {
      const nodeLength = n.textContent?.length || 0;
      if (n === range.startContainer) {
        absoluteStart = currentPosition + range.startOffset;
        foundStart = true;
      }
      if (n === range.endContainer) {
        absoluteEnd = currentPosition + range.endOffset;
        break;
      }
      if (!foundStart) currentPosition += nodeLength;
    }

    if (absoluteStart !== absoluteEnd)
      onSelect(absoluteStart, absoluteEnd, selection.toString());
  };

  return (
    <span className="annotated-text leading-7" onMouseUp={handleMouseUp}>
      {segments.map((segment, i) => (
        <TextSegment key={i} segment={segment} />
      ))}
    </span>
  );
}

interface TextSegments {
  text: string;
  annotations: Annotation[];
}

function TextSegment({
  segment: { text, annotations },
}: {
  segment: TextSegments;
}) {
  const [showTooltip, setShowTooltip] = useState(false);
  const [tooltipPosition, setTooltipPosition] = useState({ x: 0, y: 0 });
  const [hideTimeout, setHideTimeout] = useState<NodeJS.Timeout | null>(null);

  if (!annotations.length) return <>{text}</>;

  const classes = annotations.map((a) =>
    a.type === "vocab" ? "vocab-highlight" : "grammar-highlight"
  );

  const tooltipParts: string[] = [];
  annotations.forEach((a) => {
    if (a.type === "vocab") {
      const vocab = a.data as VocabItem;
      tooltipParts.push(`${vocab.word} → ${vocab.lexicalForm}`);
    } else if (a.type === "grammar") {
      const grammar = a.data as GrammarItem;
      tooltipParts.push(`Grammar: ${grammar.text}`);
    }
  });

  const handleMouseEnter = (e: React.MouseEvent) => {
    if (hideTimeout) {
      clearTimeout(hideTimeout);
      setHideTimeout(null);
    }
    const rect = e.currentTarget.getBoundingClientRect();
    setTooltipPosition({
      x: rect.left + rect.width / 2,
      y: rect.top - 16,
    });
    setShowTooltip(true);
  };

  const handleMouseLeave = () => {
    const timeout = setTimeout(() => setShowTooltip(false), 100);
    setHideTimeout(timeout);
  };

  return (
    <span
      className={`${classes.join(" ")} relative`}
      onMouseEnter={handleMouseEnter}
      onMouseLeave={handleMouseLeave}
      data-testid="annotated-segment"
    >
      {text}
      {showTooltip &&
        tooltipParts.length > 0 &&
        typeof document !== "undefined" &&
        createPortal(
          <div
            className="fixed z-50 bg-gray-900 text-white text-xs px-2 py-1 rounded shadow-lg pointer-events-none transform -translate-x-1/2 -translate-y-full"
            style={{
              left: tooltipPosition.x,
              top: tooltipPosition.y,
            }}
          >
            {tooltipParts.map((part, i) => (
              <div key={i}>{part}</div>
            ))}
          </div>,
          document.body
        )}
    </span>
  );
}

function createTextSegments(
  text: string,
  annotations: Annotation[]
): TextSegments[] {
  const segments: TextSegments[] = [];
  let lastIndex = 0;
  let activeAnnotations: Annotation[] = [];

  const positions = getUniquePositions(annotations);
  positions.forEach((pos) => {
    if (pos > lastIndex) {
      segments.push({
        text: text.slice(lastIndex, pos),
        annotations: [...activeAnnotations],
      });
    }
    activeAnnotations = activeAnnotations.filter((a) => a.end > pos);
    const newAnnotations = annotations.filter((a) => a.start === pos);
    activeAnnotations.push(...newAnnotations);
    lastIndex = pos;
  });

  if (lastIndex < text.length)
    segments.push({ text: text.slice(lastIndex), annotations: [] });
  return segments;
}

function getUniquePositions(annotations: Annotation[]): number[] {
  const positions = new Set<number>();
  annotations.forEach((a) => {
    positions.add(a.start);
    positions.add(a.end);
  });
  return Array.from(positions).sort((a, b) => a - b);
}
