import { StoriesIdentify } from "../components/StoriesIdentify";
import { useTimeTracking } from "../lib/timeTracking";
import { useEffect } from "react";
import { pageMeta } from "~/lib/pageTitle";

export function meta() {
  return pageMeta("Identify", "Identify vocabulary in the story");
}

export default function IdentifyRoute() {
  const { startTracking } = useTimeTracking();

  useEffect(() => {
    startTracking();
  }, [startTracking]);

  return <StoriesIdentify />;
}
