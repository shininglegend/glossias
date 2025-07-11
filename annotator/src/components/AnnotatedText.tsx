// src/components/AnnotatedText.tsx
import React, { useMemo } from "react";
import { VocabularyItem as VocabItem, GrammarItem } from "../types";

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
    const container = range.commonAncestorContainer.parentElement;
    if (!container?.closest(".annotated-text")) return;

    // [+] Get all text nodes in order
    const textNodes: Node[] = [];
    const walker = document.createTreeWalker(
      container.closest(".annotated-text")!,
      NodeFilter.SHOW_TEXT,
      null,
    );

    let node;
    while ((node = walker.nextNode())) {
      textNodes.push(node);
    }

    // [+] Calculate absolute positions
    let absoluteStart = 0;
    let absoluteEnd = 0;
    let foundStart = false;
    let currentPosition = 0;

    for (const node of textNodes) {
      const nodeLength = node.textContent?.length || 0;

      if (node === range.startContainer) {
        absoluteStart = currentPosition + range.startOffset;
        foundStart = true;
      }

      if (node === range.endContainer) {
        absoluteEnd = currentPosition + range.endOffset;
        break;
      }

      if (!foundStart) {
        currentPosition += nodeLength;
      }
    }

    if (absoluteStart !== absoluteEnd) {
      onSelect(absoluteStart, absoluteEnd, selection.toString());
    }
  };

  return (
    <span className="annotated-text" onMouseUp={handleMouseUp}>
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
  if (!annotations.length) return <>{text}</>;

  const classes = annotations.map((a) =>
    a.type === "vocab" ? "vocab-highlight" : "grammar-highlight",
  );
  const vocabAnnotations = annotations.filter((a) => a.type === "vocab");
  const tooltip =
    vocabAnnotations.length > 0
      ? vocabAnnotations
          .map((a) => (a.data as VocabItem).lexicalForm)
          .join("\n")
      : undefined; // No tooltip if there are no vocab annotations

  return (
    <span
      className={classes.join(" ")}
      title={tooltip}
      data-testid="annotated-segment"
    >
      {text}
    </span>
  );
}

function createTextSegments(
  text: string,
  annotations: Annotation[],
): TextSegments[] {
  const segments: TextSegments[] = [];
  let lastIndex = 0;
  let activeAnnotations: Annotation[] = [];

  const positions = getUniquePositions(annotations);

  positions.forEach((pos) => {
    // Add text segment before this position
    if (pos > lastIndex) {
      segments.push({
        text: text.slice(lastIndex, pos),
        annotations: [...activeAnnotations],
      });
    }

    // Update active annotations
    activeAnnotations = activeAnnotations.filter((a) => a.end > pos);
    const newAnnotations = annotations.filter((a) => a.start === pos);
    activeAnnotations.push(...newAnnotations);

    lastIndex = pos;
  });

  // Add remaining text
  if (lastIndex < text.length) {
    segments.push({
      text: text.slice(lastIndex),
      annotations: [],
    });
  }

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
