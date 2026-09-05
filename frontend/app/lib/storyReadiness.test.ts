import { describe, it, expect, beforeEach } from "vitest";
import {
  clearStoryReadinessCache,
  getCachedStoryReadiness,
  setCachedStoryPhase,
  videoReadiness,
} from "./storyReadiness";

const ready = {
  phase: "video",
  ready: true,
  issues: [],
};

const missing = videoReadiness("");

describe("videoReadiness", () => {
  it("is ready when a URL is present", () => {
    expect(videoReadiness("https://example.com/v").ready).toBe(true);
  });

  it("flags a blank or missing URL", () => {
    expect(videoReadiness("").ready).toBe(false);
    expect(videoReadiness("   ").ready).toBe(false);
    expect(videoReadiness(undefined).ready).toBe(false);
  });
});

describe("story readiness cache", () => {
  beforeEach(() => {
    clearStoryReadinessCache();
  });

  it("stores a phase per story and leaves other stories alone", () => {
    setCachedStoryPhase(2, "video", ready);
    expect(getCachedStoryReadiness(2).video?.ready).toBe(true);
    expect(getCachedStoryReadiness(3)).toEqual({});
  });

  it("returns the same map when the report is unchanged", () => {
    const first = setCachedStoryPhase(2, "video", missing);
    const second = setCachedStoryPhase(2, "video", videoReadiness(""));
    expect(second).toBe(first);
  });
});
