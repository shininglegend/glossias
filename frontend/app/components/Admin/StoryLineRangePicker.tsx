import React from "react";
import Button from "~/components/ui/Button";
import Modal from "~/components/ui/Modal";
import { cn } from "~/lib/cn";
import { clickLineRange } from "../../lib/produceRange";
import type { StoryLine } from "../../types/admin";

interface StoryLineRangePickerProps {
  isOpen: boolean;
  onClose: () => void;
  lines: StoryLine[];
  lineStart: number | "";
  lineEnd: number | "";
  onPick: (start: number, end: number) => void;
}

/**
 * Click story lines to choose a Produce segment's range. First click sets a
 * single line; the next click expands to that span.
 */
export default function StoryLineRangePicker({
  isOpen,
  onClose,
  lines,
  lineStart,
  lineEnd,
  onPick,
}: StoryLineRangePickerProps) {
  const [start, setStart] = React.useState<number | "">(lineStart);
  const [end, setEnd] = React.useState<number | "">(lineEnd);

  React.useEffect(() => {
    if (isOpen) {
      setStart(lineStart);
      setEnd(lineEnd);
    }
  }, [isOpen, lineStart, lineEnd]);

  const inRange = (n: number) =>
    start !== "" && end !== "" && n >= start && n <= end;

  const pick = () => {
    if (start === "" || end === "") return;
    onPick(start, end);
    onClose();
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title="Pick story lines"
      className="max-w-2xl"
    >
      {lines.length === 0 ? (
        <p className="text-sm text-slate-600">
          This story has no text yet. Add lines in the story editor first.
        </p>
      ) : (
        <ul className="max-h-[60vh] overflow-y-auto divide-y divide-slate-100 border border-slate-200 rounded-md">
          {lines.map((line) => {
            const selected = inRange(line.lineNumber);
            return (
              <li key={line.lineNumber}>
                <button
                  type="button"
                  onClick={() => {
                    const next = clickLineRange(start, end, line.lineNumber);
                    setStart(next.start);
                    setEnd(next.end);
                  }}
                  className={cn(
                    "flex w-full items-center gap-3 px-3 py-2 text-left hover:bg-slate-50",
                    selected && "bg-primary-50",
                  )}
                >
                  <span className="shrink-0 w-16 text-xs text-slate-400">
                    Line {line.lineNumber}
                  </span>
                  <span dir="rtl" className="flex-1 min-w-0 text-right">
                    {line.text}
                  </span>
                </button>
              </li>
            );
          })}
        </ul>
      )}

      <div className="flex justify-end gap-2 mt-4">
        <Button variant="secondary" onClick={onClose}>
          Cancel
        </Button>
        <Button onClick={pick} disabled={start === "" || end === ""}>
          Use lines
        </Button>
      </div>
    </Modal>
  );
}
