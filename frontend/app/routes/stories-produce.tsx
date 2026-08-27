import { StoriesProduce } from "../components/StoriesProduce";
import { useTimeTracking } from "../lib/timeTracking";
import { useEffect } from "react";
import { pageMeta } from "~/lib/pageTitle";

export function meta() {
  return pageMeta("Produce", "Produce the story's lines");
}

export default function ProduceRoute() {
  const { startTracking } = useTimeTracking();

  useEffect(() => {
    startTracking();
  }, [startTracking]);

  return <StoriesProduce />;
}
