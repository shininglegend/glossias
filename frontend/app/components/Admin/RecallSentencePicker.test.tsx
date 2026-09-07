import { describe, it, expect } from "vitest";
import { recallCoveredLineNumbers } from "./RecallSentencePicker";

const lines = [
  { lineNumber: 1, text: "שורה אחת" },
  { lineNumber: 2, text: "שורה שתיים" },
];

describe("recallCoveredLineNumbers", () => {
  it("returns the line that contains the sentence", () => {
    expect(recallCoveredLineNumbers("שורה שתיים", lines)).toEqual([2]);
  });

  it("returns every line that makes up a joined sentence", () => {
    expect(recallCoveredLineNumbers("שורה אחת שורה שתיים", lines)).toEqual([
      1, 2,
    ]);
  });

  it("returns nothing when the sentence is not in the story", () => {
    expect(recallCoveredLineNumbers("אין", lines)).toEqual([]);
  });
});
