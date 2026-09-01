import { useState, useEffect, type ReactNode } from "react";
import { useParams, useNavigate } from "react-router";
import { useApiService } from "../services/api";
import { useNavigationGuidance } from "../hooks/useNavigationGuidance";
import confetti from "canvas-confetti";

interface ScoreData {
  story_title: string;
  total_time_seconds: number;
  overall_accuracy: number;

  identify_accuracy: number;
  identify_correct_count: number;
  identify_incorrect_count: number;
  identify_total: number;

  produce_score: number;
  produce_segments_submitted: number;
  produce_segments_graded: number;
  produce_total: number;

  recall_accuracy: number;
  recall_correct_count: number;
  recall_incorrect_count: number;
  recall_attempts: number;
  recall_total: number;

  // Legacy phases (stories authored before the five-phase flow).
  vocab_accuracy: number;
  vocab_correct_count: number;
  vocab_incorrect_count: number;
  grammar_accuracy: number;
  grammar_correct_count: number;
  grammar_incorrect_count: number;

  video_time_seconds: number;
  identify_time_seconds: number;
  translation_time_seconds: number;
  produce_time_seconds: number;
  recall_time_seconds: number;
  vocab_time_seconds: number;
  grammar_time_seconds: number;
}

interface MissingActivity {
  activity: string;
  display_name: string;
  route: string;
  reason: string; // "no_data" (never started) or "incomplete" (started, not finished)
}

interface IncompleteResponse {
  complete: false;
  story_title: string;
  missing_activities: MissingActivity[];
  message: string;
}

const ACTIVITY_ICONS: Record<string, string> = {
  identify: "image_search",
  translation: "translate",
  produce: "edit_note",
  recall: "low_priority",
  vocab: "quiz",
  grammar: "school",
};

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

function scoreTextClass(score: number): string {
  if (score >= 80) return "text-green-600";
  if (score >= 60) return "text-secondary-500";
  return "text-red-600";
}

function scoreBarClass(score: number): string {
  if (score >= 80) return "bg-green-500";
  if (score >= 60) return "bg-secondary-500";
  return "bg-red-500";
}

interface PhaseCardProps {
  title: string;
  icon: string;
  colour: "primary" | "purple" | "teal" | "orange" | "gray";
  /** 0–100, or null when there is no score to show yet (e.g. grading pending). */
  score: number | null;
  scoreLabel?: string;
  timeSeconds: number;
  children?: ReactNode;
}

const COLOURS = {
  primary: {
    border: "border-primary-200",
    icon: "text-primary-600",
    title: "text-primary-900",
  },
  purple: {
    border: "border-purple-200",
    icon: "text-purple-600",
    title: "text-purple-900",
  },
  teal: {
    border: "border-teal-200",
    icon: "text-teal-600",
    title: "text-teal-900",
  },
  orange: {
    border: "border-orange-200",
    icon: "text-orange-600",
    title: "text-orange-900",
  },
  gray: {
    border: "border-gray-300",
    icon: "text-gray-600",
    title: "text-gray-800",
  },
} as const;

function PhaseCard({
  title,
  icon,
  colour,
  score,
  scoreLabel = "Accuracy:",
  timeSeconds,
  children,
}: PhaseCardProps) {
  const c = COLOURS[colour];
  return (
    <div className={`bg-white border-2 ${c.border} rounded-lg p-6`}>
      <div className="flex items-center mb-4">
        <span className={`material-icons ${c.icon} mr-3 text-2xl`}>{icon}</span>
        <h3 className={`text-xl font-bold ${c.title}`}>{title}</h3>
      </div>
      <div className="space-y-3">
        <div className="flex justify-between items-center">
          <span className="text-gray-600">{scoreLabel}</span>
          {score === null ? (
            <span className="text-lg font-semibold text-gray-500">
              Grading pending
            </span>
          ) : (
            <span className={`text-2xl font-bold ${scoreTextClass(score)}`}>
              {Math.round(score)}%
            </span>
          )}
        </div>
        {children}
        <div className="flex justify-between items-center">
          <span className="text-gray-600">Time spent:</span>
          <span className="text-lg font-medium">{formatTime(timeSeconds)}</span>
        </div>
        {score !== null && (
          <div className="w-full bg-gray-200 rounded-full h-2">
            <div
              className={`h-2 rounded-full ${scoreBarClass(score)}`}
              style={{ width: `${Math.min(100, Math.max(0, score))}%` }}
            />
          </div>
        )}
      </div>
    </div>
  );
}

function Attempts({ correct, wrong }: { correct: number; wrong: number }) {
  return (
    <div className="flex justify-between items-center">
      <span className="text-gray-600">Attempts:</span>
      <span className="text-sm font-medium">
        <span className="text-green-600">{correct} correct</span>
        <span className="text-gray-400"> • </span>
        <span className="text-red-600">{wrong} wrong</span>
      </span>
    </div>
  );
}

function TimeCell({
  label,
  seconds,
  colourClass,
}: {
  label: string;
  seconds: number;
  colourClass: string;
}) {
  return (
    <div className="text-center">
      <div className={`text-2xl font-bold ${colourClass}`}>
        {formatTime(seconds)}
      </div>
      <div className="text-sm text-gray-600">{label}</div>
    </div>
  );
}

export function StoriesScore() {
  const { id } = useParams<{ id: string }>();
  const api = useApiService();
  const navigate = useNavigate();
  const { getNavigationGuidance } = useNavigationGuidance();
  const [scoreData, setScoreData] = useState<ScoreData | null>(null);
  const [incompleteData, setIncompleteData] =
    useState<IncompleteResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [confettiFired, setConfettiFired] = useState(false);
  const [, setNextStepName] = useState<string>("Back to Stories");

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
          const data = response.data as Record<string, unknown>;
          if ("complete" in data && data.complete === false) {
            setIncompleteData(data as unknown as IncompleteResponse);
          } else {
            setScoreData(data as unknown as ScoreData);
          }
        } else {
          setError(response.error || "Failed to fetch score data");
        }
      } catch {
        setError("Failed to fetch score data");
      } finally {
        setLoading(false);
      }
    };

    fetchScoreData();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id]);

  useEffect(() => {
    const fetchNextStep = async () => {
      if (!id) return;
      try {
        const guidance = await getNavigationGuidance(id, "score");
        if (guidance) {
          setNextStepName(guidance.displayName);
        }
      } catch (error) {
        console.error("Failed to get navigation guidance:", error);
      }
    };

    fetchNextStep();
  }, [id, getNavigationGuidance]);

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
        <button onClick={() => navigate("/")}>Back to Stories</button>
      </div>
    );
  }

  if (incompleteData) {
    return (
      <>
        <header>
          <h1>{incompleteData.story_title}</h1>
          <h2>Complete Your Activities</h2>

          <div className="bg-secondary-50 border border-secondary-300 p-6 mb-4 rounded-lg text-center">
            <div className="flex items-center justify-center mb-4">
              <span className="material-icons text-secondary-600 mr-2 text-2xl">
                warning
              </span>
              <div>
                <p className="text-secondary-700 text-lg font-medium">
                  {incompleteData.message}
                </p>
              </div>
            </div>
          </div>
        </header>

        <div className="max-w-4xl mx-auto px-5">
          <div className="space-y-4">
            {incompleteData.missing_activities.map((activity) => {
              const notStarted = activity.reason === "no_data";
              return (
                <div
                  key={activity.activity}
                  className="bg-white border-2 border-orange-200 rounded-lg p-6"
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center">
                      <span className="material-icons text-orange-600 mr-3 text-2xl">
                        {ACTIVITY_ICONS[activity.activity] ?? "assignment"}
                      </span>
                      <div>
                        <h3 className="text-lg font-bold text-gray-800">
                          {activity.display_name}
                        </h3>
                        <p className="text-gray-600">
                          {notStarted
                            ? "Not started yet"
                            : "Started but not finished"}
                        </p>
                      </div>
                    </div>
                    <button
                      onClick={() =>
                        navigate(`/stories/${id}/${activity.route}`)
                      }
                      className="inline-flex items-center px-6 py-3 bg-orange-500 text-white rounded-lg hover:bg-orange-600 font-medium transition-all duration-200"
                    >
                      <span>{notStarted ? "Start" : "Continue"}</span>
                      <span className="material-icons ml-2">arrow_forward</span>
                    </button>
                  </div>
                </div>
              );
            })}
          </div>

          <div className="text-center mt-8">
            <button
              onClick={() => navigate("/")}
              className="inline-flex items-center px-6 py-3 bg-gray-500 text-white rounded-lg hover:bg-gray-600 font-medium transition-all duration-200"
            >
              <span>Back to Stories</span>
              <span className="material-icons ml-2">home</span>
            </button>
          </div>
        </div>
      </>
    );
  }

  if (!scoreData) {
    return (
      <div className="container">
        <p>No score data found</p>
        <button onClick={() => navigate("/")}>Back to Stories</button>
      </div>
    );
  }

  const overallScore = Math.round(scoreData.overall_accuracy);
  const hasIdentify = scoreData.identify_total > 0;
  const hasProduce = scoreData.produce_total > 0;
  const hasRecall = scoreData.recall_total > 0;
  const hasLegacyVocab =
    scoreData.vocab_correct_count + scoreData.vocab_incorrect_count > 0;
  const hasLegacyGrammar =
    scoreData.grammar_correct_count + scoreData.grammar_incorrect_count > 0;
  const producePending = hasProduce && scoreData.produce_segments_graded === 0;

  return (
    <>
      <header>
        <h1>{scoreData.story_title}</h1>
        <h2>🎉 Congratulations! You've completed the story! 🎉</h2>

        <div className="bg-green-50 border border-green-300 p-6 mb-4 rounded-lg text-center">
          <div className="flex items-center justify-center mb-4">
            <div>
              <h3 className="text-3xl font-bold text-green-700 mb-2">
                Overall Score: {overallScore}%
              </h3>
              <p className="text-green-600 text-lg">
                Total Time: {formatTime(scoreData.total_time_seconds)}
              </p>
              {producePending && (
                <p className="text-green-700 text-sm mt-2">
                  Your Produce writing is still being graded — check back for
                  your final score.
                </p>
              )}
            </div>
          </div>
        </div>

        <div className="text-center">
          <button
            onClick={() => navigate("/")}
            className="inline-flex items-center px-8 py-4 bg-primary-500 text-white rounded-lg hover:bg-primary-600 text-lg font-semibold transition-all duration-200 shadow-lg"
          >
            <span>Back to Stories</span>
            <span className="material-icons ml-2">home</span>
          </button>
        </div>
      </header>

      <div className="max-w-4xl mx-auto px-5">
        {/* Detailed Scores */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-8">
          {hasIdentify && (
            <PhaseCard
              title="Identify"
              icon={ACTIVITY_ICONS.identify}
              colour="primary"
              score={scoreData.identify_accuracy}
              timeSeconds={scoreData.identify_time_seconds}
            >
              <Attempts
                correct={scoreData.identify_correct_count}
                wrong={scoreData.identify_incorrect_count}
              />
              {scoreData.identify_incorrect_count > 0 && (
                <div className="text-xs text-gray-500 mt-1">
                  You picked the wrong picture{" "}
                  {scoreData.identify_incorrect_count} time
                  {scoreData.identify_incorrect_count === 1
                    ? ""
                    : "s"} across {scoreData.identify_total} target words.
                </div>
              )}
            </PhaseCard>
          )}

          {hasProduce && (
            <PhaseCard
              title="Produce"
              icon={ACTIVITY_ICONS.produce}
              colour="teal"
              score={producePending ? null : scoreData.produce_score}
              scoreLabel="AI score:"
              timeSeconds={scoreData.produce_time_seconds}
            >
              <div className="flex justify-between items-center">
                <span className="text-gray-600">Segments:</span>
                <span className="text-sm font-medium">
                  {scoreData.produce_segments_submitted} of{" "}
                  {scoreData.produce_total} submitted
                  <span className="text-gray-400"> • </span>
                  {scoreData.produce_segments_graded} graded
                </span>
              </div>
              {!producePending &&
                scoreData.produce_segments_graded <
                  scoreData.produce_segments_submitted && (
                  <div className="text-xs text-gray-500 mt-1">
                    Score so far covers the graded segments only.
                  </div>
                )}
            </PhaseCard>
          )}

          {hasRecall && (
            <PhaseCard
              title="Recall"
              icon={ACTIVITY_ICONS.recall}
              colour="orange"
              score={scoreData.recall_accuracy}
              timeSeconds={scoreData.recall_time_seconds}
            >
              <div className="flex justify-between items-center">
                <span className="text-gray-600">Attempts:</span>
                <span className="text-sm font-medium">
                  {scoreData.recall_attempts} ordering
                  {scoreData.recall_attempts === 1 ? "" : "s"} submitted
                </span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-gray-600">Sentences placed:</span>
                <span className="text-sm font-medium">
                  <span className="text-green-600">
                    {scoreData.recall_correct_count} correct
                  </span>
                  <span className="text-gray-400"> • </span>
                  <span className="text-red-600">
                    {scoreData.recall_incorrect_count} wrong
                  </span>
                </span>
              </div>
            </PhaseCard>
          )}

          {hasLegacyVocab && (
            <PhaseCard
              title="Vocabulary"
              icon={ACTIVITY_ICONS.vocab}
              colour="gray"
              score={scoreData.vocab_accuracy}
              timeSeconds={scoreData.vocab_time_seconds}
            >
              <Attempts
                correct={scoreData.vocab_correct_count}
                wrong={scoreData.vocab_incorrect_count}
              />
            </PhaseCard>
          )}

          {hasLegacyGrammar && (
            <PhaseCard
              title="Grammar"
              icon={ACTIVITY_ICONS.grammar}
              colour="purple"
              score={scoreData.grammar_accuracy}
              timeSeconds={scoreData.grammar_time_seconds}
            >
              <Attempts
                correct={scoreData.grammar_correct_count}
                wrong={scoreData.grammar_incorrect_count}
              />
            </PhaseCard>
          )}
        </div>

        {/* Time Breakdown */}
        <div className="bg-gray-50 border border-gray-300 rounded-lg p-6 mb-8">
          <div className="flex items-center mb-4">
            <span className="material-icons text-gray-600 mr-3 text-2xl">
              schedule
            </span>
            <h3 className="text-xl font-bold text-gray-800">Time Breakdown</h3>
          </div>
          <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
            <TimeCell
              label="Watch"
              seconds={scoreData.video_time_seconds}
              colourClass="text-red-600"
            />
            <TimeCell
              label="Identify"
              seconds={scoreData.identify_time_seconds}
              colourClass="text-primary-600"
            />
            <TimeCell
              label="Translate"
              seconds={scoreData.translation_time_seconds}
              colourClass="text-secondary-500"
            />
            <TimeCell
              label="Produce"
              seconds={scoreData.produce_time_seconds}
              colourClass="text-teal-600"
            />
            <TimeCell
              label="Recall"
              seconds={scoreData.recall_time_seconds}
              colourClass="text-orange-600"
            />
            {scoreData.vocab_time_seconds > 0 && (
              <TimeCell
                label="Vocabulary"
                seconds={scoreData.vocab_time_seconds}
                colourClass="text-gray-600"
              />
            )}
            {scoreData.grammar_time_seconds > 0 && (
              <TimeCell
                label="Grammar"
                seconds={scoreData.grammar_time_seconds}
                colourClass="text-purple-600"
              />
            )}
          </div>
        </div>

        {/* Encouragement Message */}
        <div className="text-center bg-gradient-to-r from-primary-50 to-purple-50 border border-primary-200 rounded-lg p-8">
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
