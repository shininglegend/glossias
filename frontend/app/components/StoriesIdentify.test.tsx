import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, act, fireEvent } from "@testing-library/react";
import { IdentifySession } from "./StoriesIdentify";
import type { IdentifyData } from "../services/api";

// ---- Fake HTMLAudioElement -------------------------------------------------

class FakeAudio extends EventTarget {
  static instances: FakeAudio[] = [];
  src: string;
  preload = "";
  currentTime = 0;
  paused = true;
  playCalls = 0;

  constructor(src = "") {
    super();
    this.src = src;
    FakeAudio.instances.push(this);
  }

  load() {}

  play(): Promise<void> {
    this.playCalls += 1;
    this.paused = false;
    return Promise.resolve();
  }

  pause() {
    this.paused = true;
  }

  ready() {
    this.dispatchEvent(new Event("canplaythrough"));
  }

  end() {
    this.paused = true;
    this.dispatchEvent(new Event("ended"));
  }

  static bySrc(src: string): FakeAudio {
    const a = FakeAudio.instances.find((i) => i.src === src);
    if (!a) throw new Error(`no audio for ${src}`);
    return a;
  }

  static byLine(lineNumber: number): FakeAudio {
    return FakeAudio.bySrc(`url-${lineNumber}`);
  }
}

// ---- Fixtures -------------------------------------------------------------

const WORDS = [
  { id: 10, lexical_form: "כלב", audio_url: "word-10", image_url: "img-10" },
  { id: 11, lexical_form: "ילד", audio_url: "word-11", image_url: "img-11" },
  { id: 12, lexical_form: "בית", audio_url: "word-12", image_url: "img-12" },
  { id: 13, lexical_form: "ספר", audio_url: "word-13", image_url: "img-13" },
  { id: 14, lexical_form: "מים", audio_url: "word-14", image_url: "img-14" },
];

/** Three lines; line index 1 holds target 10, the others hold none. */
const makePageData = (overrides: Partial<IdentifyData> = {}): IdentifyData => ({
  story_id: "1",
  story_title: "Test",
  language: "he",
  lines: [
    { text: [{ type: "text", text: "line 1" }], target_vocab_ids: [] },
    {
      text: [
        { type: "text", text: "the " },
        { type: "target", text: "כלב", target_vocab_id: 10 },
        { type: "text", text: " ran" },
      ],
      target_vocab_ids: [10],
    },
    { text: [{ type: "text", text: "line 3" }], target_vocab_ids: [] },
  ],
  target_words: WORDS,
  audio_urls: { "1": "url-1", "2": "url-2", "3": "url-3" },
  correct_picks: [],
  completed: false,
  ...overrides,
});

const gradeLocally = () =>
  vi.fn(
    async (_line: number, target: number, selected: number) =>
      target === selected,
  );

const setup = async (
  onCheckPick = gradeLocally(),
  pageOverrides: Partial<IdentifyData> = {},
) => {
  const onContinue = vi.fn();
  const utils = render(
    <IdentifySession
      pageData={makePageData(pageOverrides)}
      nextStepName="Translate"
      onCheckPick={onCheckPick}
      onContinue={onContinue}
    />,
  );
  // Complete prefetch of the narration clips.
  await act(async () => {
    FakeAudio.instances.forEach((a) => a.ready());
    await Promise.resolve();
  });
  return { ...utils, onCheckPick, onContinue };
};

const endLine = async (lineNumber: number) => {
  await act(async () => {
    FakeAudio.byLine(lineNumber).end();
    await Promise.resolve();
  });
};

const clickStart = () =>
  fireEvent.click(screen.getByRole("button", { name: /start/i }));

beforeEach(() => {
  FakeAudio.instances = [];
  vi.stubGlobal("Audio", FakeAudio);
});

afterEach(() => {
  vi.unstubAllGlobals();
});

// ---- Tests ----------------------------------------------------------------

describe("IdentifySession", () => {
  it("renders target words distinctly and starts idle", async () => {
    await setup();
    expect(screen.getByText("כלב")).toHaveClass("target-word");
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(screen.getByTestId("identify-counter")).toHaveTextContent("0 / 1");
  });

  it("plays through, pauses after the target line, and opens the quiz", async () => {
    await setup();
    clickStart();
    expect(FakeAudio.byLine(1).playCalls).toBe(1);

    await endLine(1);
    expect(FakeAudio.byLine(2).playCalls).toBe(1);
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();

    await endLine(2);
    // Paused, not continuing to line 3.
    expect(FakeAudio.byLine(3).playCalls).toBe(0);
    const dialog = screen.getByRole("dialog");
    expect(dialog).toBeInTheDocument();
    expect(dialog).toHaveTextContent("כלב");
    // Word pronunciation auto-plays.
    expect(FakeAudio.bySrc("word-10").playCalls).toBe(1);
    // All five pictures are offered.
    expect(screen.getAllByTestId(/identify-option-/)).toHaveLength(5);
  });

  it("marks wrong picks, then replays the line and continues after the right one", async () => {
    const { onCheckPick } = await setup();
    clickStart();
    await endLine(1);
    await endLine(2);

    // Wrong pick: stays open, option disabled, counted server-side.
    await act(async () => {
      fireEvent.click(screen.getByTestId("identify-option-12"));
    });
    expect(onCheckPick).toHaveBeenCalledWith(1, 10, 12);
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(screen.getByTestId("identify-option-12")).toBeDisabled();
    expect(screen.getByText(/not quite/i)).toBeInTheDocument();

    // Right pick: closes, replays line 2.
    await act(async () => {
      fireEvent.click(screen.getByTestId("identify-option-10"));
    });
    expect(onCheckPick).toHaveBeenCalledWith(1, 10, 10);
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(FakeAudio.byLine(2).playCalls).toBe(2);
    expect(screen.getByTestId("identify-counter")).toHaveTextContent("1 / 1");
    expect(FakeAudio.byLine(3).playCalls).toBe(0);

    // Replay ends → continues with line 3, no second quiz.
    await endLine(2);
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(FakeAudio.byLine(3).playCalls).toBe(1);

    // Story ends → completion message.
    await endLine(3);
    expect(screen.getByText(/great job/i)).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /continue to translate/i }),
    ).toBeInTheDocument();
  });

  it("keeps the quiz open with an error when grading fails", async () => {
    const failing = vi.fn(async () => {
      throw new Error("network");
    });
    await setup(failing);
    clickStart();
    await endLine(1);
    await endLine(2);

    await act(async () => {
      fireEvent.click(screen.getByTestId("identify-option-10"));
    });
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(screen.getByText(/couldn't save/i)).toBeInTheDocument();
    // Nothing was disabled: the pick was never graded.
    expect(screen.getByTestId("identify-option-10")).not.toBeDisabled();
  });

  it("counts every occurrence as its own quiz, not distinct words", async () => {
    const base = makePageData();
    await setup(gradeLocally(), {
      lines: [
        ...base.lines,
        {
          text: [{ type: "target", text: "כלב", target_vocab_id: 10 }],
          target_vocab_ids: [10],
        },
      ],
      audio_urls: { ...base.audio_urls, "4": "url-4" },
    });
    // The same word appears twice → two quizzes, one target word.
    expect(screen.getByTestId("identify-counter")).toHaveTextContent("0 / 2");

    clickStart();
    await endLine(1);
    await endLine(2);
    await act(async () => {
      fireEvent.click(screen.getByTestId("identify-option-10"));
    });
    expect(screen.getByTestId("identify-counter")).toHaveTextContent("1 / 2");
  });

  it("opens finished and cannot be replayed once the server says complete", async () => {
    await setup(gradeLocally(), {
      correct_picks: [{ line_index: 1, target_vocab_id: 10 }],
      completed: true,
    });
    expect(screen.getByText(/great job/i)).toBeInTheDocument();
    expect(screen.getByTestId("identify-finished")).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /start/i }),
    ).not.toBeInTheDocument();
    expect(screen.queryByTestId("identify-counter")).not.toBeInTheDocument();
    // No narration was started.
    expect(FakeAudio.instances.every((a) => a.playCalls === 0)).toBe(true);
  });

  it("resumes at the last identified line and skips answered quizzes", async () => {
    const { onCheckPick } = await setup(gradeLocally(), {
      correct_picks: [{ line_index: 1, target_vocab_id: 10 }],
      completed: false,
    });
    expect(screen.getByTestId("identify-resumed")).toHaveTextContent("line 2");
    expect(screen.getByTestId("identify-counter")).toHaveTextContent("1 / 1");

    clickStart();
    // Starts at line 2, not line 1.
    expect(FakeAudio.byLine(1).playCalls).toBe(0);
    expect(FakeAudio.byLine(2).playCalls).toBe(1);

    // Line 2's quiz was already answered: no dialog, straight on to line 3.
    await endLine(2);
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(onCheckPick).not.toHaveBeenCalled();
    expect(FakeAudio.byLine(3).playCalls).toBe(1);

    await endLine(3);
    expect(screen.getByText(/great job/i)).toBeInTheDocument();
  });

  it("pause and resume restart the current line without opening a quiz", async () => {
    await setup();
    clickStart();
    await endLine(1);
    // Now on line 2 (a target line), mid-clip.
    fireEvent.click(screen.getByRole("button", { name: /pause audio/i }));
    expect(FakeAudio.byLine(2).paused).toBe(true);
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /resume audio/i }));
    expect(FakeAudio.byLine(2).playCalls).toBe(2);
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });
});
