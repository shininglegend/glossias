import { StoriesScore } from "../components/StoriesScore";
import { pageMeta } from "~/lib/pageTitle";

export function meta() {
  return pageMeta("Score", "Your score for this story");
}

export default function ScoreRoute() {
  return <StoriesScore />;
}
