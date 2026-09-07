import { Show } from "@clerk/react-router";
import { StoryList } from "../components/StoryList";
import { LandingPage } from "../components/LandingPage";
import { Footer } from "~/components/Footer";

export function meta() {
  return [
    { title: "Interactive Language Learning | Glossias" },
    {
      name: "description",
      content:
        "Learn languages through interactive stories with video, identification, translation, writing, and recall exercises",
    },
  ];
}

export default function Home() {
  return (
    // Full-bleed: cancels the padded, centred `container` in root.tsx so the
    // landing sections and footer span the viewport. `50% - 50vw` shifts the
    // box from the container's centre to the viewport edge.
    <div className="-my-4 mx-[calc(50%-50vw)] flex-1 flex flex-col">
      <div className="flex-1 flex flex-col">
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
