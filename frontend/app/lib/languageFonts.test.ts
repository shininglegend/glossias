import { describe, it, expect } from "vitest";
import { LANGUAGE_FONTS, languageFontsCss } from "./languageFonts";

describe("languageFontsCss", () => {
  it("uses SBL Greek for Greek stories", () => {
    for (const code of ["grc", "el"]) {
      expect(LANGUAGE_FONTS[code].localNames).toContain("SBL Greek");
    }
    const css = languageFontsCss();
    expect(css).toContain('local("SBL Greek")');
    expect(css).toContain("unicode-range: U+0370-03FF, U+1F00-1FFF;");
  });

  it("keeps Hebrew in the default serif it used before", () => {
    const css = languageFontsCss({ he: LANGUAGE_FONTS.he });
    expect(css).toContain('local("Times New Roman")');
    expect(css).toContain('local("Times New Roman Bold")');
    expect(css).toContain("font-weight: bold;");
    expect(css).toContain("unicode-range: U+0590-05FF, U+FB1D-FB4F;");
  });

  it("lists every language face in the family variable", () => {
    const css = languageFontsCss({
      xx: { localNames: ["A"], unicodeRange: "U+0041" },
      yy: { localNames: ["B"], url: "/fonts/b.woff2", unicodeRange: "U+0042" },
    });
    expect(css).toContain('src: local("B"), url("/fonts/b.woff2");');
    expect(css).toContain(
      ':root { --language-fonts: "Glossias xx", "Glossias yy"; }',
    );
  });

  it("emits nothing when no language fonts are configured", () => {
    expect(languageFontsCss({})).toBe("");
  });
});
