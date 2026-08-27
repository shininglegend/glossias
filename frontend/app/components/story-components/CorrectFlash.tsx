import React from "react";
import "./CorrectFlash.css";

/** How long the checkmark stays on screen before it is removed. */
export const CORRECT_FLASH_MS = 1200;

interface CorrectFlashProps {
  /** Rendered while true; the parent clears it after CORRECT_FLASH_MS. */
  visible: boolean;
}

/**
 * A big green checkmark overlaid on the whole screen after a correct answer.
 * Purely decorative and non-blocking: it ignores pointer events so the story
 * keeps playing and the student can keep interacting underneath it. The
 * pop-and-fade animation collapses to a plain fade for reduced-motion users.
 */
export const CorrectFlash: React.FC<CorrectFlashProps> = ({ visible }) => {
  if (!visible) return null;
  return (
    <div
      className="correct-flash fixed inset-0 z-50 flex items-center justify-center pointer-events-none"
      data-testid="identify-correct-flash"
      role="status"
      aria-live="polite"
    >
      <span className="sr-only">Correct!</span>
      <span
        className="correct-flash__mark flex items-center justify-center w-48 h-48 sm:w-64 sm:h-64 rounded-full bg-green-500 text-white shadow-2xl"
        aria-hidden="true"
      >
        <span className="material-icons" style={{ fontSize: "8rem" }}>
          check
        </span>
      </span>
    </div>
  );
};
