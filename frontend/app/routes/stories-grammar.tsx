import { StoriesGrammar } from "../components/StoriesGrammar";
import { useTimeTracking } from "../lib/timeTracking";
import { useEffect } from "react";
import { pageMeta } from "~/lib/pageTitle";

export function meta() {
  return pageMeta("Grammar", "Practice the story's grammar");
}

export default function VocabRoute() {
  const { startTracking } = useTimeTracking();

  useEffect(() => {
    startTracking();
  }, [startTracking]);

  return <StoriesGrammar />;
}
