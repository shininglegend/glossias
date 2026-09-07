import { describe, it, expect } from "vitest";
import {
  adoptRangeText,
  clickLineRange,
  wouldOverrideEdit,
  hebrewForRange,
  englishForRange,
} from "./produceRange";

const lines = [
  { lineNumber: 4, text: "ד" },
  { lineNumber: 5, text: "ה" },
  { lineNumber: 6, text: "ו" },
];

describe("hebrewForRange", () => {
  it("joins inclusive story lines", () => {
    expect(hebrewForRange(lines, 4, 6)).toBe("ד\nה\nו");
  });

  it("returns a single line when start equals end", () => {
    expect(hebrewForRange(lines, 5, 5)).toBe("ה");
  });
});

describe("adoptRangeText", () => {
  it("replaces empty or still-generated text", () => {
    expect(adoptRangeText("", "ד\nה", "ד\nה\nו", false)).toBe("ד\nה\nו");
    expect(adoptRangeText("ד\nה", "ד\nה", "ד\nה\nו", false)).toBe("ד\nה\nו");
  });

  it("keeps a manual edit across a range change", () => {
    expect(adoptRangeText("trimmed", "ד\nה", "ד\nה\nו", false)).toBe("trimmed");
  });

  it("overwrites when forced", () => {
    expect(adoptRangeText("trimmed", "ד\nה", "ד\nה\nו", true)).toBe("ד\nה\nו");
  });
});

describe("clickLineRange", () => {
  it("starts a single-line range", () => {
    expect(clickLineRange("", "", 4)).toEqual({ start: 4, end: 4 });
  });

  it("expands a single line to the clicked end", () => {
    expect(clickLineRange(4, 4, 6)).toEqual({ start: 4, end: 6 });
    expect(clickLineRange(6, 6, 4)).toEqual({ start: 4, end: 6 });
  });

  it("restarts after a completed range", () => {
    expect(clickLineRange(4, 6, 2)).toEqual({ start: 2, end: 2 });
  });
});

describe("wouldOverrideEdit", () => {
  it("is false for empty or still-generated text", () => {
    expect(wouldOverrideEdit("", "ד", "ד\nה")).toBe(false);
    expect(wouldOverrideEdit("ד", "ד", "ד\nה")).toBe(false);
  });

  it("is true when an edit would be replaced", () => {
    expect(wouldOverrideEdit("trimmed", "ד", "ד\nה")).toBe(true);
  });

  it("is false when the new range already matches the edit", () => {
    expect(wouldOverrideEdit("trimmed", "ד", "trimmed")).toBe(false);
  });
});

describe("englishForRange", () => {
  it("joins stored translations and skips missing lines", () => {
    const translations = new Map<number, string>([
      [4, "four"],
      [6, "six"],
    ]);
    expect(englishForRange(translations, 4, 6)).toBe("four\nsix");
  });
});
