/** Student phases in `defaultPageOrder` (stories-navigation.go). Vocab and
 *  Grammar remain as URLs but are not in the current flow. Theme tokens match
 *  the Score page (TimeCell / PhaseCard icon+title colours). */

export type PhaseTheme = {
  icon: string;
  title: string;
  bar: string;
  barMuted: string;
  ring: string;
};

export const PHASE_THEME = {
  intro: {
    icon: "text-indigo-600",
    title: "text-indigo-900",
    bar: "bg-indigo-600",
    barMuted: "bg-indigo-100",
    ring: "ring-indigo-500",
  },
  video: {
    icon: "text-red-600",
    title: "text-red-900",
    bar: "bg-red-600",
    barMuted: "bg-red-100",
    ring: "ring-red-500",
  },
  identify: {
    icon: "text-primary-600",
    title: "text-primary-900",
    bar: "bg-primary-600",
    barMuted: "bg-primary-100",
    ring: "ring-primary-500",
  },
  translate: {
    icon: "text-secondary-500",
    title: "text-secondary-800",
    bar: "bg-secondary-500",
    barMuted: "bg-secondary-100",
    ring: "ring-secondary-500",
  },
  produce: {
    icon: "text-teal-600",
    title: "text-teal-900",
    bar: "bg-teal-600",
    barMuted: "bg-teal-100",
    ring: "ring-teal-500",
  },
  recall: {
    icon: "text-orange-600",
    title: "text-orange-900",
    bar: "bg-orange-600",
    barMuted: "bg-orange-100",
    ring: "ring-orange-500",
  },
  score: {
    icon: "text-green-600",
    title: "text-green-900",
    bar: "bg-green-600",
    barMuted: "bg-green-100",
    ring: "ring-green-500",
  },
} as const satisfies Record<string, PhaseTheme>;

export const STORY_PHASES = [
  {
    path: "video",
    icon: "play_circle",
    title: "Watch",
    body: "Start with a video of the story to get familiar with it before working through the text.",
    theme: PHASE_THEME.video,
  },
  {
    path: "identify",
    icon: "image_search",
    title: "Identify",
    body: "Follow along with the audio. When it pauses, pick the picture that matches the target word.",
    theme: PHASE_THEME.identify,
  },
  {
    path: "translate",
    icon: "translate",
    title: "Translate",
    body: "Listen line by line and reveal the English translation for the lines you choose.",
    theme: PHASE_THEME.translate,
  },
  {
    path: "produce",
    icon: "edit",
    title: "Produce",
    body: "Write what a highlighted passage means in English against the clock, then compare it with the story's English.",
    theme: PHASE_THEME.produce,
  },
  {
    path: "recall",
    icon: "reorder",
    title: "Recall",
    body: "Listen once more, then put the story's sentences back in order.",
    theme: PHASE_THEME.recall,
  },
  {
    path: "score",
    icon: "assessment",
    title: "Score",
    body: "See your accuracy for each phase once the story is complete.",
    theme: PHASE_THEME.score,
  },
] as const;

/** Bar-only preface before Watch. Same `/video` URL; `?watch=1` is Watch. */
export const STORY_INTRO_PHASE = {
  path: "intro",
  icon: "auto_stories",
  title: "Intro",
  theme: PHASE_THEME.intro,
} as const;

export const STORY_BAR_PHASES = [STORY_INTRO_PHASE, ...STORY_PHASES] as const;

export type StoryPhasePath = (typeof STORY_BAR_PHASES)[number]["path"];

export function phaseByPath(path: string) {
  return STORY_BAR_PHASES.find((phase) => phase.path === path);
}

export function phaseHref(storyId: string, path: StoryPhasePath): string {
  if (path === "intro") return `/stories/${storyId}/video`;
  if (path === "video") return `/stories/${storyId}/video?watch=1`;
  return `/stories/${storyId}/${path}`;
}
