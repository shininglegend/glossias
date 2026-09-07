import { useCallback, useEffect, useState } from "react";
import { useApiService } from "../services/api";
import type {
  AttemptStage,
  ResetPhase,
  StudentAttemptSummary,
} from "../types/api";
import Button from "./ui/Button";
import Modal from "./ui/Modal";

const PHASE_LABELS: Record<ResetPhase, string> = {
  all: "Everything",
  exercises: "Exercises",
  video: "Watch",
  identify: "Identify",
  translate: "Translate",
  produce: "Produce",
  recall: "Recall",
  vocab: "Vocab (legacy)",
  grammar: "Grammar (legacy)",
};

/** Something the instructor has clicked Delete on but not yet confirmed. */
type PendingDelete =
  | { kind: "attempt"; attempt: StudentAttemptSummary }
  | { kind: "stage"; attempt: StudentAttemptSummary; stage: AttemptStage };

export function formatSeconds(seconds: number): string {
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = seconds % 60;
  const mm = h > 0 ? m.toString().padStart(2, "0") : m.toString();
  return `${h > 0 ? `${h}:` : ""}${mm}:${s.toString().padStart(2, "0")}`;
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleString(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  });
}

function pendingText(p: PendingDelete): string {
  if (p.kind === "stage") {
    return `Delete ${PHASE_LABELS[p.stage.phase]} from the in-progress attempt? The student will redo that phase.`;
  }
  const n = p.attempt.number;
  const later = n === 1 ? " Attempt 2 becomes the official score." : "";
  return `Delete attempt ${n} and its score? Later attempts move up one number.${later}`;
}

interface ManageAttemptsModalProps {
  storyId: string;
  student: { userId: string; name: string } | null;
  onClose: () => void;
  /** Fired after any successful deletion so the caller can refresh its table. */
  onChanged: () => void;
}

/**
 * Instructor view of one student's attempts on a story. Archived attempts are
 * frozen scores and delete as a whole; the in-progress attempt lists each
 * phase with data so they can be cleared one at a time.
 */
export function ManageAttemptsModal({
  storyId,
  student,
  onClose,
  onChanged,
}: ManageAttemptsModalProps) {
  const api = useApiService();
  const [attempts, setAttempts] = useState<StudentAttemptSummary[] | null>(
    null,
  );
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState<PendingDelete | null>(null);
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    if (!student) return;
    try {
      const response = await api.getStudentAttempts(storyId, student.userId);
      if (response.success && response.data) {
        setAttempts(response.data);
        setError(null);
      } else {
        setError(response.error || "Failed to load attempts");
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load attempts");
    }
  }, [api, storyId, student]);

  useEffect(() => {
    setAttempts(null);
    setError(null);
    setPending(null);
    void load();
  }, [load]);

  const confirmDelete = async () => {
    if (!pending || !student) return;
    setBusy(true);
    setError(null);
    try {
      const response =
        pending.kind === "attempt"
          ? await api.deleteStudentAttempt(
              storyId,
              student.userId,
              pending.attempt.number,
            )
          : await api.resetStudentProgress(
              storyId,
              student.userId,
              pending.stage.phase,
            );
      if (!response.success) {
        setError(response.error || "Delete failed");
        return;
      }
      setPending(null);
      onChanged();
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Delete failed");
    } finally {
      setBusy(false);
    }
  };

  const close = () => {
    if (busy) return;
    onClose();
  };

  return (
    <Modal
      isOpen={student !== null}
      onClose={close}
      title={student ? `Attempts — ${student.name}` : "Attempts"}
      description="Attempt 1 is the official score shown in the performance table. Deleting is permanent."
      closeDisabled={busy}
      className="sm:max-w-2xl"
    >
      <div className="mt-4">
        {error && (
          <p role="alert" className="mb-3 text-red-600">
            {error}
          </p>
        )}
        {attempts === null && !error ? (
          <p>Loading attempts...</p>
        ) : attempts ? (
          <table className="w-full border-collapse text-sm">
            <thead>
              <tr className="border-b border-gray-300 text-left text-gray-600">
                <th className="py-2 pr-3">Attempt</th>
                <th className="py-2 pr-3">Status</th>
                <th className="py-2 pr-3 text-right">Score</th>
                <th className="py-2 pr-3 text-right">Time</th>
                <th className="py-2" />
              </tr>
            </thead>
            <tbody>
              {attempts.map((attempt) => (
                <AttemptRow
                  key={attempt.number}
                  attempt={attempt}
                  disabled={busy || pending !== null}
                  onDelete={() => setPending({ kind: "attempt", attempt })}
                  onDeleteStage={(stage) =>
                    setPending({ kind: "stage", attempt, stage })
                  }
                />
              ))}
            </tbody>
          </table>
        ) : null}
      </div>

      {pending && (
        <div
          role="group"
          aria-label="Confirm delete"
          className="mt-4 rounded border border-rose-200 bg-rose-50 p-3"
        >
          <p className="text-sm text-rose-900">{pendingText(pending)}</p>
          <div className="mt-3 flex justify-end gap-3">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setPending(null)}
              disabled={busy}
            >
              Cancel
            </Button>
            <Button
              variant="danger"
              size="sm"
              onClick={confirmDelete}
              disabled={busy}
            >
              {busy ? "Deleting..." : "Delete"}
            </Button>
          </div>
        </div>
      )}

      <div className="mt-6 flex justify-end">
        <Button variant="outline" onClick={close} disabled={busy}>
          Close
        </Button>
      </div>
    </Modal>
  );
}

function AttemptRow({
  attempt,
  disabled,
  onDelete,
  onDeleteStage,
}: {
  attempt: StudentAttemptSummary;
  disabled: boolean;
  onDelete: () => void;
  onDeleteStage: (stage: AttemptStage) => void;
}) {
  const stages = attempt.stages ?? [];
  const started = stages.length > 0;
  const pendingGrading =
    attempt.is_current &&
    attempt.score !== undefined &&
    attempt.score.produce_segments_submitted >
      attempt.score.produce_segments_graded;

  return (
    <>
      <tr className="border-b border-gray-200 align-top">
        <td className="py-2 pr-3 font-semibold">
          {attempt.number}
          {attempt.number === 1 && !attempt.is_current && (
            <span className="ml-2 rounded bg-primary-50 px-1.5 py-0.5 text-xs font-medium text-primary-700">
              Official
            </span>
          )}
        </td>
        <td className="py-2 pr-3">
          {attempt.is_current ? (
            <span className="text-gray-600">
              {started ? "In progress" : "Not started"}
              {pendingGrading && " · Produce grading pending"}
            </span>
          ) : (
            <span>
              Completed
              {attempt.completed_at && (
                <span className="block text-xs text-gray-500">
                  {formatDate(attempt.completed_at)}
                </span>
              )}
            </span>
          )}
        </td>
        <td className="py-2 pr-3 text-right">
          {!attempt.is_current && attempt.score
            ? `${attempt.score.overall_accuracy.toFixed(1)}%`
            : "—"}
        </td>
        <td className="py-2 pr-3 text-right">
          {attempt.score
            ? formatSeconds(attempt.score.total_time_seconds)
            : "—"}
        </td>
        <td className="py-2 text-right">
          {!attempt.is_current && (
            <Button
              variant="outline"
              size="sm"
              onClick={onDelete}
              disabled={disabled}
              aria-label={`Delete attempt ${attempt.number}`}
            >
              Delete
            </Button>
          )}
        </td>
      </tr>
      {attempt.is_current && started && (
        <tr className="border-b border-gray-200">
          <td colSpan={5} className="pb-3 pl-6 pt-1">
            <ul className="space-y-1">
              {stages.map((stage) => (
                <li
                  key={stage.phase}
                  className="flex items-center justify-between gap-3 text-sm"
                >
                  <span>
                    <span className="font-medium">
                      {PHASE_LABELS[stage.phase]}
                    </span>
                    <span className="text-gray-500">
                      {" · "}
                      <span>{stage.detail}</span>
                      {stage.seconds > 0 &&
                        ` · ${formatSeconds(stage.seconds)}`}
                    </span>
                  </span>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => onDeleteStage(stage)}
                    disabled={disabled}
                    aria-label={`Delete ${PHASE_LABELS[stage.phase]} from the in-progress attempt`}
                  >
                    Delete
                  </Button>
                </li>
              ))}
            </ul>
          </td>
        </tr>
      )}
    </>
  );
}
