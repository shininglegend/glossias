import type { Route } from "./+types/home";
import { StoryList } from "../components/StoryList";

export function meta({}: Route.MetaArgs) {
  return [
    { title: "Glossias" },
    { name: "description", content: "Select a story to begin reading" },
  ];
}

export default function Home() {
  return <StoryList />;
}
