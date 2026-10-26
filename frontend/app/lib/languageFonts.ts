/**
 * Typeface per story language. Each entry becomes an @font-face rule limited
 * to the language's Unicode range, and every face is put ahead of the app's
 * base font stack (see `--language-fonts` in app.css). The browser therefore
 * draws only that script's characters in the language font, wherever they
 * appear, and everything else in the base font.
 *
 * Faces are matched by character, not by the story's language code: two
 * languages that share a script share its characters, and the first matching
 * entry below wins for them.
 */
export interface FontSource {
  /** Names of the installed font to look for on the reader's machine. */
  localNames: string[];
  /** Self-hosted copy under `frontend/public`, e.g. "/fonts/SBL_grk.woff2". */
  url?: string;
}

export interface LanguageFont extends FontSource {
  /** CSS unicode-range the face applies to. */
  unicodeRange: string;
  /** Real bold face; without it the browser thickens the regular one. */
  bold?: FontSource;
}

// Greek and Coptic, plus Greek Extended (polytonic accents and breathings).
const GREEK_RANGE = "U+0370-03FF, U+1F00-1FFF";

const SBL_GREEK: LanguageFont = {
  localNames: ["SBL Greek", "SBLGreek"],
  // SBL Greek is free only for non-commercial use; set this once a licensed
  // copy is in public/fonts. Until then readers need it installed locally.
  url: undefined,
  unicodeRange: GREEK_RANGE,
};

// Hebrew, plus presentation forms (precomposed pointed letters).
const HEBREW_RANGE = "U+0590-05FF, U+FB1D-FB4F";

// The browser's default serif, which Hebrew used before the base stack was
// fixed: Times New Roman on Windows, macOS and iOS, Liberation Serif on Linux.
// Readers with neither get the base stack's sans-serif Hebrew.
const DEFAULT_SERIF_HEBREW: LanguageFont = {
  localNames: ["Times New Roman", "TimesNewRomanPSMT", "Liberation Serif"],
  unicodeRange: HEBREW_RANGE,
  bold: {
    localNames: [
      "Times New Roman Bold",
      "TimesNewRomanPS-BoldMT",
      "Liberation Serif Bold",
    ],
  },
};

/** Keyed by the story's languageCode. */
export const LANGUAGE_FONTS: Record<string, LanguageFont> = {
  grc: SBL_GREEK, // Ancient / Koine Greek
  el: SBL_GREEK, // Modern Greek
  he: DEFAULT_SERIF_HEBREW,
};

const familyName = (code: string) => `"Glossias ${code}"`;

function fontFaceRule(
  code: string,
  source: FontSource,
  unicodeRange: string,
  weight: "normal" | "bold",
): string {
  const sources = source.localNames.map((name) => `local("${name}")`);
  if (source.url) sources.push(`url("${source.url}")`);
  return `@font-face {
  font-family: ${familyName(code)};
  src: ${sources.join(", ")};
  font-weight: ${weight};
  unicode-range: ${unicodeRange};
  font-display: swap;
}`;
}

/** Stylesheet text declaring each language face and the family list. */
export function languageFontsCss(
  fonts: Record<string, LanguageFont> = LANGUAGE_FONTS,
): string {
  const entries = Object.entries(fonts);
  if (entries.length === 0) return "";
  const faces = entries.flatMap(([code, font]) => [
    fontFaceRule(code, font, font.unicodeRange, "normal"),
    ...(font.bold
      ? [fontFaceRule(code, font.bold, font.unicodeRange, "bold")]
      : []),
  ]);
  const families = entries.map(([code]) => familyName(code)).join(", ");
  return `${faces.join("\n")}\n:root { --language-fonts: ${families}; }`;
}
