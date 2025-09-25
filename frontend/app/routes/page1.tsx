import type { Route } from "./+types/page1";
import { Page1 } from "../components/Page1";

export function meta({}: Route.MetaArgs) {
  return [
    { title: "Glossias - Page 1" },
    { name: "description", content: "Listen to the story" },
  ];
}

export default function Page1Route() {
  return <Page1 />;
}
