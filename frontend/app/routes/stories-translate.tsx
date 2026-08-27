import { StoriesTranslate } from "~/components/StoriesTranslate";
import { useTimeTracking } from "../lib/timeTracking";
import { useEffect } from "react";
import { pageMeta } from "~/lib/pageTitle";

export function meta() {
  return pageMeta("Translate", "Translate the story");
}

export default function VocabRoute() {
  const { startTracking } = useTimeTracking();

  useEffect(() => {
    startTracking();
  }, [startTracking]);

  return <StoriesTranslate />;
}
