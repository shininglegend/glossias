import { StoriesIdentify } from "../components/StoriesIdentify";
import { useTimeTracking } from "../lib/timeTracking";
import { useEffect } from "react";

export default function IdentifyRoute() {
  const { startTracking } = useTimeTracking();

  useEffect(() => {
    startTracking();
  }, [startTracking]);

  return <StoriesIdentify />;
}
