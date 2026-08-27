import { StoriesProduce } from "../components/StoriesProduce";
import { useTimeTracking } from "../lib/timeTracking";
import { useEffect } from "react";

export default function ProduceRoute() {
  const { startTracking } = useTimeTracking();

  useEffect(() => {
    startTracking();
  }, [startTracking]);

  return <StoriesProduce />;
}
