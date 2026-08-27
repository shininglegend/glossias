import { StoriesRecall } from "../components/StoriesRecall";
import { useTimeTracking } from "../lib/timeTracking";
import { useEffect } from "react";
import { pageMeta } from "~/lib/pageTitle";

export function meta() {
  return pageMeta("Recall", "Recall the story");
}

export default function RecallRoute() {
  const { startTracking } = useTimeTracking();

  useEffect(() => {
    startTracking();
  }, [startTracking]);

  return <StoriesRecall />;
}
