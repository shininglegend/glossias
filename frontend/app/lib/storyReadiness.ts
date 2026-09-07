import type { PhaseReadiness, StoryContentReadiness } from "../types/admin";

export type StoryReadinessPhase = keyof StoryContentReadiness;
export type StoryReadinessMap = Partial<StoryContentReadiness>;

const cache = new Map<number, StoryReadinessMap>();

export function getCachedStoryReadiness(storyId: number): StoryReadinessMap {
  return cache.get(storyId) ?? {};
}

export function readinessEqual(a: PhaseReadiness, b: PhaseReadiness): boolean {
  if (
    a.phase !== b.phase ||
    a.ready !== b.ready ||
    a.issues.length !== b.issues.length
  ) {
    return false;
  }
  return a.issues.every(
    (issue, i) =>
      issue.field === b.issues[i]?.field &&
      issue.message === b.issues[i]?.message,
  );
}

export function setCachedStoryPhase(
  storyId: number,
  phase: StoryReadinessPhase,
  value: PhaseReadiness,
): StoryReadinessMap {
  const current = cache.get(storyId) ?? {};
  const prev = current[phase];
  if (prev && readinessEqual(prev, value)) return current;
  const next = { ...current, [phase]: value };
  cache.set(storyId, next);
  return next;
}

export function clearStoryReadinessCache(): void {
  cache.clear();
}

export function videoReadiness(videoUrl: string | undefined): PhaseReadiness {
  const issues = videoUrl?.trim()
    ? []
    : [{ field: "videoUrl", message: "story has no video link" }];
  return { phase: "video", ready: issues.length === 0, issues };
}

export function translateReadiness(
  lines: { english: string }[],
): PhaseReadiness {
  const missing = lines.filter((line) => line.english.trim() === "").length;
  const issues =
    missing === 0
      ? []
      : [
          {
            field: "translations",
            message:
              missing === 1
                ? "1 line has no English translation"
                : `${missing} lines have no English translation`,
          },
        ];
  return { phase: "translate", ready: issues.length === 0, issues };
}
