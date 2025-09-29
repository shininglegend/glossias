import { useState, useEffect } from "react";
import { useParams, Link } from "react-router";
import { useApiService } from "../services/api";
import confetti from "canvas-confetti";

interface ScoreData {
  story_title: string;
  total_time_seconds: number;
  vocab_accuracy: number;
  vocab_time_seconds: number;
  grammar_accuracy: number;
  grammar_time_seconds: number;
  translation_time_seconds: number;
  video_time_seconds: number;
}

interface MissingActivity {
  activity: string;
  display_name: string;
  route: string;
  reason: string; // "no_data" or "insufficient_time"
}

interface IncompleteResponse {
  complete: false;
  story_title: string;
  missing_activities: MissingActivity[];
  message: string;
}

function fireConfetti() {
  // Initial burst
  confetti({
    particleCount: 100,
    spread: 70,
    origin: { y: 0.6 },
  });

  // Side bursts
  setTimeout(() => {
    confetti({
      particleCount: 50,
      angle: 60,
      spread: 55,
      origin: { x: 0 },
    });
    confetti({
      particleCount: 50,
      angle: 120,
      spread: 55,
      origin: { x: 1 },
    });
  }, 300);

  // Final burst
  setTimeout(() => {
    confetti({
      particleCount: 80,
      spread: 100,
      origin: { y: 0.4 },
    });
  }, 600);
}

function formatTime(seconds: number): string {
  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = seconds % 60;
  return `${minutes}:${remainingSeconds.toString().padStart(2, "0")}`;
}

export function StoriesScore() {
  const { id } = useParams<{ id: string }>();
  const api = useApiService();
  const [scoreData, setScoreData] = useState<ScoreData | null>(null);
  const [incompleteData, setIncompleteData] =
    useState<IncompleteResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [confettiFired, setConfettiFired] = useState(false);

  useEffect(() => {
    const fetchScoreData = async () => {
      if (!id) {
        setError("Story ID is required");
        setLoading(false);
        return;
      }

      try {
        const response = await api.getStoryScore(id);
        if (response.success && response.data) {
          if ("complete" in response.data && response.data.complete === false) {
            setIncompleteData(response.data);
          } else {
            setScoreData(response.data as ScoreData);
          }
        } else {
          setError(response.error || "Failed to fetch score data");
        }
      } catch (err) {
        setError("Failed to fetch score data");
      } finally {
        setLoading(false);
      }
    };

    fetchScoreData();
  }, [id]);

  // Fire confetti when data loads
  useEffect(() => {
    if (scoreData && !confettiFired) {
      fireConfetti();
      setConfettiFired(true);
    }
  }, [scoreData, confettiFired]);

  if (loading) {
    return (
      <div className="container">
        <p>Loading your results...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="container">
        <p>Error: {error}</p>
        <Link to="/">Back to Stories</Link>
      </div>
    );
  }

  if (incompleteData) {
    return (
      <>
        <header>
          <h1>{incompleteData.story_title}</h1>
          <h2>Complete Your Activities</h2>

          <div className="bg-yellow-50 border border-yellow-300 p-6 mb-4 rounded-lg text-center">
            <div className="flex items-center justify-center mb-4">
              <span className="material-icons text-yellow-600 mr-2 text-2xl">
                warning
              </span>
              <div>
                <p className="text-yellow-700 text-lg font-medium">
                  {incompleteData.message}
                </p>
              </div>
            </div>
          </div>
        </header>

        <div className="max-w-4xl mx-auto px-5">
          <div className="space-y-4">
            {incompleteData.missing_activities.map((activity, index) => (
              <div
                key={index}
                className="bg-white border-2 border-orange-200 rounded-lg p-6"
              >
                <div className="flex items-center justify-between">
                  <div className="flex items-center">
                    <span className="material-icons text-orange-600 mr-3 text-2xl">
                      {activity.activity === "vocab"
                        ? "quiz"
                        : activity.activity === "grammar"
                          ? "school"
                          : "translate"}
                    </span>
                    <div>
                      <h3 className="text-lg font-bold text-gray-800">
                        {activity.display_name}
                      </h3>
                      <p className="text-gray-600">
                        {activity.reason === "no_data"
                          ? "Not started yet"
                          : "Needs more time spent"}
                      </p>
                    </div>
                  </div>
                  <Link
                    to={`/stories/${id}/${activity.route}`}
                    className="inline-flex items-center px-6 py-3 bg-orange-500 text-white rounded-lg hover:bg-orange-600 font-medium transition-all duration-200"
                  >
                    <span>
                      {activity.reason === "no_data" ? "Start" : "Continue"}
                    </span>
                    <span className="material-icons ml-2">arrow_forward</span>
                  </Link>
                </div>
              </div>
            ))}
          </div>

          <div className="text-center mt-8">
            <Link
              to="/"
              className="inline-flex items-center px-6 py-3 bg-gray-500 text-white rounded-lg hover:bg-gray-600 font-medium transition-all duration-200"
            >
              <span>Back to Stories</span>
              <span className="material-icons ml-2">home</span>
            </Link>
          </div>
        </div>
      </>
    );
  }

  if (!scoreData) {
    return (
      <div className="container">
        <p>No score data found</p>
        <Link to="/">Back to Stories</Link>
      </div>
    );
  }

  const overallScore = Math.round(
    (scoreData.vocab_accuracy + scoreData.grammar_accuracy) / 2,
  );

  return (
    <>
      <header>
        <h1>{scoreData.story_title}</h1>
        <h2>🎉 Congratulations! You've completed the story! 🎉</h2>

        <div className="bg-green-50 border border-green-300 p-6 mb-4 rounded-lg text-center">
          <div className="flex items-center justify-center mb-4">
            <span className="material-icons text-green-600 mr-2 text-4xl">
              emoji_events
            </span>
            <div>
              <h3 className="text-3xl font-bold text-green-700 mb-2">
                Overall Score: {overallScore}%
              </h3>
              <p className="text-green-600 text-lg">
                Total Time: {formatTime(scoreData.total_time_seconds)}
              </p>
            </div>
          </div>
        </div>

        <div className="text-center">
          <Link
            to="/"
            className="inline-flex items-center px-8 py-4 bg-blue-500 text-white rounded-lg hover:bg-blue-600 text-lg font-semibold transition-all duration-200 shadow-lg"
          >
            <span>Back to Stories</span>
            <span className="material-icons ml-2">home</span>
          </Link>
        </div>
      </header>

      <div className="max-w-4xl mx-auto px-5">
        {/* Detailed Scores */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-8">
          {/* Vocabulary Score */}
          <div className="bg-white border-2 border-blue-200 rounded-lg p-6">
            <div className="flex items-center mb-4">
              <span className="material-icons text-blue-600 mr-3 text-2xl">
                quiz
              </span>
              <h3 className="text-xl font-bold text-blue-900">Vocabulary</h3>
            </div>
            <div className="space-y-3">
              <div className="flex justify-between items-center">
                <span className="text-gray-600">Accuracy:</span>
                <span
                  className={`text-2xl font-bold ${
                    scoreData.vocab_accuracy >= 80
                      ? "text-green-600"
                      : scoreData.vocab_accuracy >= 60
                        ? "text-yellow-600"
                        : "text-red-600"
                  }`}
                >
                  {Math.round(scoreData.vocab_accuracy)}%
                </span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-gray-600">Time spent:</span>
                <span className="text-lg font-medium">
                  {formatTime(scoreData.vocab_time_seconds)}
                </span>
              </div>
              <div className="w-full bg-gray-200 rounded-full h-2">
                <div
                  className={`h-2 rounded-full ${
                    scoreData.vocab_accuracy >= 80
                      ? "bg-green-500"
                      : scoreData.vocab_accuracy >= 60
                        ? "bg-yellow-500"
                        : "bg-red-500"
                  }`}
                  style={{ width: `${scoreData.vocab_accuracy}%` }}
                />
              </div>
            </div>
          </div>

          {/* Grammar Score */}
          <div className="bg-white border-2 border-purple-200 rounded-lg p-6">
            <div className="flex items-center mb-4">
              <span className="material-icons text-purple-600 mr-3 text-2xl">
                school
              </span>
              <h3 className="text-xl font-bold text-purple-900">Grammar</h3>
            </div>
            <div className="space-y-3">
              <div className="flex justify-between items-center">
                <span className="text-gray-600">Accuracy:</span>
                <span
                  className={`text-2xl font-bold ${
                    scoreData.grammar_accuracy >= 80
                      ? "text-green-600"
                      : scoreData.grammar_accuracy >= 60
                        ? "text-yellow-600"
                        : "text-red-600"
                  }`}
                >
                  {Math.round(scoreData.grammar_accuracy)}%
                </span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-gray-600">Time spent:</span>
                <span className="text-lg font-medium">
                  {formatTime(scoreData.grammar_time_seconds)}
                </span>
              </div>
              <div className="w-full bg-gray-200 rounded-full h-2">
                <div
                  className={`h-2 rounded-full ${
                    scoreData.grammar_accuracy >= 80
                      ? "bg-green-500"
                      : scoreData.grammar_accuracy >= 60
                        ? "bg-yellow-500"
                        : "bg-red-500"
                  }`}
                  style={{ width: `${scoreData.grammar_accuracy}%` }}
                />
              </div>
            </div>
          </div>
        </div>

        {/* Time Breakdown */}
        <div className="bg-gray-50 border border-gray-300 rounded-lg p-6 mb-8">
          <div className="flex items-center mb-4">
            <span className="material-icons text-gray-600 mr-3 text-2xl">
              schedule
            </span>
            <h3 className="text-xl font-bold text-gray-800">Time Breakdown</h3>
          </div>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div className="text-center">
              <div className="text-2xl font-bold text-red-600">
                {formatTime(scoreData.video_time_seconds)}
              </div>
              <div className="text-sm text-gray-600">Video</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold text-blue-600">
                {formatTime(scoreData.vocab_time_seconds)}
              </div>
              <div className="text-sm text-gray-600">Vocabulary</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold text-yellow-600">
                {formatTime(scoreData.translation_time_seconds)}
              </div>
              <div className="text-sm text-gray-600">Translation</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold text-purple-600">
                {formatTime(scoreData.grammar_time_seconds)}
              </div>
              <div className="text-sm text-gray-600">Grammar</div>
            </div>
          </div>
        </div>

        {/* Encouragement Message */}
        <div className="text-center bg-gradient-to-r from-blue-50 to-purple-50 border border-blue-200 rounded-lg p-8">
          <h3 className="text-2xl font-bold text-gray-800 mb-3">
            {overallScore >= 90
              ? "Outstanding work! 🌟"
              : overallScore >= 80
                ? "Great job! 👏"
                : overallScore >= 70
                  ? "Good effort! 💪"
                  : "Keep practicing! 📚"}
          </h3>
          <p className="text-gray-600 text-lg">
            {overallScore >= 90
              ? "You've mastered this story! Ready for the next challenge?"
              : overallScore >= 80
                ? "You're doing really well! Keep up the great work."
                : overallScore >= 70
                  ? "You're making good progress. Try another story to improve!"
                  : "Every step counts! Practice makes perfect."}
          </p>
        </div>
      </div>
    </>
  );
}
