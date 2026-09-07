import { useCallback, useEffect, useState } from "react";
import { useApiService } from "../services/api";
import type {
  AttemptScoreSummary,
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
      className="sm:!max-w-3xl"
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
          <ul className="space-y-2">
            {attempts.map((attempt) => (
              <AttemptCard
                key={attempt.number}
                attempt={attempt}
                disabled={busy || pending !== null}
                onDelete={() => setPending({ kind: "attempt", attempt })}
                onDeleteStage={(stage) =>
                  setPending({ kind: "stage", attempt, stage })
                }
              />
            ))}
          </ul>
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

type BreakdownRow = {
  phase: ResetPhase;
  headline: string;
  stats: string[];
  extra?: string[];
  seconds: number;
};

function pct(value: number | undefined): string {
  return value === undefined ? "" : `${value.toFixed(1)}%`;
}

function countPair(
  correct: number | undefined,
  incorrect: number | undefined,
  total?: number,
): string[] {
  const stats = [`${correct ?? 0} correct`, `${incorrect ?? 0} incorrect`];
  if (total !== undefined && total > 0) {
    stats.push(`of ${total}`);
  }
  return stats;
}

/** Phase rows for the attempt dropdown: snapshot first, live stages as fallback. */
export function attemptBreakdown(
  attempt: StudentAttemptSummary,
): BreakdownRow[] {
  const fromScore = scoreBreakdown(attempt.score);
  if (fromScore.length > 0) return fromScore;
  return (attempt.stages ?? []).map((stage) => ({
    phase: stage.phase,
    headline: stage.detail,
    stats: [],
    seconds: stage.seconds,
  }));
}

function scoreBreakdown(
  score: AttemptScoreSummary | undefined,
): BreakdownRow[] {
  if (!score) return [];
  const rows: BreakdownRow[] = [];
  const add = (
    phase: ResetPhase,
    seconds: number | undefined,
    hasWork: boolean,
    row: Omit<BreakdownRow, "phase" | "seconds">,
  ) => {
    const time = seconds ?? 0;
    if (hasWork || time > 0) {
      rows.push({ phase, seconds: time, ...row });
    }
  };

  add("video", score.video_time_seconds, false, {
    headline: "Watched",
    stats: ["No answers"],
  });

  add(
    "identify",
    score.identify_time_seconds,
    (score.identify_correct_count ?? 0) +
      (score.identify_incorrect_count ?? 0) >
      0,
    {
      headline: pct(score.identify_accuracy),
      stats: countPair(
        score.identify_correct_count,
        score.identify_incorrect_count,
        score.identify_total,
      ),
    },
  );

  const lines = score.requested_lines ?? [];
  add(
    "translate",
    score.translation_time_seconds,
    Boolean(score.translation_completed) || lines.length > 0,
    {
      headline: score.translation_completed ? "Completed" : "Started",
      stats: [
        lines.length === 0
          ? "No lines requested"
          : `${lines.length} line${lines.length === 1 ? "" : "s"} requested`,
        ...(lines.length > 0 ? [`${lines.join(", ")}`] : []),
      ],
    },
  );

  const produceExtra = (score.produce_segments ?? []).map((seg) => {
    const n = seg.segment_order;
    if (seg.ai_score === null || seg.ai_score === undefined) {
      return `Passage ${n}: ungraded`;
    }
    return `Passage ${n}: ${seg.ai_score}`;
  });
  add(
    "produce",
    score.produce_time_seconds,
    score.produce_segments_submitted > 0,
    {
      headline:
        score.produce_segments_graded > 0
          ? pct(score.produce_score)
          : score.produce_segments_submitted > 0
            ? "Grading pending"
            : "",
      stats: [
        `${score.produce_segments_submitted}/${score.produce_total ?? 0} submitted`,
        `${score.produce_segments_graded} graded`,
      ],
      extra: produceExtra.length > 0 ? produceExtra : undefined,
    },
  );

  const recallAttempts = score.recall_attempts ?? 0;
  add(
    "recall",
    score.recall_time_seconds,
    (score.recall_correct_count ?? 0) + (score.recall_incorrect_count ?? 0) > 0,
    {
      headline: pct(score.recall_accuracy),
      stats: [
        ...countPair(
          score.recall_correct_count,
          score.recall_incorrect_count,
          score.recall_total,
        ),
        `${recallAttempts} attempt${recallAttempts === 1 ? "" : "s"}`,
      ],
    },
  );

  add(
    "vocab",
    score.vocab_time_seconds,
    (score.vocab_correct_count ?? 0) + (score.vocab_incorrect_count ?? 0) > 0,
    {
      headline: pct(score.vocab_accuracy),
      stats: countPair(score.vocab_correct_count, score.vocab_incorrect_count),
    },
  );
  add(
    "grammar",
    score.grammar_time_seconds,
    (score.grammar_correct_count ?? 0) + (score.grammar_incorrect_count ?? 0) >
      0,
    {
      headline: pct(score.grammar_accuracy),
      stats: countPair(
        score.grammar_correct_count,
        score.grammar_incorrect_count,
      ),
    },
  );
  return rows;
}

function AttemptCard({
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
  const rows = attemptBreakdown(attempt);
  const pendingGrading =
    attempt.is_current &&
    attempt.score !== undefined &&
    attempt.score.produce_segments_submitted >
      attempt.score.produce_segments_graded;
  const status = attempt.is_current
    ? `${started ? "In progress" : "Not started"}${
        pendingGrading ? " · Produce grading pending" : ""
      }`
    : "Completed";
  const scoreLabel =
    !attempt.is_current && attempt.score
      ? `${attempt.score.overall_accuracy.toFixed(1)}%`
      : "—";
  const timeLabel = attempt.score
    ? formatSeconds(attempt.score.total_time_seconds)
    : "—";

  const header = (
    <>
      <span className="w-8 shrink-0 font-semibold">{attempt.number}</span>
      <span className="min-w-0 flex-1">
        <span className="text-slate-700">{status}</span>
        {attempt.number === 1 && !attempt.is_current && (
          <span className="ml-2 rounded bg-primary-50 px-1.5 py-0.5 text-xs font-medium text-primary-700">
            Official
          </span>
        )}
        {attempt.completed_at && (
          <span className="mt-0.5 block text-xs text-slate-500">
            {formatDate(attempt.completed_at)}
          </span>
        )}
      </span>
      <span className="w-16 shrink-0 text-right tabular-nums">
        {scoreLabel}
      </span>
      <span className="w-20 shrink-0 text-right tabular-nums text-slate-600">
        {timeLabel}
      </span>
    </>
  );

  const deleteAttempt = !attempt.is_current ? (
    <Button
      variant="outline"
      size="sm"
      className="shrink-0"
      onClick={(e) => {
        e.preventDefault();
        e.stopPropagation();
        onDelete();
      }}
      disabled={disabled}
      aria-label={`Delete attempt ${attempt.number}`}
    >
      Delete
    </Button>
  ) : null;

  if (rows.length === 0) {
    return (
      <li className="flex items-center gap-3 rounded-lg border border-slate-200 px-3 py-2.5 text-sm">
        <span className="w-5 shrink-0" aria-hidden="true" />
        {header}
        {deleteAttempt}
      </li>
    );
  }

  return (
    <li className="rounded-lg border border-slate-200 text-sm">
      <details className="group" open={attempt.is_current && started}>
        <summary
          aria-label={`Attempt ${attempt.number}, ${status}`}
          className="flex cursor-pointer list-none items-center gap-3 px-3 py-2.5 select-none hover:bg-slate-50 [&::-webkit-details-marker]:hidden"
        >
          <span
            className="material-icons w-5 shrink-0 text-base text-slate-400 transition-transform group-open:rotate-180"
            aria-hidden="true"
          >
            expand_more
          </span>
          {header}
          {deleteAttempt}
        </summary>
        <div className="px-3 pb-3 pl-11">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left text-xs text-slate-500">
                <th className="w-24 py-1 pr-3 font-medium">Phase</th>
                <th className="py-1 pr-3 font-medium">Score</th>
                <th className="w-16 py-1 pr-3 text-right font-medium">Time</th>
                {attempt.is_current ? <th className="w-16 py-1" /> : null}
              </tr>
            </thead>
            <tbody>
              {rows.map((row) => {
                const stage = stages.find((s) => s.phase === row.phase);
                return (
                  <tr key={row.phase}>
                    <td className="py-1.5 pr-3 font-medium">
                      {PHASE_LABELS[row.phase]}
                    </td>
                    <td className="py-1.5 pr-3 text-slate-600">
                      <div className="flex flex-wrap items-baseline gap-x-4 gap-y-0.5">
                        {row.headline ? (
                          <span className="font-semibold text-slate-800">
                            {row.headline}
                          </span>
                        ) : null}
                        {row.stats.map((stat) => (
                          <span key={stat}>{stat}</span>
                        ))}
                      </div>
                      {row.extra && row.extra.length > 0 ? (
                        <div className="mt-1 flex flex-wrap gap-x-4 gap-y-0.5 text-xs text-slate-500">
                          {row.extra.map((item) => (
                            <span key={item}>{item}</span>
                          ))}
                        </div>
                      ) : null}
                    </td>
                    <td className="py-1.5 pr-3 text-right tabular-nums text-slate-600">
                      {row.seconds > 0 ? formatSeconds(row.seconds) : "—"}
                    </td>
                    {attempt.is_current ? (
                      <td className="py-1.5 text-right">
                        {stage ? (
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => onDeleteStage(stage)}
                            disabled={disabled}
                            aria-label={`Delete ${PHASE_LABELS[row.phase]} from the in-progress attempt`}
                          >
                            Delete
                          </Button>
                        ) : null}
                      </td>
                    ) : null}
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </details>
    </li>
  );
}
