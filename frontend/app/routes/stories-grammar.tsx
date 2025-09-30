import { StoriesGrammar } from "../components/StoriesGrammar";
import { useTimeTracking } from "../lib/timeTracking";
import { useEffect } from "react";

export default function VocabRoute() {
  const { startTracking } = useTimeTracking();

  useEffect(() => {
    startTracking();
  }, [startTracking]);

  return <StoriesGrammar />;
}
