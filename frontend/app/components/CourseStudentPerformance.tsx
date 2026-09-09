import { useState, useEffect } from "react";
import { Link, useParams } from "react-router";
import { useApiService } from "../services/api";
import { useCoursesApi, type CourseSection } from "../services/coursesApi";
import type { Story } from "../types/api";
import Button from "./ui/Button";
import Badge from "./ui/Badge";
import { ManageAttemptsModal } from "./ManageAttemptsModal";

/** Mirrors models.CourseStudentPerformance; rows arrive best overall first. */
interface StudentPerformanceData {
  user_id: string;
  user_name: string;
  email: string;
  story_id: number;
  story_title: string;

  overall_accuracy: number;

  identify_correct: number;
  identify_incorrect: number;
  identify_accuracy: number;

  translation_completed: boolean;
  requested_lines: number[];

  produce_submitted: number;
  produce_total: number;
  produce_graded: number;
  produce_score: number;

  recall_correct: number;
  recall_incorrect: number;
  recall_attempts: number;
  recall_accuracy: number;

  vocab_correct: number;
  vocab_incorrect: number;
  vocab_accuracy: number;
  grammar_correct: number;
  grammar_incorrect: number;
  grammar_accuracy: number;

  video_time_seconds: number;
  identify_time_seconds: number;
  translation_time_seconds: number;
  produce_time_seconds: number;
  recall_time_seconds: number;
  vocab_time_seconds: number;
  grammar_time_seconds: number;
  total_time_seconds: number;
  attempt_count: number;
  sections?: CourseSection[];
}

function producePending(s: StudentPerformanceData): boolean {
  return s.produce_submitted > 0 && s.produce_graded === 0;
}

function hasLegacyData(rows: StudentPerformanceData[]): boolean {
  return rows.some(
    (s) =>
      s.vocab_correct +
        s.vocab_incorrect +
        s.grammar_correct +
        s.grammar_incorrect >
      0,
  );
}

function downloadCSV(
  data: StudentPerformanceData[],
  storyTitle: string,
  includeLegacy: boolean,
) {
  const headers = [
    "Student Name",
    "Email",
    "Attempts",
    "Overall Score (%)",
    "Total Time (seconds)",
    "Watch Time (seconds)",
    "Identify Accuracy (%)",
    "Identify Correct",
    "Identify Incorrect",
    "Identify Time (seconds)",
    "Translation Completed",
    "Translation Requested Lines",
    "Translation Time (seconds)",
    "Produce AI Score (%)",
    "Produce Segments Submitted",
    "Produce Segments Graded",
    "Produce Time (seconds)",
    "Recall Accuracy (%)",
    "Recall Attempts",
    "Recall Correct",
    "Recall Incorrect",
    "Recall Time (seconds)",
  ];
  if (includeLegacy) {
    headers.push(
      "Vocab Accuracy (%)",
      "Vocab Correct",
      "Vocab Incorrect",
      "Vocab Time (seconds)",
      "Grammar Accuracy (%)",
      "Grammar Correct",
      "Grammar Incorrect",
      "Grammar Time (seconds)",
    );
  }

  const rows = data.map((s) => {
    const row: (string | number)[] = [
      s.user_name,
      s.email,
      s.attempt_count ?? 0,
      s.overall_accuracy.toFixed(1),
      s.total_time_seconds,
      s.video_time_seconds,
      s.identify_accuracy.toFixed(1),
      s.identify_correct,
      s.identify_incorrect,
      s.identify_time_seconds,
      s.translation_completed ? "Yes" : "No",
      s.requested_lines?.join("; ") || "",
      s.translation_time_seconds,
      producePending(s) ? "Pending" : s.produce_score.toFixed(1),
      s.produce_submitted,
      s.produce_graded,
      s.produce_time_seconds,
      s.recall_accuracy.toFixed(1),
      s.recall_attempts,
      s.recall_correct,
      s.recall_incorrect,
      s.recall_time_seconds,
    ];
    if (includeLegacy) {
      row.push(
        s.vocab_accuracy.toFixed(1),
        s.vocab_correct,
        s.vocab_incorrect,
        s.vocab_time_seconds,
        s.grammar_accuracy.toFixed(1),
        s.grammar_correct,
        s.grammar_incorrect,
        s.grammar_time_seconds,
      );
    }
    return row;
  });

  const csv = [headers, ...rows]
    .map((row) => row.map((cell) => `"${cell}"`).join(","))
    .join("\n");

  const blob = new Blob([csv], { type: "text/csv" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = `${storyTitle.replace(/[^a-z0-9]/gi, "_")}_performance.csv`;
  a.click();
  URL.revokeObjectURL(url);
}

function formatTime(seconds: number): string {
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  const remainingSeconds = seconds % 60;

  if (hours > 0) {
    return `${hours}:${minutes.toString().padStart(2, "0")}:${remainingSeconds.toString().padStart(2, "0")}`;
  }
  return `${minutes}:${remainingSeconds.toString().padStart(2, "0")}`;
}

function formatAccuracy(accuracy: number): string {
  return `${accuracy.toFixed(1)}%`;
}

function accuracyClass(accuracy: number): string {
  if (accuracy >= 80) return "text-green-600";
  if (accuracy >= 60) return "text-yellow-600";
  return "text-red-600";
}

function AccuracyCell({
  accuracy,
  detail,
  time,
}: {
  accuracy: number;
  detail: string;
  time: number;
}) {
  return (
    <td className="border border-gray-300 p-3 text-center">
      <span className={`font-semibold ${accuracyClass(accuracy)}`}>
        {formatAccuracy(accuracy)}
      </span>
      <div className="text-xs text-gray-500">{detail}</div>
      <div className="text-xs text-gray-400">{formatTime(time)}</div>
    </td>
  );
}

const TH = "border border-gray-300 p-3 text-center";

export function CourseStudentPerformance() {
  const { id } = useParams<{ id: string }>();
  const api = useApiService();
  const coursesApi = useCoursesApi();
  const [performanceData, setPerformanceData] = useState<
    StudentPerformanceData[]
  >([]);
  const [stories, setStories] = useState<Story[]>([]);
  const [selectedStoryId, setSelectedStoryId] = useState<number | null>(null);
  const [loadingStories, setLoadingStories] = useState(true);
  const [loadingPerformance, setLoadingPerformance] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [statusFilter, setStatusFilter] = useState<string>("active");
  const [sections, setSections] = useState<CourseSection[]>([]);
  const [sectionFilter, setSectionFilter] = useState<string>("all");
  const [refreshKey, setRefreshKey] = useState(0);
  // Student whose attempts are open in the Manage dialog.
  const [managing, setManaging] = useState<{
    userId: string;
    name: string;
  } | null>(null);

  useEffect(() => {
    const fetchStories = async () => {
      if (!id) {
        setError("Course ID is required");
        setLoadingStories(false);
        return;
      }

      try {
        const [response, sectionRes] = await Promise.all([
          api.getCourseStories(id),
          coursesApi.getCourseSections(Number(id)).catch(() => ({
            sections: [] as CourseSection[],
          })),
        ]);
        if (response.success && response.data) {
          setStories(response.data);
        } else {
          setError(response.error || "Failed to fetch course stories");
        }
        setSections(sectionRes.sections || []);
      } catch {
        setError("Failed to fetch course stories");
      } finally {
        setLoadingStories(false);
      }
    };

    fetchStories();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id]);

  useEffect(() => {
    const fetchPerformance = async (attempt = 0) => {
      if (!selectedStoryId) {
        setPerformanceData([]);
        return;
      }

      setLoadingPerformance(true);
      setError(null);

      try {
        const response = await api.getStoryStudentPerformance(
          selectedStoryId.toString(),
          statusFilter,
          id,
          sectionFilter,
        );
        if (response.success && response.data) {
          setPerformanceData(response.data as StudentPerformanceData[]);
        } else {
          if (
            response.error?.includes("504") ||
            response.error?.includes("timeout")
          ) {
            if (attempt === 0) {
              setError("Request timed out. Retrying...");
              setTimeout(() => fetchPerformance(1), 1000);
              return;
            } else {
              setError(
                "Request timed out after retry. The server may be overloaded.",
              );
            }
          } else {
            setError(
              response.error || "Failed to fetch student performance data",
            );
          }
        }
      } catch (err) {
        const errorMessage =
          err instanceof Error ? err.message : "Unknown error";
        if (
          errorMessage.includes("504") ||
          errorMessage.toLowerCase().includes("timeout")
        ) {
          if (attempt === 0) {
            setError("Request timed out. Retrying...");
            setTimeout(() => fetchPerformance(1), 1000);
            return;
          } else {
            setError(
              "Request timed out after retry. The server may be overloaded.",
            );
          }
        } else {
          setError("Failed to fetch student performance data");
        }
      } finally {
        setLoadingPerformance(false);
      }
    };

    fetchPerformance();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedStoryId, statusFilter, sectionFilter, refreshKey]);

  // The server orders rows best overall score first (ties: least time, then
  // email), so the table and the CSV share one ordering.
  const showLegacy = hasLegacyData(performanceData);

  if (loadingStories) {
    return (
      <div className="container">
        <h1>Student Performance</h1>
        <p>Loading course stories...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="container">
        <h1>Student Performance</h1>
        <p className="text-red-600">Error: {error}</p>
      </div>
    );
  }

  return (
    <div className="container">
      <h1>Student Performance</h1>

      {stories.length === 0 ? (
        <p>No stories found for this course.</p>
      ) : (
        <div>
          <div className="mb-4">
            <label htmlFor="story-select" className="block font-semibold mb-2">
              Select Story:
            </label>
            <select
              id="story-select"
              value={selectedStoryId || ""}
              onChange={(e) =>
                setSelectedStoryId(Number(e.target.value) || null)
              }
              className="border border-gray-300 rounded px-3 py-2 w-full max-w-md"
            >
              <option value="">-- Select a story --</option>
              {stories.map((story) => (
                <option
                  key={story.metadata.storyId}
                  value={story.metadata.storyId}
                >
                  {typeof story.metadata.title === "string"
                    ? story.metadata.title
                    : story.metadata.title?.en || "Untitled"}
                </option>
              ))}
            </select>
            {selectedStoryId && (
              <div className="mt-4 flex gap-4 items-center">
                <div>
                  <label
                    htmlFor="status-filter"
                    className="block font-semibold mb-2"
                  >
                    Filter by Status:
                  </label>
                  <select
                    id="status-filter"
                    value={statusFilter}
                    onChange={(e) => setStatusFilter(e.target.value)}
                    className="border border-gray-300 rounded px-3 py-2"
                  >
                    <option value="active">Current Students</option>
                    <option value="">All Students</option>
                    <option value="future">Future Students</option>
                    <option value="past">Past Students</option>
                  </select>
                </div>
                {sections.length > 0 && (
                  <div>
                    <label
                      htmlFor="section-filter"
                      className="block font-semibold mb-2"
                    >
                      Section:
                    </label>
                    <select
                      id="section-filter"
                      value={sectionFilter}
                      onChange={(e) => setSectionFilter(e.target.value)}
                      className="border border-gray-300 rounded px-3 py-2"
                    >
                      <option value="all">All Sections</option>
                      <option value="unassigned">Unassigned</option>
                      {sections.map((section) => (
                        <option
                          key={section.course_id}
                          value={section.course_id}
                        >
                          {section.course_number} — {section.name}
                        </option>
                      ))}
                    </select>
                  </div>
                )}
                <div>
                  <label className="block font-semibold mb-2">&nbsp;</label>
                  <button
                    onClick={() => {
                      const story = stories.find(
                        (s) => s.metadata.storyId === selectedStoryId,
                      );
                      const title = story?.metadata.title;
                      const titleStr =
                        typeof title === "string"
                          ? title
                          : (title as { [key: string]: string })?.en || "story";
                      downloadCSV(performanceData, titleStr, showLegacy);
                    }}
                    disabled={performanceData.length === 0}
                    className="px-4 py-2 bg-primary-500 text-white rounded hover:bg-primary-600 disabled:bg-gray-400 disabled:cursor-not-allowed"
                  >
                    Download CSV
                  </button>
                </div>
              </div>
            )}
          </div>

          {!selectedStoryId ? (
            <p>Please select a story to view performance data.</p>
          ) : loadingPerformance ? (
            <p>Loading performance data...</p>
          ) : performanceData.length === 0 ? (
            <p>No performance data found for the selected story.</p>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full border-collapse border border-gray-300 bg-white">
                <thead>
                  <tr className="bg-gray-100">
                    <th className="border border-gray-300 p-3 text-left">
                      Student
                    </th>
                    <th className={TH}>Attempts</th>
                    <th className={TH}>Overall</th>
                    <th className={TH}>Total Time</th>
                    <th className={TH}>Watch</th>
                    <th className={TH}>Identify</th>
                    <th className={TH}>Translate</th>
                    <th className={TH}>Produce</th>
                    <th className={TH}>Recall</th>
                    {showLegacy && <th className={TH}>Vocab (legacy)</th>}
                    {showLegacy && <th className={TH}>Grammar (legacy)</th>}
                  </tr>
                </thead>
                <tbody>
                  {performanceData.map((student) => (
                    <tr key={student.user_id} className="hover:bg-gray-50">
                      <td className="border border-gray-300 p-3">
                        <div>
                          <Link
                            to={`/admin/stories/${selectedStoryId}/students/${encodeURIComponent(student.user_id)}`}
                            className="font-semibold text-primary-600 hover:underline"
                            title="View this student's answers and submissions"
                          >
                            {student.user_name || student.email}
                          </Link>
                          <div className="text-sm text-gray-600">
                            {student.email}
                          </div>
                          {student.sections?.length ? (
                            <div className="mt-1 flex flex-wrap gap-1">
                              {student.sections.map((section) => (
                                <Badge key={section.course_id} variant="muted">
                                  {section.course_number}
                                </Badge>
                              ))}
                            </div>
                          ) : null}
                        </div>
                      </td>
                      <td className="border border-gray-300 p-3 text-center">
                        <div className="font-semibold">
                          {student.attempt_count ?? 0}
                        </div>
                        <Button
                          variant="outline"
                          size="sm"
                          className="mt-1"
                          onClick={() =>
                            setManaging({
                              userId: student.user_id,
                              name: student.user_name || student.email,
                            })
                          }
                          aria-label={`Manage attempts for ${student.user_name || student.email}`}
                        >
                          Manage
                        </Button>
                      </td>
                      <td className="border border-gray-300 p-3 text-center">
                        <span
                          className={`font-bold ${accuracyClass(student.overall_accuracy)}`}
                        >
                          {formatAccuracy(student.overall_accuracy)}
                        </span>
                        {producePending(student) && (
                          <div className="text-xs text-gray-500">
                            excl. Produce (pending)
                          </div>
                        )}
                      </td>
                      <td className="border border-gray-300 p-3 text-center font-semibold">
                        {formatTime(student.total_time_seconds)}
                      </td>
                      <td className="border border-gray-300 p-3 text-center">
                        {formatTime(student.video_time_seconds)}
                      </td>
                      <AccuracyCell
                        accuracy={student.identify_accuracy}
                        detail={`${student.identify_correct} correct / ${student.identify_incorrect} incorrect`}
                        time={student.identify_time_seconds}
                      />
                      <td className="border border-gray-300 p-3 text-center">
                        <span
                          className={
                            student.translation_completed
                              ? "text-green-600"
                              : "text-red-600"
                          }
                        >
                          {student.translation_completed ? "✓" : "✗"}
                        </span>
                        {student.requested_lines &&
                          student.requested_lines.length > 0 && (
                            <div className="text-xs text-gray-500 mt-1">
                              Lines: {student.requested_lines.join(", ")}
                            </div>
                          )}
                        <div className="text-xs text-gray-400">
                          {formatTime(student.translation_time_seconds)}
                        </div>
                      </td>
                      <td className="border border-gray-300 p-3 text-center">
                        {student.produce_submitted === 0 ? (
                          <span className="text-red-600">✗</span>
                        ) : producePending(student) ? (
                          <span className="font-semibold text-gray-500">
                            Pending
                          </span>
                        ) : (
                          <span
                            className={`font-semibold ${accuracyClass(student.produce_score)}`}
                          >
                            {formatAccuracy(student.produce_score)}
                          </span>
                        )}
                        <div className="text-xs text-gray-500">
                          {student.produce_submitted}/{student.produce_total}{" "}
                          submitted
                          {student.produce_submitted > 0 &&
                            ` · ${student.produce_graded} graded`}
                        </div>
                        <div className="text-xs text-gray-400">
                          {formatTime(student.produce_time_seconds)}
                        </div>
                      </td>
                      <AccuracyCell
                        accuracy={student.recall_accuracy}
                        detail={`${student.recall_attempts} attempt${student.recall_attempts === 1 ? "" : "s"} · ${student.recall_correct} correct / ${student.recall_incorrect} incorrect`}
                        time={student.recall_time_seconds}
                      />
                      {showLegacy && (
                        <AccuracyCell
                          accuracy={student.vocab_accuracy}
                          detail={`${student.vocab_correct} correct / ${student.vocab_incorrect} incorrect`}
                          time={student.vocab_time_seconds}
                        />
                      )}
                      {showLegacy && (
                        <AccuracyCell
                          accuracy={student.grammar_accuracy}
                          detail={`${student.grammar_correct} correct / ${student.grammar_incorrect} incorrect`}
                          time={student.grammar_time_seconds}
                        />
                      )}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}

      {selectedStoryId && (
        <ManageAttemptsModal
          storyId={selectedStoryId.toString()}
          student={managing}
          onClose={() => setManaging(null)}
          onChanged={() => setRefreshKey((k) => k + 1)}
        />
      )}
    </div>
  );
}
