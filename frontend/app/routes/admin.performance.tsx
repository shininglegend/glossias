import { Link } from "react-router";
import { useUserContext, isUserAdminOfCourses } from "../contexts/UserContext";
import { Card } from "../components/ui/Card";
import Badge from "../components/ui/Badge";

export default function AdminPerformance() {
  const { userInfo, loading } = useUserContext();

  if (loading) {
    return (
      <main className="mx-auto max-w-6xl px-4 py-6">
        <div className="text-center py-8">Loading...</div>
      </main>
    );
  }

  if (!userInfo || !isUserAdminOfCourses(userInfo)) {
    return (
      <main className="mx-auto max-w-6xl px-4 py-6">
        <div className="text-center py-8">
          <h1 className="text-2xl font-semibold text-red-600 mb-2">
            Access Denied
          </h1>
          <p className="text-slate-600">
            Course admin privileges required to view student performance.
          </p>
          <Link
            to="/admin"
            className="text-blue-600 hover:underline mt-4 inline-block"
          >
            ← Back to Admin Dashboard
          </Link>
        </div>
      </main>
    );
  }

  const courses = userInfo.course_admin_rights || [];

  return (
    <main className="mx-auto max-w-6xl px-4 py-6">
      <div className="flex flex-col gap-6">
        <div>
          <h1 className="text-3xl font-semibold tracking-tight">
            Student Performance
          </h1>
          <p className="text-sm text-slate-500">
            Select a course to view student performance data
          </p>
        </div>

        {courses.length === 0 ? (
          <div className="text-center py-8 text-slate-500">
            No courses assigned. Contact a super admin to be assigned to a course.
          </div>
        ) : (
          <div className="grid gap-4">
            {courses.map((course) => (
              <Link
                key={course.course_id}
                to={`/admin/courses/${course.course_id}/students`}
              >
                <Card className="p-4 hover:bg-slate-50 transition-colors cursor-pointer">
                  <div className="flex items-center justify-between">
                    <div>
                      <div className="flex items-center gap-2 mb-1">
                        <h3 className="font-semibold text-lg">
                          {course.course_name}
                        </h3>
                        <Badge>{course.course_number}</Badge>
                      </div>
                      <div className="text-xs text-slate-500">
                        Assigned: {new Date(course.assigned_at).toLocaleDateString()}
                      </div>
                    </div>
                    <span className="material-icons text-slate-400">
                      arrow_forward
                    </span>
                  </div>
                </Card>
              </Link>
            ))}
          </div>
        )}
      </div>
    </main>
  );
}
