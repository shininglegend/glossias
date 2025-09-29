import type { Route } from "./+types/home";
import { SignedIn, SignedOut } from "@clerk/react-router";
import { StoryList } from "../components/StoryList";
import { LandingPage } from "../components/LandingPage";
import { Footer } from "~/components/Footer";

export function meta({}: Route.MetaArgs) {
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
        <SignedOut>
          <LandingPage />
        </SignedOut>
        <SignedIn>
          <StoryList />
        </SignedIn>
      </div>
      <Footer />
    </div>
  );
}
