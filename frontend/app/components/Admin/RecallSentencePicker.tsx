import React from "react";
import Button from "~/components/ui/Button";
import Modal from "~/components/ui/Modal";
import { cn } from "~/lib/cn";
import type { StoryContent } from "../../types/admin";

export interface StorySentence {
  /** Stable key: `${lineNumber}-${indexWithinLine}`. */
  key: string;
  lineNumber: number;
  text: string;
}

/**
 * Splits every story line into sentences on Hebrew/Latin terminal punctuation
 * (. ! ? and sof pasuq), keeping the punctuation on the sentence it closes.
 * Most story lines carry no punctuation at all, so in practice a line is
 * usually one sentence; leading tabs used for indentation are stripped.
 */
export function splitStoryIntoSentences(
  content: StoryContent | null,
): StorySentence[] {
  if (!content) return [];
  const sentences: StorySentence[] = [];
  for (const line of content.lines) {
    const parts = line.text
      .split(/(?<=[.!?׃])\s+/u)
      .map((part) => part.trim())
      .filter(Boolean);
    parts.forEach((text, index) => {
      sentences.push({
        key: `${line.lineNumber}-${index}`,
        lineNumber: line.lineNumber,
        text,
      });
    });
  }
  return sentences;
}

interface RecallSentencePickerProps {
  isOpen: boolean;
  onClose: () => void;
  sentences: StorySentence[];
  /** Sentence text -> position that already uses it (so admins avoid duplicates). */
  usedByPosition: Map<string, number>;
  currentOrder: number;
  onPick: (text: string) => void;
}

/**
 * Lets an admin choose one or more story sentences for a recall position
 * instead of retyping them. Selections are joined in story order.
 */
export default function RecallSentencePicker({
  isOpen,
  onClose,
  sentences,
  usedByPosition,
  currentOrder,
  onPick,
}: RecallSentencePickerProps) {
  const [selected, setSelected] = React.useState<Set<string>>(new Set());

  // Start fresh each time the dialog opens.
  React.useEffect(() => {
    if (isOpen) setSelected(new Set());
  }, [isOpen]);

  const toggle = (key: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  };

  const pick = () => {
    const text = sentences
      .filter((s) => selected.has(s.key))
      .map((s) => s.text)
      .join(" ");
    onPick(text);
    onClose();
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={`Pick sentence(s) for position ${currentOrder}`}
      description="Select one or more sentences from the story. They are joined in story order and you can still edit the result."
      className="max-w-2xl"
    >
      {sentences.length === 0 ? (
        <p className="text-sm text-slate-600">
          This story has no text yet. Add lines in the story editor first.
        </p>
      ) : (
        <ul className="max-h-[60vh] overflow-y-auto divide-y divide-slate-100 border border-slate-200 rounded-md">
          {sentences.map((sentence) => {
            const usedAt = usedByPosition.get(sentence.text);
            const isChecked = selected.has(sentence.key);
            return (
              <li key={sentence.key}>
                <label
                  className={cn(
                    "flex items-center gap-3 px-3 py-2 cursor-pointer hover:bg-slate-50",
                    isChecked && "bg-primary-50",
                  )}
                >
                  <input
                    type="checkbox"
                    checked={isChecked}
                    onChange={() => toggle(sentence.key)}
                  />
                  <span className="shrink-0 w-24 text-xs text-slate-400 leading-6">
                    Line {sentence.lineNumber}
                    {usedAt !== undefined && usedAt !== currentOrder && (
                      <span className="block">used at pos. {usedAt}</span>
                    )}
                  </span>
                  <span dir="rtl" className="flex-1 min-w-0 text-right">
                    {sentence.text}
                  </span>
                </label>
              </li>
            );
          })}
        </ul>
      )}

      <div className="flex justify-end gap-2 mt-4">
        <Button variant="secondary" onClick={onClose}>
          Cancel
        </Button>
        <Button onClick={pick} disabled={selected.size === 0}>
          Use {selected.size || ""}{" "}
          {selected.size === 1 ? "sentence" : "sentences"}
        </Button>
      </div>
    </Modal>
  );
}
