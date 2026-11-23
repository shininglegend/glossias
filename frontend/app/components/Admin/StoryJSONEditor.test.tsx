import React from "react";
import { describe, it, expect, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import StoryJSONEditor from "./StoryJSONEditor";
import type { Story } from "../../types/admin";

const story = {
  metadata: { storyId: 1 },
  content: { lines: [] },
} as unknown as Story;

describe("StoryJSONEditor", () => {
  it("shows the error when the save is refused", async () => {
    const onSubmit = vi
      .fn()
      .mockRejectedValue(new Error("HTTP 409: story already has lines"));
    render(<StoryJSONEditor value={story} onSubmit={onSubmit} />);

    fireEvent.click(screen.getByRole("button", { name: "Save" }));

    expect(
      await screen.findByText("HTTP 409: story already has lines"),
    ).toBeInTheDocument();
    expect(onSubmit).toHaveBeenCalledOnce();
  });

  it("does not submit invalid JSON", () => {
    const onSubmit = vi.fn();
    render(<StoryJSONEditor value={story} onSubmit={onSubmit} />);

    fireEvent.change(screen.getByRole("textbox"), { target: { value: "{" } });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));

    expect(onSubmit).not.toHaveBeenCalled();
  });
});
