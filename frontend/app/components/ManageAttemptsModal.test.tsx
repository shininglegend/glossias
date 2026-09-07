import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import {
  ManageAttemptsModal,
  attemptBreakdown,
  formatSeconds,
} from "./ManageAttemptsModal";
import type { StudentAttemptSummary } from "../types/api";

const getStudentAttempts = vi.fn();
const deleteStudentAttempt = vi.fn();
const resetStudentProgress = vi.fn();
// Stable object, like the memoized service in production; a fresh object per
// render would re-run the modal's load effect on every state change.
const apiMock = {
  getStudentAttempts,
  deleteStudentAttempt,
  resetStudentProgress,
};
vi.mock("../services/api", () => ({ useApiService: () => apiMock }));

const archived = (number: number, accuracy: number): StudentAttemptSummary => ({
  number,
  is_current: false,
  completed_at: "2026-09-07T17:26:11Z",
  score: {
    overall_accuracy: accuracy,
    total_time_seconds: 3725,
    produce_segments_submitted: 1,
    produce_segments_graded: 1,
    identify_accuracy: 80,
    identify_correct_count: 4,
    identify_incorrect_count: 1,
    identify_total: 5,
    identify_time_seconds: 90,
    video_time_seconds: 120,
    produce_score: 70,
    produce_total: 2,
    produce_time_seconds: 200,
    recall_accuracy: 100,
    recall_correct_count: 3,
    recall_incorrect_count: 0,
    recall_attempts: 1,
    recall_time_seconds: 45,
    translation_completed: true,
    translation_time_seconds: 60,
    requested_lines: [3, 10],
    recall_total: 3,
    produce_segments: [{ segment_order: 1, ai_score: 70 }],
  },
});

const live: StudentAttemptSummary = {
  number: 3,
  is_current: true,
  score: {
    overall_accuracy: 0,
    total_time_seconds: 180,
    produce_segments_submitted: 0,
    produce_segments_graded: 0,
  },
  stages: [
    { phase: "video", detail: "watched", seconds: 120 },
    { phase: "identify", detail: "3 correct / 1 incorrect", seconds: 60 },
  ],
};

function renderModal(onChanged = vi.fn()) {
  render(
    <ManageAttemptsModal
      storyId="2"
      student={{ userId: "user_1", name: "Titus" }}
      onClose={vi.fn()}
      onChanged={onChanged}
    />,
  );
  return onChanged;
}

describe("ManageAttemptsModal", () => {
  beforeEach(() => {
    getStudentAttempts.mockReset();
    deleteStudentAttempt.mockReset();
    resetStudentProgress.mockReset();
  });

  it("formats seconds as m:ss and h:mm:ss", () => {
    expect(formatSeconds(65)).toBe("1:05");
    expect(formatSeconds(3725)).toBe("1:02:05");
  });

  it("builds a score/time breakdown from a frozen snapshot", () => {
    const rows = attemptBreakdown(archived(1, 63.1));
    expect(rows.map((r) => r.phase)).toEqual([
      "video",
      "identify",
      "translate",
      "produce",
      "recall",
    ]);
    expect(rows[1]).toMatchObject({
      headline: "80.0%",
      stats: ["4 correct", "1 incorrect", "of 5"],
      seconds: 90,
    });
    expect(rows[2]).toMatchObject({
      headline: "Completed",
      stats: ["2 lines requested", "3, 10"],
    });
    expect(rows[3].extra).toEqual(["Passage 1: 70"]);
  });

  it("lists archived attempts with scores and the live attempt by stage", async () => {
    getStudentAttempts.mockResolvedValue({
      success: true,
      data: [archived(1, 63.1), archived(2, 84.7), live],
    });
    renderModal();

    expect(await screen.findByText("63.1%")).toBeInTheDocument();
    expect(screen.getByText("84.7%")).toBeInTheDocument();
    expect(screen.getByText("Official")).toBeInTheDocument();
    expect(screen.getByText("In progress")).toBeInTheDocument();
    expect(
      screen.getByLabelText("Attempt 3, In progress").closest("details"),
    ).toHaveAttribute("open");
    expect(screen.getByText("3 correct / 1 incorrect")).toBeInTheDocument();
    // Archived rows delete whole; the live row deletes by stage only.
    expect(
      screen.getAllByRole("button", { name: /^Delete attempt/ }),
    ).toHaveLength(2);
    expect(
      screen.getByRole("button", {
        name: "Delete Identify from the in-progress attempt",
      }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Delete attempt 3" }),
    ).toBeNull();

    const archivedSummary = screen.getByLabelText("Attempt 1, Completed");
    const archivedDetails = archivedSummary.closest("details");
    expect(archivedDetails).not.toHaveAttribute("open");
    fireEvent.click(archivedSummary);
    expect(archivedDetails).toHaveAttribute("open");
    expect(archivedDetails).toHaveTextContent("80.0%");
    expect(archivedDetails).toHaveTextContent("4 correct");
    expect(archivedDetails).toHaveTextContent("of 5");
    expect(archivedDetails).toHaveTextContent("Passage 1: 70");
    expect(archivedDetails).toHaveTextContent("1:30");
  });

  it("shows an untouched live attempt as not started with nothing to delete", async () => {
    getStudentAttempts.mockResolvedValue({
      success: true,
      data: [{ number: 1, is_current: true, stages: [] }],
    });
    renderModal();
    expect(await screen.findByText("Not started")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Delete/ })).toBeNull();
  });

  it("confirms before deleting an archived attempt, then reloads and notifies", async () => {
    getStudentAttempts
      .mockResolvedValueOnce({
        success: true,
        data: [archived(1, 63.1), archived(2, 84.7), live],
      })
      .mockResolvedValueOnce({
        success: true,
        data: [archived(1, 84.7), { ...live, number: 2 }],
      });
    deleteStudentAttempt.mockResolvedValue({
      success: true,
      data: { attempt: 1 },
    });
    const onChanged = renderModal();

    fireEvent.click(
      await screen.findByRole("button", { name: "Delete attempt 1" }),
    );
    expect(deleteStudentAttempt).not.toHaveBeenCalled();
    expect(
      screen.getByText(/Attempt 2 becomes the official score/),
    ).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Delete" }));
    await waitFor(() =>
      expect(deleteStudentAttempt).toHaveBeenCalledWith("2", "user_1", 1),
    );
    await waitFor(() => expect(screen.queryByText("63.1%")).toBeNull());
    expect(onChanged).toHaveBeenCalledTimes(1);
    expect(getStudentAttempts).toHaveBeenCalledTimes(2);
  });

  it("deletes a live stage through the phase reset endpoint", async () => {
    getStudentAttempts.mockResolvedValue({ success: true, data: [live] });
    resetStudentProgress.mockResolvedValue({
      success: true,
      data: { phase: "identify", deleted: {} },
    });
    renderModal();

    fireEvent.click(
      await screen.findByRole("button", {
        name: "Delete Identify from the in-progress attempt",
      }),
    );
    fireEvent.click(screen.getByRole("button", { name: "Delete" }));
    await waitFor(() =>
      expect(resetStudentProgress).toHaveBeenCalledWith(
        "2",
        "user_1",
        "identify",
      ),
    );
    expect(deleteStudentAttempt).not.toHaveBeenCalled();
  });

  it("surfaces a failed delete without closing the confirmation", async () => {
    getStudentAttempts.mockResolvedValue({
      success: true,
      data: [archived(1, 63.1), live],
    });
    deleteStudentAttempt.mockResolvedValue({
      success: false,
      error: "Attempt not found",
    });
    const onChanged = renderModal();

    fireEvent.click(
      await screen.findByRole("button", { name: "Delete attempt 1" }),
    );
    fireEvent.click(screen.getByRole("button", { name: "Delete" }));
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Attempt not found",
    );
    expect(onChanged).not.toHaveBeenCalled();
  });
});
