import { StoriesVideo } from "../components/StoriesVideo";
import { useTimeTracking } from "../lib/timeTracking";
import { useEffect } from "react";

export function meta() {
  return [
    { title: "Glossias - Video" },
    { name: "description", content: "Watch the story video" },
  ];
}

export default function VideoRoute() {
  const { startTracking } = useTimeTracking();

  useEffect(() => {
    startTracking();
  }, [startTracking]);

  return <StoriesVideo />;
}
