import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { AttemptPicker } from "./AttemptPicker";

describe("AttemptPicker", () => {
  it("renders nothing for a single attempt", () => {
    const { container } = render(
      <AttemptPicker
        attempts={[{ number: 1 }]}
        selected={1}
        onSelect={vi.fn()}
      />,
    );
    expect(container).toBeEmptyDOMElement();
  });

  it("marks the selected attempt and reports clicks", () => {
    const onSelect = vi.fn();
    render(
      <AttemptPicker
        attempts={[{ number: 1 }, { number: 2, suffix: "(grading)" }]}
        selected={2}
        onSelect={onSelect}
      />,
    );
    const first = screen.getByRole("button", { name: "1" });
    const second = screen.getByRole("button", { name: "2 (grading)" });
    expect(first).toHaveAttribute("aria-pressed", "false");
    expect(second).toHaveAttribute("aria-pressed", "true");

    fireEvent.click(first);
    expect(onSelect).toHaveBeenCalledWith(1);
  });
});
