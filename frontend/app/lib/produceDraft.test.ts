import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import {
  loadProduceDraft,
  saveProduceDraft,
  clearProduceDraft,
  produceDraftKey,
} from "./produceDraft";

/** Minimal in-memory Storage; the test DOM doesn't provide localStorage. */
function memoryStorage(): Storage {
  const map = new Map<string, string>();
  return {
    getItem: (k) => map.get(k) ?? null,
    setItem: (k, v) => void map.set(k, String(v)),
    removeItem: (k) => void map.delete(k),
    clear: () => map.clear(),
    key: (i) => Array.from(map.keys())[i] ?? null,
    get length() {
      return map.size;
    },
  };
}

describe("produceDraft", () => {
  beforeEach(() => vi.stubGlobal("localStorage", memoryStorage()));
  afterEach(() => vi.unstubAllGlobals());

  it("round-trips a draft per story and segment", () => {
    saveProduceDraft("7", 10, "שלום");
    expect(loadProduceDraft("7", 10)).toBe("שלום");
    expect(loadProduceDraft("7", 20)).toBe("");
    expect(loadProduceDraft("8", 10)).toBe("");
  });

  it("saving an empty draft removes the entry", () => {
    saveProduceDraft("7", 10, "x");
    saveProduceDraft("7", 10, "");
    expect(localStorage.getItem(produceDraftKey("7", 10))).toBeNull();
  });

  it("clear removes the entry", () => {
    saveProduceDraft("7", 10, "x");
    clearProduceDraft("7", 10);
    expect(loadProduceDraft("7", 10)).toBe("");
  });

  it("survives storage that throws", () => {
    const blocked = new Proxy({} as Storage, {
      get: () => () => {
        throw new Error("blocked");
      },
    });
    vi.stubGlobal("localStorage", blocked);
    expect(() => saveProduceDraft("7", 10, "x")).not.toThrow();
    expect(() => clearProduceDraft("7", 10)).not.toThrow();
    expect(loadProduceDraft("7", 10)).toBe("");
  });

  it("survives localStorage being undefined", () => {
    vi.stubGlobal("localStorage", undefined);
    expect(() => saveProduceDraft("7", 10, "x")).not.toThrow();
    expect(loadProduceDraft("7", 10)).toBe("");
  });
});
