import { useState, useEffect, useCallback } from "react";
import { useParams } from "react-router";
import { useApiService } from "../services/api";
import type { ResetPhase, Story } from "../types/api";
import Button from "./ui/Button";
import Modal from "./ui/Modal";

const RESET_PHASE_OPTIONS: { value: ResetPhase; label: string }[] = [
  { value: "all", label: "Entire story (all phases + time)" },
  { value: "video", label: "Watch (time only)" },
  { value: "identify", label: "Identify" },
  { value: "translate", label: "Translate" },
  { value: "produce", label: "Produce" },
  { value: "recall", label: "Recall" },
  { value: "vocab", label: "Vocab (legacy)" },
  { value: "grammar", label: "Grammar (legacy)" },
];

interface StudentPerformanceData {
  user_id: string;
  user_name: string;
  email: string;
  story_id: number;
  story_title: string;
  vocab_correct: number;
  vocab_incorrect: number;
  vocab_accuracy: number;
  grammar_correct: number;
  grammar_incorrect: number;
  grammar_accuracy: number;
  translation_completed: boolean;
  requested_lines: number[];
  vocab_time_seconds: number;
  grammar_time_seconds: number;
  translation_time_seconds: number;
  video_time_seconds: number;
  total_time_seconds: number;
}

function downloadCSV(data: StudentPerformanceData[], storyTitle: string) {
  const headers = [
    "Student Name",
    "Email",
    "Total Time (seconds)",
    "Video Time (seconds)",
    "Vocab Accuracy (%)",
    "Vocab Correct",
    "Vocab Incorrect",
    "Vocab Time (seconds)",
    "Grammar Accuracy (%)",
    "Grammar Correct",
    "Grammar Incorrect",
    "Grammar Time (seconds)",
    "Translation Completed",
    "Translation Requested Lines",
    "Translation Time (seconds)",
  ];

  const rows = data.map((s) => [
    s.user_name,
    s.email,
    s.total_time_seconds,
    s.video_time_seconds,
    s.vocab_accuracy.toFixed(1),
    s.vocab_correct,
    s.vocab_incorrect,
    s.vocab_time_seconds,
    s.grammar_accuracy.toFixed(1),
    s.grammar_correct,
    s.grammar_incorrect,
    s.grammar_time_seconds,
    s.translation_completed ? "Yes" : "No",
    s.requested_lines?.join("; ") || "",
    s.translation_time_seconds,
  ]);

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

export function CourseStudentPerformance() {
  const { id } = useParams<{ id: string }>();
  const api = useApiService();
  const [performanceData, setPerformanceData] = useState<
    StudentPerformanceData[]
  >([]);
  const [stories, setStories] = useState<Story[]>([]);
  const [selectedStoryId, setSelectedStoryId] = useState<number | null>(null);
  const [loadingStories, setLoadingStories] = useState(true);
  const [loadingPerformance, setLoadingPerformance] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [statusFilter, setStatusFilter] = useState<string>("active");
  const [refreshKey, setRefreshKey] = useState(0);

  // Reset-progress dialog state
  const [resetTarget, setResetTarget] = useState<{
    userId: string;
    name: string;
  } | null>(null);
  const [resetPhase, setResetPhase] = useState<ResetPhase>("all");
  const [resetting, setResetting] = useState(false);
  const [resetError, setResetError] = useState<string | null>(null);
  const [resetNotice, setResetNotice] = useState<string | null>(null);

  const closeResetDialog = useCallback(() => {
    if (resetting) return;
    setResetTarget(null);
    setResetPhase("all");
    setResetError(null);
  }, [resetting]);

  const handleResetConfirm = async () => {
    if (!resetTarget || !selectedStoryId) return;
    setResetting(true);
    setResetError(null);
    try {
      const response = await api.resetStudentProgress(
        selectedStoryId.toString(),
        resetTarget.userId,
        resetPhase,
      );
      if (!response.success || !response.data) {
        setResetError(response.error || "Reset failed");
        return;
      }
      const rows = Object.values(response.data.deleted).reduce(
        (sum, n) => sum + n,
        0,
      );
      const phaseLabel =
        RESET_PHASE_OPTIONS.find((o) => o.value === resetPhase)?.label ??
        resetPhase;
      setResetNotice(
        `Reset ${phaseLabel} for ${resetTarget.name} (${rows} rows removed).`,
      );
      setResetTarget(null);
      setResetPhase("all");
      setRefreshKey((k) => k + 1);
    } catch (err) {
      setResetError(err instanceof Error ? err.message : "Reset failed");
    } finally {
      setResetting(false);
    }
  };

  useEffect(() => {
    const fetchStories = async () => {
      if (!id) {
        setError("Course ID is required");
        setLoadingStories(false);
        return;
      }

      try {
        const response = await api.getCourseStories(id);
        if (response.success && response.data) {
          setStories(response.data);
        } else {
          setError(response.error || "Failed to fetch course stories");
        }
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
  }, [selectedStoryId, statusFilter, refreshKey]);

  // Sort performance data by:
  // 1. Most combined correct (vocab + grammar)
  // 2. Least combined incorrect (vocab + grammar)
  // 3. Alphabetically by email
  const sortedPerformanceData = performanceData.slice().sort((a, b) => {
    const totalCorrectA = a.vocab_correct + a.grammar_correct;
    const totalCorrectB = b.vocab_correct + b.grammar_correct;

    if (totalCorrectB !== totalCorrectA) {
      return totalCorrectB - totalCorrectA; // Higher correct first
    }

    const totalIncorrectA = a.vocab_incorrect + a.grammar_incorrect;
    const totalIncorrectB = b.vocab_incorrect + b.grammar_incorrect;

    if (totalIncorrectA !== totalIncorrectB) {
      return totalIncorrectA - totalIncorrectB; // Lower incorrect first
    }

    return a.email.localeCompare(b.email); // Alphabetical by email
  });

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
                      downloadCSV(sortedPerformanceData, titleStr);
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

          {resetNotice && (
            <p
              role="status"
              className="mb-4 rounded border border-green-300 bg-green-50 p-3 text-green-800"
            >
              {resetNotice}
            </p>
          )}

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
                    <th className="border border-gray-300 p-3 text-center">
                      Total Time
                    </th>
                    <th className="border border-gray-300 p-3 text-center">
                      Video Time
                    </th>
                    <th className="border border-gray-300 p-3 text-center">
                      Vocab Accuracy
                    </th>
                    <th className="border border-gray-300 p-3 text-center">
                      Vocab Time
                    </th>
                    <th className="border border-gray-300 p-3 text-center">
                      Grammar Accuracy
                    </th>
                    <th className="border border-gray-300 p-3 text-center">
                      Grammar Time
                    </th>
                    <th className="border border-gray-300 p-3 text-center">
                      Translation
                    </th>
                    <th className="border border-gray-300 p-3 text-center">
                      Translation Time
                    </th>
                    <th className="border border-gray-300 p-3 text-center">
                      Actions
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {sortedPerformanceData.map((student) => (
                    <tr key={student.user_id} className="hover:bg-gray-50">
                      <td className="border border-gray-300 p-3">
                        <div>
                          <div className="font-semibold">
                            {student.user_name}
                          </div>
                          <div className="text-sm text-gray-600">
                            {student.email}
                          </div>
                        </div>
                      </td>
                      <td className="border border-gray-300 p-3 text-center font-semibold">
                        {formatTime(student.total_time_seconds)}
                      </td>
                      <td className="border border-gray-300 p-3 text-center">
                        {formatTime(student.video_time_seconds)}
                      </td>
                      <td className="border border-gray-300 p-3 text-center">
                        <span
                          className={`font-semibold ${student.vocab_accuracy >= 80 ? "text-green-600" : student.vocab_accuracy >= 60 ? "text-yellow-600" : "text-red-600"}`}
                        >
                          {formatAccuracy(student.vocab_accuracy)}
                        </span>
                        <div className="text-xs text-gray-500">
                          {student.vocab_correct} correct /{" "}
                          {student.vocab_incorrect} incorrect
                        </div>
                      </td>
                      <td className="border border-gray-300 p-3 text-center">
                        {formatTime(student.vocab_time_seconds)}
                      </td>
                      <td className="border border-gray-300 p-3 text-center">
                        <span
                          className={`font-semibold ${student.grammar_accuracy >= 80 ? "text-green-600" : student.grammar_accuracy >= 60 ? "text-yellow-600" : "text-red-600"}`}
                        >
                          {formatAccuracy(student.grammar_accuracy)}
                        </span>
                        <div className="text-xs text-gray-500">
                          {student.grammar_correct} correct /{" "}
                          {student.grammar_incorrect} incorrect
                        </div>
                      </td>
                      <td className="border border-gray-300 p-3 text-center">
                        {formatTime(student.grammar_time_seconds)}
                      </td>
                      <td className="border border-gray-300 p-3 text-center">
                        <span
                          className={`${student.translation_completed ? "text-green-600" : "text-red-600"}`}
                        >
                          {student.translation_completed ? "✓" : "✗"}
                        </span>
                        {student.requested_lines &&
                          student.requested_lines.length > 0 && (
                            <div className="text-xs text-gray-500 mt-1">
                              Lines: {student.requested_lines.join(", ")}
                            </div>
                          )}
                      </td>
                      <td className="border border-gray-300 p-3 text-center">
                        {formatTime(student.translation_time_seconds)}
                      </td>
                      <td className="border border-gray-300 p-3 text-center">
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => {
                            setResetNotice(null);
                            setResetTarget({
                              userId: student.user_id,
                              name: student.user_name || student.email,
                            });
                          }}
                        >
                          Reset…
                        </Button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}

      <Modal
        isOpen={resetTarget !== null}
        onClose={closeResetDialog}
        title="Reset student progress"
        description={
          resetTarget
            ? `Delete ${resetTarget.name}'s answers, submissions and time for this story so they can redo it. This cannot be undone.`
            : undefined
        }
        closeDisabled={resetting}
      >
        <div className="mt-4">
          <label htmlFor="reset-phase" className="block font-semibold mb-2">
            What to reset
          </label>
          <select
            id="reset-phase"
            value={resetPhase}
            onChange={(e) => setResetPhase(e.target.value as ResetPhase)}
            disabled={resetting}
            className="border border-gray-300 rounded px-3 py-2 w-full"
          >
            {RESET_PHASE_OPTIONS.map((o) => (
              <option key={o.value} value={o.value}>
                {o.label}
              </option>
            ))}
          </select>
          {resetError && (
            <p role="alert" className="mt-3 text-red-600">
              {resetError}
            </p>
          )}
        </div>
        <div className="mt-6 flex gap-3 justify-end">
          <Button
            variant="outline"
            onClick={closeResetDialog}
            disabled={resetting}
          >
            Cancel
          </Button>
          <Button
            variant="danger"
            onClick={handleResetConfirm}
            disabled={resetting}
          >
            {resetting ? "Resetting..." : "Reset"}
          </Button>
        </div>
      </Modal>
    </div>
  );
}
