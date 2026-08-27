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
}

// ---- Fixtures -------------------------------------------------------------

/** Correct story order is 1,2,3,4,5; the server hands them out shuffled. */
const SENTENCES = [
  { id: 3, hebrew_text: "שלוש", image_url: "img-3" },
  { id: 1, hebrew_text: "אחת", image_url: "img-1" },
  { id: 5, hebrew_text: "חמש" },
  { id: 2, hebrew_text: "שתיים", image_url: "img-2" },
  { id: 4, hebrew_text: "ארבע", image_url: "img-4" },
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

/** Grades against the true order 1..5. */
const gradeLocally = () =>
  vi.fn(async (ids: number[]) => {
    const results = ids.map((id, i) => id === i + 1);
    return { results, all_correct: results.every(Boolean) };
  });

const setup = async (
  pageOverrides: Partial<RecallData> = {},
  onCheckOrder = gradeLocally(),
) => {
  const onContinue = vi.fn();
  const utils = render(
    <RecallSession
      pageData={makePageData(pageOverrides)}
      nextStepName="Score"
      onCheckOrder={onCheckOrder}
      onContinue={onContinue}
    />,
  );
  await act(async () => {
    FakeAudio.instances.forEach((a) => a.ready());
    await Promise.resolve();
  });
  return { ...utils, onCheckOrder, onContinue };
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
    .map((li) => Number(li.dataset.testid?.replace("recall-card-", "")));

const moveUp = (position: number) =>
  fireEvent.click(
    screen.getByRole("button", { name: `Move sentence ${position} up` }),
  );
const moveDown = (position: number) =>
  fireEvent.click(
    screen.getByRole("button", { name: `Move sentence ${position} down` }),
  );

const submit = async () => {
  await act(async () => {
    fireEvent.click(screen.getByRole("button", { name: /check order/i }));
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

  it("replays the previous line on Back 1 line and carries on from there", async () => {
    await setup();
    expect(
      screen.queryByRole("button", { name: /back 1 line/i }),
    ).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /start/i }));
    await endLine(1);
    await endLine(2);
    expect(FakeAudio.byLine(3).playCalls).toBe(1);

    // On line 3 → go back to line 2, then continue with 3 again.
    fireEvent.click(screen.getByRole("button", { name: /back 1 line/i }));
    expect(FakeAudio.byLine(2).playCalls).toBe(2);
    expect(screen.getByTestId("recall-progress")).toHaveTextContent(
      "Line 2 of 3",
    );
    await endLine(2);
    expect(FakeAudio.byLine(3).playCalls).toBe(2);

    // Works while paused too, and resumes playback.
    fireEvent.click(screen.getByRole("button", { name: /pause audio/i }));
    fireEvent.click(screen.getByRole("button", { name: /back 1 line/i }));
    expect(FakeAudio.byLine(2).playCalls).toBe(3);
    expect(
      screen.getByRole("button", { name: /pause audio/i }),
    ).toBeInTheDocument();

    // The bar follows the current position back.
    expect(screen.getByRole("progressbar")).toHaveAttribute(
      "aria-valuenow",
      "33",
    );
    expect(screen.getByTestId("recall-progress")).toHaveTextContent(
      "Line 2 of 3",
    );

    // Stepping back again reaches line 1 and the bar goes to zero; on line 1
    // it just restarts line 1.
    fireEvent.click(screen.getByRole("button", { name: /back 1 line/i }));
    expect(FakeAudio.byLine(1).playCalls).toBe(2);
    expect(screen.getByRole("progressbar")).toHaveAttribute(
      "aria-valuenow",
      "0",
    );
    fireEvent.click(screen.getByRole("button", { name: /back 1 line/i }));
    expect(FakeAudio.byLine(1).playCalls).toBe(3);
    expect(screen.getByTestId("recall-progress")).toHaveTextContent(
      "Line 1 of 3",
    );

    // Playing forward again moves it up; the saved resume point is unchanged.
    await endLine(1);
    expect(screen.getByRole("progressbar")).toHaveAttribute(
      "aria-valuenow",
      "33",
    );
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

  it("shows the cards in server order and lets the student reorder them", async () => {
    await setup();
    await listenThrough();

    expect(cardOrder()).toEqual([3, 1, 5, 2, 4]);
    // Four of the five cards have a picture (decorative, so not role="img").
    expect(
      screen.getByTestId("recall-cards").querySelectorAll("img"),
    ).toHaveLength(4);

    moveUp(2); // 1 to the top
    expect(cardOrder()).toEqual([1, 3, 5, 2, 4]);
    moveDown(2); // 3 down one
    expect(cardOrder()).toEqual([1, 5, 3, 2, 4]);

    // Bounds are enforced.
    expect(
      screen.getByRole("button", { name: "Move sentence 1 up" }),
    ).toBeDisabled();
    expect(
      screen.getByRole("button", { name: "Move sentence 5 down" }),
    ).toBeDisabled();
  });

  it("marks wrong positions, counts the attempt, and completes on a correct order", async () => {
    const { onCheckOrder, onContinue } = await setup();
    await listenThrough();

    // First attempt: server order, only nothing is right.
    await submit();
    expect(onCheckOrder).toHaveBeenCalledWith([3, 1, 5, 2, 4]);
    expect(screen.getByTestId("recall-feedback")).toHaveTextContent(
      "0 of 5 in the right place",
    );
    expect(screen.getByTestId("recall-attempts")).toHaveTextContent("1");
    expect(screen.getByTestId("recall-card-3")).toHaveAttribute(
      "data-result",
      "wrong",
    );
    expect(screen.queryByText(/great job/i)).not.toBeInTheDocument();

    // Fix it: 3,1,5,2,4 → 1,2,3,4,5
    moveUp(2); // 1,3,5,2,4
    moveUp(4); // 1,3,2,5,4
    moveUp(3); // 1,2,3,5,4
    moveUp(5); // 1,2,3,4,5
    expect(cardOrder()).toEqual([1, 2, 3, 4, 5]);
    // Moving a card clears the stale markers.
    expect(screen.queryByTestId("recall-feedback")).not.toBeInTheDocument();

    await submit();
    expect(onCheckOrder).toHaveBeenLastCalledWith([1, 2, 3, 4, 5]);
    expect(screen.getByText(/great job/i)).toBeInTheDocument();
    expect(screen.getByTestId("recall-card-1")).toHaveAttribute(
      "data-result",
      "correct",
    );
    // Locked: no more reordering or checking.
    expect(
      screen.queryByRole("button", { name: /check order/i }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /move sentence/i }),
    ).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /continue to score/i }));
    expect(onContinue).toHaveBeenCalled();
  });

  it("reports a failed check and lets the student try again", async () => {
    const failing = vi.fn().mockRejectedValueOnce(new Error("boom"));
    await setup({}, failing);
    await listenThrough();

    await submit();
    expect(screen.getByRole("alert")).toHaveTextContent(/couldn't check/i);
    expect(screen.queryByTestId("recall-attempts")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: /check order/i })).toBeEnabled();
  });

  it("opens finished when the server says the phase is complete", async () => {
    await setup({ completed: true, attempts: 2 });
    expect(screen.queryByTestId("recall-listening")).not.toBeInTheDocument();
    expect(screen.getByTestId("recall-already-complete")).toBeInTheDocument();
    expect(screen.getByText(/great job/i)).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /check order/i }),
    ).not.toBeInTheDocument();
  });

  it("skips listening when there is no narration", async () => {
    await setup({ audio_urls: {}, line_count: 0 });
    expect(screen.queryByTestId("recall-listening")).not.toBeInTheDocument();
    expect(screen.getByTestId("recall-cards")).toBeInTheDocument();
  });

  it("finishes after listening when the story has no recall sentences", async () => {
    await setup({ sentences: [] });
    await listenThrough();
    expect(screen.getByTestId("recall-no-sentences")).toBeInTheDocument();
    expect(screen.getByText(/great job/i)).toBeInTheDocument();
  });
});
