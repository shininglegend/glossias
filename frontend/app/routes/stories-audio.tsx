import type { Route } from "./+types/stories-audio";
import { StoriesAudio } from "../components/StoriesAudio";

export function meta({}: Route.MetaArgs) {
  return [
    { title: "Glossias - Audio" },
    { name: "description", content: "Listen to the story" },
  ];
}

export default function AudioRoute() {
  return <StoriesAudio />;
}
