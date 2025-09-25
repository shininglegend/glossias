// [moved from annotator/src/components/AnnotatedText.tsx]
import React, { useMemo, useState } from "react";
import { createPortal } from "react-dom";
import type {
  VocabularyItem as VocabItem,
  GrammarItem,
  GrammarPoint,
} from "../../types/api";

const RTL_LANGUAGES = ["he", "ar", "fa", "ur"];

export interface Props {
  text: string;
  vocabulary: VocabItem[];
  grammar: GrammarItem[];
  grammarPoints?: GrammarPoint[];
  languageCode?: string;
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
  grammarPoints = [],
  languageCode,
  onSelect,
}: Props) {
  const isRTL = languageCode && RTL_LANGUAGES.includes(languageCode);

  // Store original text for position calculations
  const originalText = useMemo(() => text, [text]);

  // Handle RTL indentation by converting leading tabs to padding
  const { displayText, indentLevel } = useMemo(() => {
    if (!isRTL) return { displayText: text, indentLevel: 0 };

    const leadingTabs = text.match(/^\t*/)?.[0] || "";
    const tabCount = leadingTabs.length;
    const textWithoutTabs = text.slice(tabCount);

    return {
      displayText: textWithoutTabs,
      indentLevel: tabCount,
    };
  }, [text, isRTL]);

  const segments = useMemo(() => {
    const annotations: Annotation[] = [
      ...vocabulary.map((v) => ({
        start: v.position[0] - (isRTL ? indentLevel : 0), // Adjust positions for removed tabs
        end: v.position[1] - (isRTL ? indentLevel : 0),
        type: "vocab" as const,
        data: v,
      })),
      ...grammar.map((g) => ({
        start: g.position[0] - (isRTL ? indentLevel : 0),
        end: g.position[1] - (isRTL ? indentLevel : 0),
        type: "grammar" as const,
        data: g,
      })),
    ].sort((a, b) => a.start - b.start);

    return createTextSegments(displayText, annotations);
  }, [displayText, vocabulary, grammar, isRTL, indentLevel]);

  const handleMouseUp = () => {
    const selection = window.getSelection();
    if (!selection || !onSelect) return;

    const range = selection.getRangeAt(0);
    const annotatedTextElement = (
      range.commonAncestorContainer.nodeType === Node.TEXT_NODE
        ? range.commonAncestorContainer.parentElement
        : (range.commonAncestorContainer as HTMLElement)
    )?.closest(".annotated-text");

    if (!annotatedTextElement) return;

    // Get the selected text directly from the original text
    const selectedText = selection.toString();
    if (!selectedText.trim()) return;

    // Find the selection in the original text
    // We need to account for RTL indentation adjustments
    const searchText = isRTL ? displayText : originalText;
    const startIndex = searchText.indexOf(selectedText);

    if (startIndex === -1) return;

    // Adjust positions back to original text coordinates
    const originalStart = isRTL ? startIndex + indentLevel : startIndex;
    const originalEnd = originalStart + selectedText.length;

    onSelect(originalStart, originalEnd, selectedText);
  };

  return (
    <span
      className={`annotated-text leading-7 whitespace-pre ${isRTL ? "text-right" : "text-left"}`}
      dir={isRTL ? "rtl" : "ltr"}
      style={isRTL ? { paddingRight: `${indentLevel * 2}em` } : undefined}
      onMouseUp={handleMouseUp}
    >
      {segments.map((segment, i) => (
        <TextSegment key={i} segment={segment} grammarPoints={grammarPoints} />
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
  grammarPoints,
}: {
  segment: TextSegments;
  grammarPoints: GrammarPoint[];
}) {
  const [showTooltip, setShowTooltip] = useState(false);
  const [tooltipPosition, setTooltipPosition] = useState({ x: 0, y: 0 });
  const [hideTimeout, setHideTimeout] = useState<NodeJS.Timeout | null>(null);

  if (!annotations.length) return <>{text}</>;

  const grammarColorClasses = [
    "text-decoration: underline wavy #ef4444;", // red
    "text-decoration: underline wavy #3b82f6;", // blue
    "text-decoration: underline wavy #10b981;", // green
    "text-decoration: underline wavy #f59e0b;", // amber
    "text-decoration: underline wavy #8b5cf6;", // violet
    "text-decoration: underline wavy #ec4899;", // pink
    "text-decoration: underline wavy #06b6d4;", // cyan
    "text-decoration: underline wavy #84cc16;", // lime
  ];

  const classes: string[] = [];
  const styles: React.CSSProperties = {};

  annotations.forEach((a) => {
    if (a.type === "vocab") {
      classes.push("vocab-highlight");
    } else {
      const grammar = a.data as GrammarItem;
      const grammarPoint = grammarPoints.find(
        (gp: GrammarPoint) => gp.id === grammar.grammarPointId,
      );
      if (grammarPoint) {
        const colorIndex = grammarPoint.id % grammarColorClasses.length;
        styles.textDecoration = grammarColorClasses[colorIndex]
          .split(": ")[1]
          .slice(0, -1);
        styles.textUnderlineOffset = "3px";
      }
      classes.push("grammar-highlight");
    }
  });

  const tooltipParts: string[] = [];
  annotations.forEach((a) => {
    if (a.type === "vocab") {
      const vocab = a.data as VocabItem;
      tooltipParts.push(`${vocab.word} → ${vocab.lexicalForm}`);
    } else if (a.type === "grammar") {
      const grammar = a.data as GrammarItem;
      const grammarPoint = grammarPoints.find(
        (gp: GrammarPoint) => gp.id === grammar.grammarPointId,
      );
      const grammarPointName = grammarPoint ? grammarPoint.name : "Unknown";
      tooltipParts.push(`${grammarPointName}: ${grammar.text}`);
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
      style={styles}
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
          document.body,
        )}
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
