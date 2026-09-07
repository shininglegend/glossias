import { useState } from "react";
import Modal from "../ui/Modal";
import Button from "../ui/Button";

export interface ProduceFeedbackDetail {
  segmentOrder: number;
  hebrewText: string;
  studentText: string;
  referenceEnglish: string;
  aiScore?: number | null;
  aiFeedback?: string;
}

function DetailBlock({
  label,
  children,
  dir,
  lang,
}: {
  label: string;
  children: React.ReactNode;
  dir?: "rtl" | "ltr";
  lang?: string;
}) {
  return (
    <div className="rounded-lg border border-slate-200 bg-slate-50 p-3">
      <p className="text-xs font-semibold uppercase tracking-wider text-slate-500 mb-1">
        {label}
      </p>
      <div
        className="text-gray-900 whitespace-pre-wrap leading-relaxed"
        dir={dir}
        lang={lang}
      >
        {children}
      </div>
    </div>
  );
}

/** Opens a modal with the Hebrew, the student's English, the story English, and the AI notes. */
export function ProduceFeedbackModal({
  detail,
  onClose,
}: {
  detail: ProduceFeedbackDetail | null;
  onClose: () => void;
}) {
  if (!detail) return null;

  return (
    <Modal
      isOpen
      onClose={onClose}
      title={`Passage ${detail.segmentOrder}`}
      className="sm:max-w-lg"
    >
      <div className="mt-4 space-y-3" data-testid="produce-feedback-modal">
        <DetailBlock label="Original" dir="rtl" lang="he">
          {detail.hebrewText || (
            <span className="italic text-gray-400">(none)</span>
          )}
        </DetailBlock>
        <DetailBlock label="Your English" dir="ltr" lang="en">
          {detail.studentText || (
            <span className="italic text-gray-400">(nothing written)</span>
          )}
        </DetailBlock>
        <DetailBlock label="The story's English" dir="ltr" lang="en">
          {detail.referenceEnglish || (
            <span className="italic text-gray-400">(none)</span>
          )}
        </DetailBlock>
        <div className="rounded-lg border border-teal-200 bg-teal-50 p-3">
          <p className="text-xs font-semibold uppercase tracking-wider text-teal-700 mb-1">
            AI feedback
            {detail.aiScore != null ? ` · ${detail.aiScore}%` : ""}
          </p>
          {detail.aiScore != null ? (
            <p className="text-gray-800 whitespace-pre-wrap">
              {detail.aiFeedback}
            </p>
          ) : (
            <p className="italic text-gray-500">Waiting for AI feedback…</p>
          )}
        </div>
      </div>
      <div className="mt-6 flex justify-end">
        <Button variant="outline" onClick={onClose}>
          Close
        </Button>
      </div>
    </Modal>
  );
}

export function ProduceMoreDetailsButton({
  detail,
}: {
  detail: ProduceFeedbackDetail;
}) {
  const [open, setOpen] = useState(false);
  return (
    <>
      <Button
        type="button"
        variant="outline"
        size="sm"
        onClick={() => setOpen(true)}
        data-testid="produce-more-details"
      >
        More details
      </Button>
      {open && (
        <ProduceFeedbackModal detail={detail} onClose={() => setOpen(false)} />
      )}
    </>
  );
}
