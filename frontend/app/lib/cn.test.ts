import { describe, it, expect } from "vitest";
import { cn } from "./cn";

describe("cn classname utility", () => {
  it("joins simple class strings", () => {
    expect(cn("class1", "class2")).toBe("class1 class2");
  });

  it("filters out falsy values", () => {
    expect(cn("class1", false, "class2", null, undefined)).toBe(
      "class1 class2",
    );
  });

  it("returns an empty string if all inputs are falsy", () => {
    expect(cn(false, null, undefined)).toBe("");
  });

  it("handles a single class name", () => {
    expect(cn("class-only")).toBe("class-only");
  });
});
