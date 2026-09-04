import { useState, useEffect } from "react";
import { useNavigate, useParams } from "react-router";
import { useApiService } from "../services/api";
import Button from "./ui/Button";

/** Mirrors models.StudentStoryDrilldown. */
interface DrilldownData {
  user_id: string;
  user_name: string;
  email: string;
  story_id: number;
  story_title: string;

  identify_answers: IdentifyAnswer[];
  translate: TranslateDetail;
  produce_segments: ProduceSegment[];
  recall_attempts: RecallAttempt[];

  time: PhaseTime;
}

interface IdentifyAnswer {
  line_number: number;
  correct: boolean;
  target_word: string;
  selected_word?: string;
  attempted_at: string;
}

interface TranslateDetail {
  started: boolean;
  completed: boolean;
  requested_lines: number[];
  completed_at?: string;
}

interface ProduceSegment {
  segment_order: number;
  hebrew_text: string;
  reference_english: string;
  grammar_point_name?: string;
  submissions: ProduceSubmission[];
}

interface ProduceSubmission {
  student_text: string;
  ai_score?: number;
  ai_feedback?: string;
  graded_at?: string;
  created_at: string;
}

interface RecallAttempt {
  attempted_at: string;
  all_correct: boolean;
  placements: RecallPlacement[];
}

interface RecallPlacement {
  selected_position: number;
  correct_position: number;
  hebrew_text: string;
  correct: boolean;
}

interface PhaseTime {
  video_seconds: number;
  identify_seconds: number;
  translate_seconds: number;
  produce_seconds: number;
  recall_seconds: number;
  vocab_seconds: number;
  grammar_seconds: number;
}

function formatTime(seconds: number): string {
  const minutes = Math.floor(seconds / 60);
  const remaining = seconds % 60;
  return `${minutes}:${remaining.toString().padStart(2, "0")}`;
}

function formatWhen(iso: string): string {
  const date = new Date(iso);
  if (isNaN(date.getTime())) return "";
  return date.toLocaleString();
}

function ResultMark({ correct }: { correct: boolean }) {
  return (
    <span className={correct ? "text-green-600" : "text-red-600"}>
      {correct ? "✓" : "✗"}
    </span>
  );
}

function SectionCard({
  title,
  time,
  children,
}: {
  title: string;
  time?: number;
  children: React.ReactNode;
}) {
  return (
    <section className="mb-6 rounded border border-gray-300 bg-white p-4">
      <h2 className="mb-3 flex items-baseline justify-between text-lg font-semibold">
        {title}
        {time !== undefined && (
          <span className="text-sm font-normal text-gray-500">
            {formatTime(time)} spent
          </span>
        )}
      </h2>
      {children}
    </section>
  );
}

const TH = "border border-gray-300 p-2 text-center bg-gray-100";
const TD = "border border-gray-300 p-2 text-center";

export function StudentStoryDrilldown() {
  const { id, userId } = useParams<{ id: string; userId: string }>();
  const api = useApiService();
  const navigate = useNavigate();
  const [data, setData] = useState<DrilldownData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchDrilldown = async () => {
      if (!id || !userId) {
        setError("Story ID and student ID are required");
        setLoading(false);
        return;
      }
      try {
        const response = await api.getStudentStoryDrilldown(id, userId);
        if (response.success && response.data) {
          setData(response.data as DrilldownData);
        } else {
          setError(response.error || "Failed to fetch student detail");
        }
      } catch {
        setError("Failed to fetch student detail");
      } finally {
        setLoading(false);
      }
    };
    fetchDrilldown();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id, userId]);

  if (loading) {
    return (
      <div className="container">
        <h1>Student Detail</h1>
        <p>Loading…</p>
      </div>
    );
  }

  if (error || !data) {
    return (
      <div className="container">
        <h1>Student Detail</h1>
        <p className="text-red-600">Error: {error || "No data"}</p>
        <Button variant="outline" onClick={() => navigate(-1)}>
          Back
        </Button>
      </div>
    );
  }

  const legacyTime = data.time.vocab_seconds + data.time.grammar_seconds;

  return (
    <div className="container">
      <div className="mb-4 flex items-start justify-between gap-4">
        <div>
          <h1>{data.user_name || data.email}</h1>
          <p className="text-gray-600">
            {data.email}
            {data.story_title && (
              <>
                {" · "}
                <span className="font-semibold">{data.story_title}</span>
              </>
            )}
          </p>
        </div>
        <Button variant="outline" onClick={() => navigate(-1)}>
          Back
        </Button>
      </div>

      <SectionCard title="Watch" time={data.time.video_seconds}>
        <p className="text-sm text-gray-600">
          Time on the video page. There are no answers to show for this phase.
        </p>
      </SectionCard>

      <SectionCard title="Identify" time={data.time.identify_seconds}>
        {data.identify_answers.length === 0 ? (
          <p className="text-sm text-gray-600">No picture-quiz picks yet.</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full border-collapse border border-gray-300">
              <thead>
                <tr>
                  <th className={TH}>Line</th>
                  <th className={TH}>Target word</th>
                  <th className={TH}>Picked</th>
                  <th className={TH}>Result</th>
                  <th className={TH}>When</th>
                </tr>
              </thead>
              <tbody>
                {data.identify_answers.map((answer, i) => (
                  <tr key={i}>
                    <td className={TD}>{answer.line_number}</td>
                    <td className={TD} dir="rtl" lang="he">
                      {answer.target_word}
                    </td>
                    <td className={TD} dir="rtl" lang="he">
                      {answer.correct
                        ? answer.target_word
                        : answer.selected_word}
                    </td>
                    <td className={TD}>
                      <ResultMark correct={answer.correct} />
                    </td>
                    <td className={`${TD} text-sm text-gray-500`}>
                      {formatWhen(answer.attempted_at)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </SectionCard>

      <SectionCard title="Translate" time={data.time.translate_seconds}>
        {!data.translate.started ? (
          <p className="text-sm text-gray-600">Not started.</p>
        ) : (
          <div className="text-sm">
            <p>
              Status:{" "}
              {data.translate.completed ? (
                <span className="font-semibold text-green-600">
                  Completed
                  {data.translate.completed_at &&
                    ` (${formatWhen(data.translate.completed_at)})`}
                </span>
              ) : (
                <span className="font-semibold text-yellow-600">
                  In progress
                </span>
              )}
            </p>
            <p className="mt-1">
              Requested translations:{" "}
              {data.translate.requested_lines.length > 0 ? (
                <>
                  {data.translate.requested_lines.length} line
                  {data.translate.requested_lines.length === 1 ? "" : "s"}{" "}
                  <span className="text-gray-600">
                    (lines {data.translate.requested_lines.join(", ")})
                  </span>
                </>
              ) : (
                "none"
              )}
            </p>
          </div>
        )}
      </SectionCard>

      <SectionCard title="Produce" time={data.time.produce_seconds}>
        {data.produce_segments.length === 0 ? (
          <p className="text-sm text-gray-600">
            This story has no Produce segments.
          </p>
        ) : (
          data.produce_segments.map((segment) => (
            <div
              key={segment.segment_order}
              className="mb-4 rounded border border-gray-200 p-3 last:mb-0"
            >
              <h3 className="font-semibold">
                Segment {segment.segment_order}
                {segment.grammar_point_name && (
                  <span className="ml-2 text-sm font-normal text-gray-500">
                    {segment.grammar_point_name}
                  </span>
                )}
              </h3>
              <p className="mt-1" dir="rtl" lang="he">
                {segment.hebrew_text}
              </p>
              <p className="text-sm text-gray-600">
                Reference: {segment.reference_english}
              </p>
              {segment.submissions.length === 0 ? (
                <p className="mt-2 text-sm text-gray-600">No submissions.</p>
              ) : (
                <ul className="mt-2 space-y-2">
                  {segment.submissions.map((submission, i) => (
                    <li key={i} className="rounded bg-gray-50 p-2 text-sm">
                      <p className="whitespace-pre-wrap">
                        {submission.student_text || (
                          <span className="italic text-gray-500">
                            (blank submission)
                          </span>
                        )}
                      </p>
                      <p className="mt-1 text-gray-600">
                        {submission.ai_score !== undefined &&
                        submission.ai_score !== null ? (
                          <>
                            <span className="font-semibold">
                              Score: {submission.ai_score}%
                            </span>
                            {submission.ai_feedback &&
                              ` — ${submission.ai_feedback}`}
                          </>
                        ) : (
                          <span className="italic">Grading pending</span>
                        )}
                      </p>
                      <p className="text-xs text-gray-400">
                        {formatWhen(submission.created_at)}
                      </p>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          ))
        )}
      </SectionCard>

      <SectionCard title="Recall" time={data.time.recall_seconds}>
        {data.recall_attempts.length === 0 ? (
          <p className="text-sm text-gray-600">No ordering attempts yet.</p>
        ) : (
          data.recall_attempts.map((attempt, i) => (
            <div
              key={i}
              className="mb-4 rounded border border-gray-200 p-3 last:mb-0"
            >
              <h3 className="font-semibold">
                Attempt {i + 1}{" "}
                {attempt.all_correct ? (
                  <span className="text-sm font-normal text-green-600">
                    all correct
                  </span>
                ) : (
                  <span className="text-sm font-normal text-red-600">
                    {attempt.placements.filter((p) => !p.correct).length}{" "}
                    misplaced
                  </span>
                )}
                <span className="ml-2 text-xs font-normal text-gray-400">
                  {formatWhen(attempt.attempted_at)}
                </span>
              </h3>
              <div className="mt-2 overflow-x-auto">
                <table className="w-full border-collapse border border-gray-300 text-sm">
                  <thead>
                    <tr>
                      <th className={TH}>Placed at</th>
                      <th className={TH}>Sentence</th>
                      <th className={TH}>Belongs at</th>
                      <th className={TH}>Result</th>
                    </tr>
                  </thead>
                  <tbody>
                    {attempt.placements.map((placement) => (
                      <tr key={placement.selected_position}>
                        <td className={TD}>{placement.selected_position}</td>
                        <td className={TD} dir="rtl" lang="he">
                          {placement.hebrew_text}
                        </td>
                        <td className={TD}>{placement.correct_position}</td>
                        <td className={TD}>
                          <ResultMark correct={placement.correct} />
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          ))
        )}
      </SectionCard>

      {legacyTime > 0 && (
        <SectionCard title="Legacy phases">
          <p className="text-sm text-gray-600">
            Time from the pre-2026 flow: Vocab{" "}
            {formatTime(data.time.vocab_seconds)}, Grammar{" "}
            {formatTime(data.time.grammar_seconds)}.
          </p>
        </SectionCard>
      )}
    </div>
  );
}
