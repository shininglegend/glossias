import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, act, fireEvent, within } from "@testing-library/react";
import { RecallSession } from "./StoriesRecall";
import type { RecallData } from "../services/api";

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

  static byLine(lineNumber: number): FakeAudio {
    const a = FakeAudio.instances.find((i) => i.src === `url-${lineNumber}`);
    if (!a) throw new Error(`no audio for line ${lineNumber}`);
    return a;
  }

  static bySentence(id: number): FakeAudio {
    const a = FakeAudio.instances.find((i) => i.src === `sent-${id}`);
    if (!a) throw new Error(`no sentence audio ${id}`);
    return a;
  }
}

// ---- Fixtures -------------------------------------------------------------

/** Correct story order is 1,2,3,4,5; the server hands them out shuffled. */
const SENTENCES = [
  {
    id: 3,
    hebrew_text: "שלוש",
    image_url: "img-3",
    audio_urls: ["sent-3"],
    text: [
      { type: "text" as const, text: "ש" },
      { type: "target" as const, text: "לוש", target_vocab_id: 3 },
    ],
  },
  { id: 1, hebrew_text: "אחת", image_url: "img-1", audio_urls: ["sent-1"] },
  { id: 5, hebrew_text: "חמש", audio_urls: ["sent-5"] },
  { id: 2, hebrew_text: "שתיים", image_url: "img-2", audio_urls: ["sent-2"] },
  { id: 4, hebrew_text: "ארבע", image_url: "img-4", audio_urls: ["sent-4"] },
];

const makePageData = (overrides: Partial<RecallData> = {}): RecallData => ({
  story_id: "1",
  story_title: "Test",
  language: "he",
  line_count: 3,
  audio_urls: { "1": "url-1", "2": "url-2", "3": "url-3" },
  sentences: SENTENCES,
  attempts: 0,
  completed: false,
  ...overrides,
});

/** Grades against the true order 1..5 (sentence id === position). */
const gradeLocally = () =>
  vi.fn(async (sentenceId: number, position: number) => ({
    correct: sentenceId === position,
  }));

const setup = async (
  pageOverrides: Partial<RecallData> = {},
  onCheckPick = gradeLocally(),
) => {
  const onContinue = vi.fn();
  const utils = render(
    <RecallSession
      pageData={makePageData(pageOverrides)}
      nextStepName="Score"
      onCheckPick={onCheckPick}
      onContinue={onContinue}
    />,
  );
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

const listenThrough = async () => {
  fireEvent.click(screen.getByRole("button", { name: /start/i }));
  await endLine(1);
  await endLine(2);
  await endLine(3);
};

/** Sentence IDs in their current on-screen order. */
const cardOrder = () =>
  within(screen.getByTestId("recall-cards"))
    .getAllByRole("listitem")
    .map((el) => Number(el.dataset.testid?.replace("recall-card-", "")));

const pickCard = async (id: number) => {
  await act(async () => {
    fireEvent.click(screen.getByTestId(`recall-card-${id}`));
  });
};

// happy-dom under vitest does not provide localStorage; a minimal in-memory
// stand-in is enough to exercise the resume-on-reload behaviour.
const makeStorage = () => {
  const store = new Map<string, string>();
  return {
    getItem: (k: string) => store.get(k) ?? null,
    setItem: (k: string, v: string) => void store.set(k, String(v)),
    removeItem: (k: string) => void store.delete(k),
    clear: () => store.clear(),
  };
};

beforeEach(() => {
  FakeAudio.instances = [];
  vi.stubGlobal("Audio", FakeAudio);
  vi.stubGlobal("localStorage", makeStorage());
});

afterEach(() => {
  vi.unstubAllGlobals();
});

// ---- Tests ----------------------------------------------------------------

describe("RecallSession", () => {
  it("starts idle with no text or cards shown, then plays straight through", async () => {
    await setup();
    expect(screen.getByTestId("recall-listening")).toBeInTheDocument();
    expect(screen.queryByTestId("recall-cards")).not.toBeInTheDocument();
    expect(screen.queryByText("אחת")).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /start/i }));
    expect(FakeAudio.byLine(1).playCalls).toBe(1);
    expect(screen.getByTestId("recall-progress")).toHaveTextContent(
      "Line 1 of 3",
    );

    // No pause between lines, even though no line is "completed".
    await endLine(1);
    expect(FakeAudio.byLine(2).playCalls).toBe(1);
    expect(screen.getByRole("progressbar")).toHaveAttribute(
      "aria-valuenow",
      "33",
    );
    await endLine(2);
    expect(FakeAudio.byLine(3).playCalls).toBe(1);
    expect(screen.queryByTestId("recall-cards")).not.toBeInTheDocument();

    await endLine(3);
    expect(screen.queryByTestId("recall-listening")).not.toBeInTheDocument();
    expect(screen.getByTestId("recall-cards")).toBeInTheDocument();
  });

  it("can pause and resume without losing its place", async () => {
    await setup();
    fireEvent.click(screen.getByRole("button", { name: /start/i }));
    await endLine(1);

    fireEvent.click(screen.getByRole("button", { name: /pause audio/i }));
    expect(FakeAudio.byLine(2).paused).toBe(true);

    fireEvent.click(screen.getByRole("button", { name: /resume audio/i }));
    expect(FakeAudio.byLine(2).playCalls).toBe(2);
    expect(FakeAudio.byLine(3).playCalls).toBe(0);
  });

  it("lets the student step back and forward within the part already heard", async () => {
    await setup();
    expect(
      screen.queryByRole("button", { name: /back 1 line/i }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /forward 1 line/i }),
    ).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /start/i }));
    // On line 1 with nothing heard yet: neither direction is available.
    expect(screen.getByRole("button", { name: /back 1 line/i })).toBeDisabled();
    expect(
      screen.getByRole("button", { name: /forward 1 line/i }),
    ).toBeDisabled();

    await endLine(1);
    await endLine(2);
    expect(FakeAudio.byLine(3).playCalls).toBe(1);
    // On line 3, the furthest line: back yes, forward no.
    expect(screen.getByRole("button", { name: /back 1 line/i })).toBeEnabled();
    expect(
      screen.getByRole("button", { name: /forward 1 line/i }),
    ).toBeDisabled();

    // Back to line 2; the bar follows the current position.
    fireEvent.click(screen.getByRole("button", { name: /back 1 line/i }));
    expect(FakeAudio.byLine(2).playCalls).toBe(2);
    expect(screen.getByTestId("recall-progress")).toHaveTextContent(
      "Line 2 of 3",
    );
    expect(screen.getByRole("progressbar")).toHaveAttribute(
      "aria-valuenow",
      "33",
    );
    expect(
      screen.getByRole("button", { name: /forward 1 line/i }),
    ).toBeEnabled();

    // Back again to line 1; the bar hits zero and Back is disabled.
    fireEvent.click(screen.getByRole("button", { name: /back 1 line/i }));
    expect(FakeAudio.byLine(1).playCalls).toBe(2);
    expect(screen.getByRole("progressbar")).toHaveAttribute(
      "aria-valuenow",
      "0",
    );
    expect(screen.getByRole("button", { name: /back 1 line/i })).toBeDisabled();

    // Forward twice returns to line 3, the furthest heard, then stops there.
    fireEvent.click(screen.getByRole("button", { name: /forward 1 line/i }));
    expect(FakeAudio.byLine(2).playCalls).toBe(3);
    fireEvent.click(screen.getByRole("button", { name: /forward 1 line/i }));
    expect(FakeAudio.byLine(3).playCalls).toBe(2);
    expect(screen.getByTestId("recall-progress")).toHaveTextContent(
      "Line 3 of 3",
    );
    expect(
      screen.getByRole("button", { name: /forward 1 line/i }),
    ).toBeDisabled();

    // Works while paused too, and resumes playback.
    fireEvent.click(screen.getByRole("button", { name: /pause audio/i }));
    fireEvent.click(screen.getByRole("button", { name: /back 1 line/i }));
    expect(FakeAudio.byLine(2).playCalls).toBe(4);
    expect(
      screen.getByRole("button", { name: /pause audio/i }),
    ).toBeInTheDocument();

    // The saved resume point is the furthest line heard, not the current one.
    expect(window.localStorage.getItem("recall-listened:1")).toBe("2");
  });

  it("remembers the furthest line heard and resumes there on reload", async () => {
    const { unmount } = await setup();
    fireEvent.click(screen.getByRole("button", { name: /start/i }));
    await endLine(1);
    await endLine(2);
    expect(window.localStorage.getItem("recall-listened:1")).toBe("2");
    unmount();

    FakeAudio.instances = [];
    await setup();
    expect(screen.getByTestId("recall-resume")).toHaveTextContent(
      "pick up from line 3",
    );
    expect(screen.getByRole("progressbar")).toHaveAttribute(
      "aria-valuenow",
      "67",
    );

    fireEvent.click(screen.getByRole("button", { name: /start/i }));
    expect(FakeAudio.byLine(1).playCalls).toBe(0);
    expect(FakeAudio.byLine(2).playCalls).toBe(0);
    expect(FakeAudio.byLine(3).playCalls).toBe(1);

    // Finishing clears the saved position.
    await endLine(3);
    expect(screen.getByTestId("recall-cards")).toBeInTheDocument();
    expect(window.localStorage.getItem("recall-listened:1")).toBeNull();
  });

  it("ignores a saved position that is out of range", async () => {
    window.localStorage.setItem("recall-listened:1", "99");
    await setup();
    expect(screen.queryByTestId("recall-resume")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /start/i }));
    expect(FakeAudio.byLine(1).playCalls).toBe(1);
  });

  it("shows the cards in server order as 3:4 boxes and does not move them", async () => {
    await setup();
    await listenThrough();

    expect(cardOrder()).toEqual([3, 1, 5, 2, 4]);
    expect(screen.getByTestId("recall-prompt")).toHaveTextContent(
      "Select the box that occurs first in the story.",
    );
    expect(
      screen.getByTestId("recall-cards").querySelectorAll("img"),
    ).toHaveLength(4);
    expect(screen.getByTestId("recall-card-3")).toHaveClass("aspect-[3/4]");
    expect(screen.getByText("לוש")).toHaveClass("target-word");
    expect(screen.getByText("לוש")).not.toHaveClass("underline");

    await pickCard(3);
    expect(cardOrder()).toEqual([3, 1, 5, 2, 4]);
  });

  it("marks wrong picks red, then clears them on a correct pick and plays audio", async () => {
    const { onCheckPick, onContinue } = await setup();
    await listenThrough();

    await pickCard(3);
    expect(onCheckPick).toHaveBeenCalledWith(3, 1);
    expect(screen.getByTestId("recall-card-3")).toHaveAttribute(
      "data-result",
      "wrong",
    );
    expect(screen.getByTestId("recall-card-3")).toHaveClass("border-red-400");
    expect(screen.getByTestId("recall-attempts")).toHaveTextContent("1");
    expect(screen.getByTestId("recall-prompt")).toHaveTextContent("first");

    await pickCard(1);
    expect(onCheckPick).toHaveBeenLastCalledWith(1, 1);
    expect(screen.getByTestId("recall-card-3")).toHaveAttribute(
      "data-result",
      "pending",
    );
    expect(screen.getByTestId("recall-card-1")).toHaveAttribute(
      "data-result",
      "correct",
    );
    expect(screen.getByTestId("recall-card-1")).toHaveClass("border-green-500");
    expect(screen.getByTestId("recall-card-1")).toBeDisabled();
    expect(FakeAudio.bySentence(1).playCalls).toBe(1);
    expect(screen.getByTestId("recall-prompt")).toHaveTextContent(
      "Select the box that occurs second in the story.",
    );

    await pickCard(2);
    await pickCard(3);
    await pickCard(4);
    await pickCard(5);
    expect(screen.getByText(/great job/i)).toBeInTheDocument();
    expect(screen.getByTestId("recall-card-5")).toHaveAttribute(
      "data-result",
      "correct",
    );
    expect(FakeAudio.bySentence(5).playCalls).toBe(1);
    expect(cardOrder()).toEqual([3, 1, 5, 2, 4]);

    fireEvent.click(screen.getByRole("button", { name: /continue to score/i }));
    expect(onContinue).toHaveBeenCalled();
  });

  it("reports a failed check and lets the student try again", async () => {
    const failing = vi.fn().mockRejectedValueOnce(new Error("boom"));
    await setup({}, failing);
    await listenThrough();

    await pickCard(1);
    expect(screen.getByRole("alert")).toHaveTextContent(/couldn't check/i);
    expect(screen.queryByTestId("recall-attempts")).not.toBeInTheDocument();
    expect(screen.getByTestId("recall-card-1")).toBeEnabled();
  });

  it("opens finished when the server says the phase is complete", async () => {
    await setup({ completed: true, attempts: 2 });
    expect(screen.queryByTestId("recall-listening")).not.toBeInTheDocument();
    expect(screen.getByTestId("recall-already-complete")).toBeInTheDocument();
    expect(screen.getByText(/great job/i)).toBeInTheDocument();
    expect(screen.getByTestId("recall-card-1")).toHaveAttribute(
      "data-result",
      "correct",
    );
    expect(screen.getByTestId("recall-card-1")).toBeDisabled();
  });

  it("plays chained sentence clips in order after a correct pick", async () => {
    const sentences = SENTENCES.map((s) =>
      s.id === 1 ? { ...s, audio_urls: ["sent-1a", "sent-1b"] } : s,
    );
    await setup({ audio_urls: {}, line_count: 0, sentences });

    await pickCard(1);
    expect(FakeAudio.instances.map((a) => a.src)).toContain("sent-1a");
    expect(FakeAudio.instances.map((a) => a.src)).not.toContain("sent-1b");

    await act(async () => {
      const first = FakeAudio.instances.find((a) => a.src === "sent-1a");
      first?.end();
      await Promise.resolve();
    });
    expect(FakeAudio.instances.map((a) => a.src)).toContain("sent-1b");
    expect(
      FakeAudio.instances.find((a) => a.src === "sent-1b")?.playCalls,
    ).toBe(1);
  });

  it("skips listening when there is no narration", async () => {
    await setup({ audio_urls: {}, line_count: 0 });
    expect(screen.queryByTestId("recall-listening")).not.toBeInTheDocument();
    expect(screen.getByTestId("recall-cards")).toBeInTheDocument();
    expect(screen.getByTestId("recall-prompt")).toHaveTextContent("first");
  });

  it("finishes after listening when the story has no recall sentences", async () => {
    await setup({ sentences: [] });
    await listenThrough();
    expect(screen.getByTestId("recall-no-sentences")).toBeInTheDocument();
    expect(screen.getByText(/great job/i)).toBeInTheDocument();
  });
});
