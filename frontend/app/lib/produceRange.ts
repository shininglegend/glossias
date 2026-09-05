/** Keep a manual edit; adopt the new range text only when the field is still generated. */
export function adoptRangeText(
  current: string,
  previousGenerated: string,
  nextGenerated: string,
  force: boolean,
): string {
  if (force) return nextGenerated;
  if (current.trim() === "" || current === previousGenerated)
    return nextGenerated;
  return current;
}

/** First click sets a single line; the next click expands to that range. */
export function clickLineRange(
  start: number | "",
  end: number | "",
  clicked: number,
): { start: number; end: number } {
  if (start === "" || end === "" || start !== end) {
    return { start: clicked, end: clicked };
  }
  return {
    start: Math.min(start, clicked),
    end: Math.max(end, clicked),
  };
}

/** True when current text is a manual edit that the new range would replace. */
export function wouldOverrideEdit(
  current: string,
  previousGenerated: string,
  nextGenerated: string,
): boolean {
  if (current.trim() === "" || current === previousGenerated) return false;
  return current !== nextGenerated;
}

/** Joins a story's Hebrew text for lines start..end (inclusive). */
export function hebrewForRange(
  storyLines: { lineNumber: number; text: string }[],
  start: number,
  end: number,
): string {
  return storyLines
    .filter((line) => line.lineNumber >= start && line.lineNumber <= end)
    .map((line) => line.text)
    .join("\n");
}

/**
 * Joins the English translations for lines start..end. Lines with no stored
 * translation are skipped rather than left as gaps.
 */
export function englishForRange(
  translations: Map<number, string>,
  start: number,
  end: number,
): string {
  const parts: string[] = [];
  for (let n = start; n <= end; n++) {
    const text = translations.get(n);
    if (text) parts.push(text);
  }
  return parts.join("\n");
}
