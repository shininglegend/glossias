import { Show } from "@clerk/react-router";
import { StoryList } from "../components/StoryList";
import { LandingPage } from "../components/LandingPage";
import { Footer } from "~/components/Footer";

export function meta() {
  return [
    { title: "Glossias - Interactive Language Learning" },
    {
      name: "description",
      content:
        "Learn languages through interactive stories with audio, vocabulary, and grammar support",
    },
  ];
}

export default function Home() {
  return (
    <div className="min-h-screen flex flex-col">
      <div className="flex-1">
        <Show when="signed-out">
          <LandingPage />
        </Show>
        <Show when="signed-in">
          <StoryList />
        </Show>
      </div>
      <Footer />
    </div>
  );
}
