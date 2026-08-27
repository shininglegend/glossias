/**
 * Per-browser persistence of the Produce textarea, so a reload mid-countdown
 * keeps what the student had typed (the countdown itself is server-side).
 *
 * Drafts are keyed by story and segment. `localStorage` can be unavailable
 * (private windows, storage disabled) or throw, so every call is guarded and
 * the page must work with nothing stored.
 */

const PREFIX = "produce-draft";

export function produceDraftKey(storyId: string, segmentId: number): string {
  return `${PREFIX}:${storyId}:${segmentId}`;
}

export function loadProduceDraft(storyId: string, segmentId: number): string {
  try {
    return localStorage.getItem(produceDraftKey(storyId, segmentId)) ?? "";
  } catch {
    return "";
  }
}

export function saveProduceDraft(
  storyId: string,
  segmentId: number,
  text: string,
): void {
  try {
    const key = produceDraftKey(storyId, segmentId);
    if (text === "") {
      localStorage.removeItem(key);
    } else {
      localStorage.setItem(key, text);
    }
  } catch {
    // Storage unavailable: the draft simply isn't kept across reloads.
  }
}

export function clearProduceDraft(storyId: string, segmentId: number): void {
  try {
    localStorage.removeItem(produceDraftKey(storyId, segmentId));
  } catch {
    // Nothing to clear.
  }
}
