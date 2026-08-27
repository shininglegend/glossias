import React, { useRef } from "react";
import Modal from "../ui/Modal";
import Button from "../ui/Button";

interface ProduceExplanationModalProps {
  isOpen: boolean;
  /** Authored contrastive grammar explanation; empty when none exists. */
  explanation: string;
  grammarPointNames: string[];
  onClose: () => void;
}

/**
 * The popup shown after both Produce segments: how the grammar point works in
 * each segment, what it contributes, and how the two compare. The text is
 * authored per story (admin Produce editor); paragraphs are split on blank
 * lines.
 */
export const ProduceExplanationModal: React.FC<
  ProduceExplanationModalProps
> = ({ isOpen, explanation, grammarPointNames, onClose }) => {
  const closeRef = useRef<HTMLButtonElement>(null);
  const paragraphs = explanation
    .split(/\n\s*\n/)
    .map((p) => p.trim())
    .filter(Boolean);
  const names = Array.from(new Set(grammarPointNames.filter(Boolean)));

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title="How the grammar works"
      description={
        names.length > 0 ? `Grammar point: ${names.join(", ")}` : undefined
      }
      initialFocusRef={closeRef}
      className="sm:max-w-2xl"
    >
      <div
        className="mt-4 max-h-[60vh] overflow-y-auto space-y-3 text-gray-800 leading-relaxed"
        data-testid="produce-explanation"
      >
        {paragraphs.length > 0 ? (
          paragraphs.map((p, i) => (
            <p key={i} className="whitespace-pre-line">
              {p}
            </p>
          ))
        ) : (
          <p className="text-gray-500 italic">
            No explanation has been written for this story yet. Compare your
            attempts with the reference sentences above.
          </p>
        )}
      </div>
      <div className="mt-6 flex justify-end">
        <Button ref={closeRef} variant="primary" onClick={onClose}>
          Got it
        </Button>
      </div>
    </Modal>
  );
};
