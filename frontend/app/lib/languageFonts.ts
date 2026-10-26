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
export interface LanguageFont {
  /** Names of the installed font to look for on the reader's machine. */
  localNames: string[];
  /** Self-hosted copy under `frontend/public`, e.g. "/fonts/SBL_grk.woff2". */
  url?: string;
  /** CSS unicode-range the face applies to. */
  unicodeRange: string;
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

/** Keyed by the story's languageCode. */
export const LANGUAGE_FONTS: Record<string, LanguageFont> = {
  grc: SBL_GREEK, // Ancient / Koine Greek
  el: SBL_GREEK, // Modern Greek
};

const familyName = (code: string) => `"Glossias ${code}"`;

function fontFaceRule(code: string, font: LanguageFont): string {
  const sources = font.localNames.map((name) => `local("${name}")`);
  if (font.url) sources.push(`url("${font.url}")`);
  return `@font-face {
  font-family: ${familyName(code)};
  src: ${sources.join(", ")};
  unicode-range: ${font.unicodeRange};
  font-display: swap;
}`;
}

/** Stylesheet text declaring each language face and the family list. */
export function languageFontsCss(
  fonts: Record<string, LanguageFont> = LANGUAGE_FONTS,
): string {
  const entries = Object.entries(fonts);
  if (entries.length === 0) return "";
  const faces = entries.map(([code, font]) => fontFaceRule(code, font));
  const families = entries.map(([code]) => familyName(code)).join(", ");
  return `${faces.join("\n")}\n:root { --language-fonts: ${families}; }`;
}
