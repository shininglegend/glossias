import type { ReactNode } from "react";

export interface AttemptOption {
  number: number;
  /** Extra text after the number, e.g. "(current)". */
  suffix?: string;
}

interface AttemptPickerProps {
  attempts: AttemptOption[];
  selected: number;
  onSelect: (attempt: number) => void;
  /** Trailing note, e.g. how many times the story was done. */
  children?: ReactNode;
  className?: string;
}

/**
 * Row of attempt-number toggles shared by the student Score page and the
 * admin drill-down. Renders nothing when there is only one attempt to show.
 */
export function AttemptPicker({
  attempts,
  selected,
  onSelect,
  children,
  className = "",
}: AttemptPickerProps) {
  if (attempts.length <= 1) return null;
  return (
    <div className={`flex flex-wrap items-center gap-2 ${className}`}>
      <span className="text-sm font-semibold text-gray-700">Attempt</span>
      {attempts.map((item) => (
        <button
          key={item.number}
          type="button"
          onClick={() => onSelect(item.number)}
          aria-pressed={item.number === selected}
          className={`rounded border px-3 py-1 text-sm ${
            item.number === selected
              ? "border-primary-500 bg-primary-50 font-semibold text-primary-700"
              : "border-gray-300 bg-white text-gray-700 hover:bg-gray-50"
          }`}
        >
          {item.number}
          {item.suffix ? ` ${item.suffix}` : ""}
        </button>
      ))}
      {children}
    </div>
  );
}
