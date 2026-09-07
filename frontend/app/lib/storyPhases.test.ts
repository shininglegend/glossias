import { describe, expect, it } from "vitest";
import { STORY_BAR_PHASES, STORY_PHASES, phaseHref } from "./storyPhases";

describe("story phases", () => {
  it("puts Intro first in the bar, before Watch", () => {
    expect(STORY_BAR_PHASES.map((p) => p.path)).toEqual([
      "intro",
      ...STORY_PHASES.map((p) => p.path),
    ]);
  });

  it("keeps landing phases without Intro", () => {
    expect(STORY_PHASES.map((p) => p.path)).not.toContain("intro");
  });

  it("maps Intro and Watch to the video route", () => {
    expect(phaseHref("2", "intro")).toBe("/stories/2/video");
    expect(phaseHref("2", "video")).toBe("/stories/2/video?watch=1");
    expect(phaseHref("2", "identify")).toBe("/stories/2/identify");
  });
});
