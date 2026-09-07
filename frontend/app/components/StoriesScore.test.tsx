import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import { StoriesScore } from "./StoriesScore";

vi.mock("canvas-confetti", () => ({ default: vi.fn() }));

const getStoryScore = vi.fn();
vi.mock("../services/api", () => ({
  useApiService: () => ({ getStoryScore }),
}));

const baseScore = {
  story_title: "A Story",
  total_time_seconds: 90,
  overall_accuracy: 80,
  identify_accuracy: 80,
  identify_correct_count: 4,
  identify_incorrect_count: 1,
  identify_total: 4,
  produce_score: 85,
  produce_segments_submitted: 2,
  produce_segments_graded: 2,
  produce_total: 2,
  produce_segments: [],
  recall_accuracy: 75,
  recall_correct_count: 3,
  recall_incorrect_count: 1,
  recall_attempts: 1,
  recall_total: 3,
  video_time_seconds: 10,
  identify_time_seconds: 20,
  translation_time_seconds: 20,
  produce_time_seconds: 20,
  recall_time_seconds: 20,
};

function renderScore() {
  return render(
    <MemoryRouter initialEntries={["/stories/7/score"]}>
      <Routes>
        <Route path="/stories/:id/score" element={<StoriesScore />} />
        <Route path="/stories/:id/identify" element={<p>Identify page</p>} />
      </Routes>
    </MemoryRouter>,
  );
}

describe("StoriesScore attempts", () => {
  beforeEach(() => {
    getStoryScore.mockReset();
  });

  it("switches attempts through the picker and offers a redo for archived ones", async () => {
    getStoryScore.mockImplementation(async (_id: string, attempt?: number) => ({
      success: true,
      data: {
        ...baseScore,
        overall_accuracy: attempt === 1 ? 60 : 80,
        attempt_number: attempt ?? 2,
        attempts: [{ number: 1 }, { number: 2 }],
        archived: true,
      },
    }));
    renderScore();

    await screen.findByText("Overall Score: 80%");
    expect(getStoryScore).toHaveBeenLastCalledWith("7", undefined);
    expect(
      screen.getByRole("button", { name: /Do this story again/ }),
    ).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "1" }));
    await screen.findByText("Overall Score: 60%");
    expect(getStoryScore).toHaveBeenLastCalledWith("7", 1);

    fireEvent.click(
      screen.getByRole("button", { name: /Do this story again/ }),
    );
    await waitFor(() =>
      expect(screen.getByText("Identify page")).toBeInTheDocument(),
    );
  });

  it("holds the redo and shows the grading note while the attempt is still live", async () => {
    getStoryScore.mockResolvedValue({
      success: true,
      data: {
        ...baseScore,
        produce_segments_graded: 0,
        attempt_number: 1,
        attempts: [{ number: 1, pending: true }],
        archived: false,
      },
    });
    renderScore();

    await screen.findByText("Overall Score: 80%");
    expect(screen.getByText(/still being graded/)).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /Do this story again/ }),
    ).not.toBeInTheDocument();
  });
});
