/** Student phases in `defaultPageOrder` (stories-navigation.go). Vocab and
 *  Grammar remain as URLs but are not in the current flow. */
export const STORY_PHASES = [
  {
    icon: "play_circle",
    title: "Watch",
    body: "Start with a video of the story to get familiar with it before working through the text.",
  },
  {
    icon: "image_search",
    title: "Identify",
    body: "Follow along with the audio. When it pauses, pick the picture that matches the target word.",
  },
  {
    icon: "translate",
    title: "Translate",
    body: "Listen line by line and reveal the English translation for the lines you choose.",
  },
  {
    icon: "edit",
    title: "Produce",
    body: "Write what a highlighted passage means in English against the clock, then compare it with the story's English.",
  },
  {
    icon: "reorder",
    title: "Recall",
    body: "Listen once more, then put the story's sentences back in order.",
  },
  {
    icon: "assessment",
    title: "Score",
    body: "See your accuracy for each phase once the story is complete.",
  },
] as const;
