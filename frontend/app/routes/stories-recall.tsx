import { StoriesRecall } from "../components/StoriesRecall";
import { useTimeTracking } from "../lib/timeTracking";
import { useEffect } from "react";

export default function RecallRoute() {
  const { startTracking } = useTimeTracking();

  useEffect(() => {
    startTracking();
  }, [startTracking]);

  return <StoriesRecall />;
}
