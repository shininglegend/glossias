import { useState, useEffect } from "react";
import { useParams } from "react-router";
import { useApiService } from "../services/api";

interface Course {
  course_id: number;
  course_name: string;
}

interface Story {
  story_id: number;
  week_number: number;
  day_letter: string;
  title: string;
}

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
  const [allData, setAllData] = useState<StudentPerformanceData[]>([]);
  const [stories, setStories] = useState<Story[]>([]);
  const [selectedStoryId, setSelectedStoryId] = useState<number | null>(null);
  const [filteredData, setFilteredData] = useState<StudentPerformanceData[]>(
    [],
  );
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchData = async () => {
      if (!id) {
        setError("Course ID is required");
        setLoading(false);
        return;
      }

      try {
        const response = await api.getCourseStudentPerformance(id);
        if (response.success && response.data) {
          setAllData(response.data);

          // Extract unique stories
          const storyMap = new Map<number, Story>();
          response.data.forEach((item: StudentPerformanceData) => {
            if (!storyMap.has(item.story_id)) {
              storyMap.set(item.story_id, {
                story_id: item.story_id,
                week_number: 0,
                day_letter: "",
                title: item.story_title,
              });
            }
          });
          const uniqueStories = Array.from(storyMap.values()).sort((a, b) =>
            a.title.localeCompare(b.title),
          );
          setStories(uniqueStories);

          if (uniqueStories.length > 0) {
            setSelectedStoryId(uniqueStories[0].story_id);
          }
        } else {
          setError(
            response.error || "Failed to fetch student performance data",
          );
        }
      } catch (err) {
        setError("Failed to fetch student performance data");
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, [id, api]);

  useEffect(() => {
    if (selectedStoryId !== null) {
      const filtered = allData.filter(
        (item) => item.story_id === selectedStoryId,
      );
      // Sort by performance: 50% vocab + 50% grammar, ties by name
      filtered.sort((a, b) => {
        const scoreA = (a.vocab_accuracy + a.grammar_accuracy) / 2;
        const scoreB = (b.vocab_accuracy + b.grammar_accuracy) / 2;
        if (scoreB !== scoreA) {
          return scoreB - scoreA; // Higher score first
        }
        return a.user_name.localeCompare(b.user_name); // Alphabetical tie-break
      });
      setFilteredData(filtered);
    } else {
      setFilteredData([]);
    }
  }, [selectedStoryId, allData]);

  if (loading) {
    return (
      <div className="container">
        <h1>Student Performance</h1>
        <p>Loading student performance data...</p>
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
        <p>No student performance data found for this course.</p>
      ) : (
        <div>
          <div className="mb-4">
            <label htmlFor="story-select" className="block font-semibold mb-2">
              Select Story:
            </label>
            <select
              id="story-select"
              value={selectedStoryId || ""}
              onChange={(e) => setSelectedStoryId(Number(e.target.value))}
              className="border border-gray-300 rounded px-3 py-2 w-full max-w-md"
            >
              {stories.map((story) => (
                <option key={story.story_id} value={story.story_id}>
                  {story.title}
                </option>
              ))}
            </select>
            <button
              onClick={() =>
                downloadCSV(
                  filteredData,
                  stories.find((s) => s.story_id === selectedStoryId)?.title ||
                    "story",
                )
              }
              disabled={filteredData.length === 0}
              className="mt-2 px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:bg-gray-400 disabled:cursor-not-allowed"
            >
              Download CSV
            </button>
          </div>

          {filteredData.length === 0 ? (
            <p>No performance data for the selected story.</p>
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
                  </tr>
                </thead>
                <tbody>
                  {filteredData.map((student) => (
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
                          {student.vocab_correct}/
                          {student.vocab_correct + student.vocab_incorrect}
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
                          {student.grammar_correct}/
                          {student.grammar_correct + student.grammar_incorrect}
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
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
