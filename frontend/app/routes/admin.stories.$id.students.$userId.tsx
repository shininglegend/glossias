import { StudentStoryDrilldown } from "../components/StudentStoryDrilldown";
import { pageMeta } from "~/lib/pageTitle";

export function meta() {
  return pageMeta("Student Detail");
}

export default function AdminStoryStudentDrilldown() {
  return <StudentStoryDrilldown />;
}
