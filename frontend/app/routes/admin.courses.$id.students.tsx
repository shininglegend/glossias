import { CourseStudentPerformance } from "../components/CourseStudentPerformance";
import { pageMeta } from "~/lib/pageTitle";

export function meta() {
  return pageMeta("Course Students");
}

export default function AdminCourseStudents() {
  return <CourseStudentPerformance />;
}
