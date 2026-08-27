import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useAudioPlayer } from "./AudioPlayer";
import type { VocabData, VocabLine } from "../../services/api";

// ---- Fake HTMLAudioElement -------------------------------------------------

class FakeAudio extends EventTarget {
  static instances: FakeAudio[] = [];
  src: string;
  preload = "";
  currentTime = 0;
  paused = true;
  playCalls = 0;
  pauseCalls = 0;
  failPlay = false;

  constructor(src: string) {
    super();
    this.src = src;
    FakeAudio.instances.push(this);
  }

  load() {}

  play(): Promise<void> {
    this.playCalls += 1;
    if (this.failPlay) return Promise.reject(new Error("play failed"));
    this.paused = false;
    return Promise.resolve();
  }

  pause() {
    this.pauseCalls += 1;
    this.paused = true;
  }

  /** Simulate the prefetch completing. */
  ready() {
    this.dispatchEvent(new Event("canplaythrough"));
  }

  /** Simulate the clip finishing. */
  end() {
    this.paused = true;
    this.dispatchEvent(new Event("ended"));
  }

  static byLine(lineNumber: number): FakeAudio {
    const a = FakeAudio.instances.find((i) => i.src === `url-${lineNumber}`);
    if (!a) throw new Error(`no audio for line ${lineNumber}`);
    return a;
  }
}

// ---- Fixtures -------------------------------------------------------------

const line = (n: number, withBlank = false): VocabLine => ({
  text: withBlank
    ? [
        { type: "text", text: "foo " },
        { type: "blank", text: "", vocab_key: `${n - 1}-0` },
      ]
    : [{ type: "text", text: `line ${n}` }],
  audio_files: [],
});

const makePageData = (count: number, blankLines: number[] = []): VocabData =>
  ({
    story_id: 1,
    story_title: "Test",
    language: "he",
    vocab_bank: [],
    lines: Array.from({ length: count }, (_, i) =>
      line(i + 1, blankLines.includes(i)),
    ),
  }) as unknown as VocabData;

const makeURLs = (count: number): Record<string, string> =>
  Object.fromEntries(
    Array.from({ length: count }, (_, i) => [`${i + 1}`, `url-${i + 1}`]),
  );

type HookProps = Parameters<typeof useAudioPlayer>[0];

const LINE_COUNT = 5;

const setup = async (overrides: Partial<HookProps> = {}) => {
  const onPlayedLinesChange = vi.fn();
  const onCurrentLineChange = vi.fn();
  const onPlayingStateChange = vi.fn();
  const audioURLs = makeURLs(LINE_COUNT);

  const initialProps: HookProps = {
    audioURLs,
    pageData: makePageData(LINE_COUNT),
    onPlayedLinesChange,
    onCurrentLineChange,
    onPlayingStateChange,
    completedLines: new Set<number>(),
    ...overrides,
  };

  const hook = renderHook((props: HookProps) => useAudioPlayer(props), {
    initialProps,
  });

  // Complete prefetch for every line.
  await act(async () => {
    FakeAudio.instances.forEach((a) => a.ready());
    await Promise.resolve();
  });

  const rerenderWith = (patch: Partial<HookProps>) => {
    Object.assign(initialProps, patch);
    hook.rerender({ ...initialProps });
  };

  return {
    ...hook,
    rerenderWith,
    onPlayedLinesChange,
    onCurrentLineChange,
    onPlayingStateChange,
  };
};

const endLine = async (lineNumber: number) => {
  await act(async () => {
    FakeAudio.byLine(lineNumber).end();
  });
};

// ---- Tests ----------------------------------------------------------------

beforeEach(() => {
  FakeAudio.instances = [];
  vi.stubGlobal("Audio", FakeAudio);
  vi.useFakeTimers();
});

afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

describe("useAudioPlayer — existing behaviour", () => {
  it("prefetches one Audio per URL", async () => {
    const { result } = await setup();
    expect(FakeAudio.instances).toHaveLength(LINE_COUNT);
    expect(Object.keys(result.current.prefetchedAudio)).toHaveLength(
      LINE_COUNT,
    );
  });

  it("plays lines sequentially and reports played lines", async () => {
    const { result, onPlayedLinesChange } = await setup();

    act(() => result.current.playStoryAudio());
    expect(result.current.isPlaying).toBe(true);
    expect(result.current.currentLineIndex).toBe(0);
    expect(FakeAudio.byLine(1).playCalls).toBe(1);

    await endLine(1);
    expect(result.current.currentLineIndex).toBe(1);
    expect(FakeAudio.byLine(2).playCalls).toBe(1);
    expect(result.current.playedLines).toEqual(new Set([0]));
    expect(onPlayedLinesChange).toHaveBeenLastCalledWith(new Set([0]));
  });

  it("stops and resets to line 0 after the last line", async () => {
    const { result } = await setup();
    act(() => result.current.playStoryAudio());
    for (let n = 1; n <= LINE_COUNT; n++) await endLine(n);

    expect(result.current.isPlaying).toBe(false);
    expect(result.current.currentLineIndex).toBe(0);
    expect(result.current.playedLines).toEqual(new Set([0, 1, 2, 3, 4]));
  });

  it("by default pauses after lines containing vocab blanks", async () => {
    const { result } = await setup({
      pageData: makePageData(LINE_COUNT, [1]),
    });
    act(() => result.current.playStoryAudio());
    await endLine(1);
    expect(result.current.isPlaying).toBe(true);

    await endLine(2);
    expect(result.current.isPlaying).toBe(false);
    expect(result.current.currentLineIndex).toBe(1);
    expect(FakeAudio.byLine(3).playCalls).toBe(0);
  });

  it("with pauseAfterEveryLine, pauses after every uncompleted line", async () => {
    const { result, rerenderWith } = await setup({
      pauseAfterEveryLine: true,
    });
    act(() => result.current.playStoryAudio());
    await endLine(1);
    expect(result.current.isPlaying).toBe(false);
    expect(FakeAudio.byLine(2).playCalls).toBe(0);

    // Consumer marks line complete and continues.
    rerenderWith({ completedLines: new Set([0]) });
    act(() => result.current.playNextLineFromIndex(0));
    expect(result.current.isPlaying).toBe(true);
    expect(result.current.currentLineIndex).toBe(1);
    expect(FakeAudio.byLine(2).playCalls).toBe(1);
  });

  it("playStoryAudio toggles pause and resumes from the current line", async () => {
    const { result } = await setup();
    act(() => result.current.playStoryAudio());
    await endLine(1);

    act(() => result.current.playStoryAudio());
    expect(result.current.isPlaying).toBe(false);
    expect(FakeAudio.byLine(2).pauseCalls).toBe(1);
    expect(result.current.currentLineIndex).toBe(1);

    act(() => result.current.playStoryAudio());
    expect(result.current.isPlaying).toBe(true);
    expect(FakeAudio.byLine(2).playCalls).toBe(2);
  });

  it("playLineAudio plays a single line, marks it played and stops", async () => {
    const { result } = await setup();
    act(() => result.current.playLineAudio(2));
    expect(result.current.isPlaying).toBe(true);
    expect(result.current.currentLineIndex).toBe(2);

    await endLine(3);
    expect(result.current.isPlaying).toBe(false);
    expect(result.current.playedLines).toEqual(new Set([2]));
    expect(FakeAudio.byLine(4).playCalls).toBe(0);
  });

  it("skips lines whose audio failed to play", async () => {
    vi.spyOn(console, "error").mockImplementation(() => {});
    const { result } = await setup();
    FakeAudio.byLine(2).failPlay = true;

    act(() => result.current.playStoryAudio());
    await endLine(1);
    // Rejected play() promise resolves on a microtask.
    await act(async () => {
      await Promise.resolve();
    });
    expect(result.current.currentLineIndex).toBe(2);
    expect(FakeAudio.byLine(3).playCalls).toBe(1);
  });
});

describe("useAudioPlayer — pauseOnLines", () => {
  it("pauses only after the listed lines", async () => {
    const { result } = await setup({
      pageData: makePageData(LINE_COUNT, [0]), // blank on line 0 must be ignored
      pauseOnLines: new Set([2]),
    });
    act(() => result.current.playStoryAudio());

    await endLine(1);
    expect(result.current.isPlaying).toBe(true);
    await endLine(2);
    expect(result.current.isPlaying).toBe(true);
    await endLine(3);
    expect(result.current.isPlaying).toBe(false);
    expect(result.current.currentLineIndex).toBe(2);
    expect(FakeAudio.byLine(4).playCalls).toBe(0);
  });

  it("takes precedence over pauseAfterEveryLine", async () => {
    const { result } = await setup({
      pauseAfterEveryLine: true,
      pauseOnLines: new Set([3]),
    });
    act(() => result.current.playStoryAudio());
    await endLine(1);
    expect(result.current.isPlaying).toBe(true);
    await endLine(2);
    await endLine(3);
    expect(result.current.isPlaying).toBe(true);
    await endLine(4);
    expect(result.current.isPlaying).toBe(false);
  });

  it("does not pause on a listed line that is already completed", async () => {
    const { result } = await setup({
      pauseOnLines: new Set([0]),
      completedLines: new Set([0]),
    });
    act(() => result.current.playStoryAudio());
    await endLine(1);
    expect(result.current.isPlaying).toBe(true);
    expect(FakeAudio.byLine(2).playCalls).toBe(1);
  });

  it("uses the latest pauseOnLines even if it changes mid-line", async () => {
    const { result, rerenderWith } = await setup({
      pauseOnLines: new Set([0]),
    });
    act(() => result.current.playStoryAudio());
    // While line 1 plays, consumer decides not to pause on it after all.
    rerenderWith({ pauseOnLines: new Set([1]) });
    await endLine(1);
    expect(result.current.isPlaying).toBe(true);
    await endLine(2);
    expect(result.current.isPlaying).toBe(false);
  });
});

describe("useAudioPlayer — skipLines", () => {
  it("skips listed lines without playing or marking them", async () => {
    const { result } = await setup({ skipLines: new Set([1, 2]) });
    act(() => result.current.playStoryAudio());
    await endLine(1);

    expect(FakeAudio.byLine(2).playCalls).toBe(0);
    expect(FakeAudio.byLine(3).playCalls).toBe(0);
    expect(FakeAudio.byLine(4).playCalls).toBe(1);
    expect(result.current.currentLineIndex).toBe(3);
    expect(result.current.playedLines).toEqual(new Set([0]));
  });

  it("skips the starting line when resuming onto a skipped line", async () => {
    const { result } = await setup({ skipLines: new Set([0]) });
    act(() => result.current.playStoryAudio());
    expect(FakeAudio.byLine(1).playCalls).toBe(0);
    expect(FakeAudio.byLine(2).playCalls).toBe(1);
    expect(result.current.currentLineIndex).toBe(1);
  });

  it("finishes cleanly when the trailing lines are all skipped", async () => {
    const { result } = await setup({ skipLines: new Set([3, 4]) });
    act(() => result.current.playStoryAudio());
    await endLine(1);
    await endLine(2);
    await endLine(3);
    expect(result.current.isPlaying).toBe(false);
    expect(result.current.currentLineIndex).toBe(0);
    expect(result.current.playedLines).toEqual(new Set([0, 1, 2]));
  });

  it("honours skipLines updated after playback started", async () => {
    const { result, rerenderWith } = await setup();
    act(() => result.current.playStoryAudio());
    rerenderWith({ skipLines: new Set([1]) });
    await endLine(1);
    expect(FakeAudio.byLine(2).playCalls).toBe(0);
    expect(FakeAudio.byLine(3).playCalls).toBe(1);
  });
});

describe("useAudioPlayer — replayLine", () => {
  it("replays a line without changing currentLineIndex or playedLines", async () => {
    const { result, onPlayedLinesChange } = await setup({
      pauseOnLines: new Set([1]),
    });
    act(() => result.current.playStoryAudio());
    await endLine(1);
    await endLine(2); // paused on line index 1
    expect(result.current.isPlaying).toBe(false);
    const playedCalls = onPlayedLinesChange.mock.calls.length;

    let done = false;
    act(() => {
      result.current.replayLine(1).then(() => {
        done = true;
      });
    });
    expect(result.current.isPlaying).toBe(true);
    expect(FakeAudio.byLine(2).playCalls).toBe(2);
    expect(result.current.currentLineIndex).toBe(1);

    await endLine(2);
    await act(async () => {
      await Promise.resolve();
    });
    expect(done).toBe(true);
    expect(result.current.isPlaying).toBe(false);
    expect(result.current.currentLineIndex).toBe(1);
    expect(result.current.playedLines).toEqual(new Set([0, 1]));
    expect(onPlayedLinesChange.mock.calls.length).toBe(playedCalls);
    // Sequential playback did not continue on its own.
    expect(FakeAudio.byLine(3).playCalls).toBe(0);
  });

  it("interrupts in-progress sequential playback without continuing it", async () => {
    const { result } = await setup();
    act(() => result.current.playStoryAudio());
    expect(FakeAudio.byLine(1).playCalls).toBe(1);

    act(() => {
      void result.current.replayLine(3);
    });
    expect(FakeAudio.byLine(1).pauseCalls).toBe(1);
    expect(FakeAudio.byLine(4).playCalls).toBe(1);

    // The interrupted line's `ended` must no longer drive the sequence.
    await endLine(1);
    expect(FakeAudio.byLine(2).playCalls).toBe(0);
    expect(result.current.playedLines).toEqual(new Set());
  });

  it("resolves immediately when the line has no audio", async () => {
    const { result } = await setup();
    await expect(result.current.replayLine(99)).resolves.toBeUndefined();
    expect(result.current.isPlaying).toBe(false);
  });

  it("resolves (without throwing) when play() fails", async () => {
    vi.spyOn(console, "error").mockImplementation(() => {});
    const { result } = await setup();
    FakeAudio.byLine(1).failPlay = true;
    await act(async () => {
      await result.current.replayLine(0);
    });
    expect(result.current.isPlaying).toBe(false);
  });
});

describe("useAudioPlayer — scheduleResume", () => {
  it("continues from the next line after the delay", async () => {
    const { result } = await setup({ pauseAfterEveryLine: true });
    act(() => result.current.playStoryAudio());
    await endLine(1);
    expect(result.current.isPlaying).toBe(false);

    act(() => result.current.scheduleResume(2000));
    expect(result.current.isResumeScheduled).toBe(true);
    expect(FakeAudio.byLine(2).playCalls).toBe(0);

    act(() => {
      vi.advanceTimersByTime(1999);
    });
    expect(FakeAudio.byLine(2).playCalls).toBe(0);

    act(() => {
      vi.advanceTimersByTime(1);
    });
    expect(result.current.isResumeScheduled).toBe(false);
    expect(result.current.isPlaying).toBe(true);
    expect(result.current.currentLineIndex).toBe(1);
    expect(FakeAudio.byLine(2).playCalls).toBe(1);
  });

  it("can resume from an explicit line index", async () => {
    const { result } = await setup();
    act(() => result.current.scheduleResume(500, 3));
    act(() => {
      vi.advanceTimersByTime(500);
    });
    expect(result.current.currentLineIndex).toBe(3);
    expect(FakeAudio.byLine(4).playCalls).toBe(1);
  });

  it("is cancelled by cancelScheduledResume", async () => {
    const { result } = await setup();
    act(() => result.current.scheduleResume(1000));
    act(() => result.current.cancelScheduledResume());
    expect(result.current.isResumeScheduled).toBe(false);
    act(() => {
      vi.advanceTimersByTime(5000);
    });
    expect(result.current.isPlaying).toBe(false);
    expect(FakeAudio.instances.every((a) => a.playCalls === 0)).toBe(true);
  });

  it("is cancelled by pauseAudio and stopAudio", async () => {
    const { result } = await setup();
    act(() => result.current.scheduleResume(1000));
    act(() => result.current.pauseAudio());
    expect(result.current.isResumeScheduled).toBe(false);

    act(() => result.current.scheduleResume(1000));
    act(() => result.current.stopAudio());
    expect(result.current.isResumeScheduled).toBe(false);

    act(() => {
      vi.advanceTimersByTime(5000);
    });
    expect(FakeAudio.instances.every((a) => a.playCalls === 0)).toBe(true);
  });

  it("replaces an earlier pending resume", async () => {
    const { result } = await setup();
    act(() => result.current.scheduleResume(1000, 1));
    act(() => result.current.scheduleResume(3000, 2));
    act(() => {
      vi.advanceTimersByTime(1000);
    });
    expect(FakeAudio.byLine(2).playCalls).toBe(0);
    act(() => {
      vi.advanceTimersByTime(2000);
    });
    expect(FakeAudio.byLine(2).playCalls).toBe(0);
    expect(FakeAudio.byLine(3).playCalls).toBe(1);
  });

  it("does not fire after unmount", async () => {
    const { result, unmount } = await setup();
    act(() => result.current.scheduleResume(1000));
    unmount();
    act(() => {
      vi.advanceTimersByTime(5000);
    });
    expect(FakeAudio.instances.every((a) => a.playCalls === 0)).toBe(true);
  });
});

describe("useAudioPlayer — cleanup", () => {
  it("pauses the playing audio on unmount", async () => {
    const { result, unmount } = await setup();
    act(() => result.current.playStoryAudio());
    unmount();
    expect(FakeAudio.byLine(1).pauseCalls).toBe(1);
    // A late `ended` must not start the next line.
    FakeAudio.byLine(1).end();
    expect(FakeAudio.byLine(2).playCalls).toBe(0);
  });
});

describe("useAudioPlayer — onPlaybackEnd", () => {
  it("fires once when sequential playback runs past the last line", async () => {
    const onPlaybackEnd = vi.fn();
    const { result } = await setup({ onPlaybackEnd });

    act(() => result.current.playStoryAudio());
    for (let n = 1; n <= LINE_COUNT; n++) await endLine(n);

    expect(onPlaybackEnd).toHaveBeenCalledTimes(1);
    expect(result.current.isPlaying).toBe(false);
    expect(result.current.currentLineIndex).toBe(0);
  });

  it("fires when the trailing lines are all skipped", async () => {
    const onPlaybackEnd = vi.fn();
    const { result } = await setup({
      onPlaybackEnd,
      skipLines: new Set([3, 4]),
    });

    act(() => result.current.playStoryAudio());
    for (let n = 1; n <= 3; n++) await endLine(n);

    expect(onPlaybackEnd).toHaveBeenCalledTimes(1);
  });

  it("is not fired by pauses, stops, or a pauseOnLines pause", async () => {
    const onPlaybackEnd = vi.fn();
    const { result } = await setup({
      onPlaybackEnd,
      pauseOnLines: new Set([LINE_COUNT - 1]),
    });

    act(() => result.current.playStoryAudio());
    act(() => result.current.pauseAudio());
    act(() => result.current.stopAudio());
    act(() => result.current.playStoryAudio());
    for (let n = 1; n <= LINE_COUNT; n++) await endLine(n);

    // Paused after the last line rather than running off the end.
    expect(result.current.isPlaying).toBe(false);
    expect(onPlaybackEnd).not.toHaveBeenCalled();
  });

  it("uses the latest callback even when it changes mid-playback", async () => {
    const first = vi.fn();
    const second = vi.fn();
    const { result, rerenderWith } = await setup({ onPlaybackEnd: first });

    act(() => result.current.playStoryAudio());
    await endLine(1);
    rerenderWith({ onPlaybackEnd: second });
    for (let n = 2; n <= LINE_COUNT; n++) await endLine(n);

    expect(first).not.toHaveBeenCalled();
    expect(second).toHaveBeenCalledTimes(1);
  });
});
