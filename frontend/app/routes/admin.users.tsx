import React from "react";
import { Link } from "react-router";
import Button from "~/components/ui/Button";
import { Card } from "~/components/ui/Card";
import Badge from "~/components/ui/Badge";
import { useAuthenticatedFetch } from "../lib/authFetch";
import {
  useCoursesApi,
  type Course as CourseType,
} from "../services/coursesApi";

type User = {
  id: string;
  email: string;
  name: string;
  role: "student" | "course_admin" | "super_admin";
  enrolled_at: string;
};

export default function AdminUsers() {
  const [users, setUsers] = React.useState<User[]>([]);
  const [courses, setCourses] = React.useState<CourseType[]>([]);
  const [selectedCourse, setSelectedCourse] = React.useState<number | null>(
    null
  );
  const [loading, setLoading] = React.useState(true);
  const [showAddForm, setShowAddForm] = React.useState(false);
  const [adding, setAdding] = React.useState(false);
  const [removing, setRemoving] = React.useState<string | null>(null);
  const authenticatedFetch = useAuthenticatedFetch();
  const coursesApi = useCoursesApi();

  React.useEffect(() => {
    async function fetchCourses() {
      try {
        const response = await coursesApi.getCourses();
        setCourses(response.courses);
        if (response.courses.length > 0) {
          setSelectedCourse(response.courses[0].course_id);
        }
      } catch (error) {
        console.error("Failed to fetch courses:", error);
      } finally {
        setLoading(false);
      }
    }
    fetchCourses();
  }, []);

  React.useEffect(() => {
    if (selectedCourse === null) return;

    async function fetchUsers() {
      try {
        const res = await authenticatedFetch(
          `/api/admin/course-users/${selectedCourse}`,
          {
            headers: { Accept: "application/json" },
          }
        );
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const json = await res.json();
        setUsers(json.users);
      } catch (error) {
        console.error("Failed to fetch users:", error);
      }
    }
    fetchUsers();
  }, [selectedCourse, authenticatedFetch]);

  const handleAddUser = async (email: string, courseId: number) => {
    setAdding(true);
    try {
      const res = await authenticatedFetch(
        `/api/admin/course-users/${courseId}`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ email }),
        }
      );

      if (!res.ok) {
        const errorData = await res.json().catch(() => ({}));
        let errorMessage = "Failed to add user";

        if (res.status === 404) {
          errorMessage =
            "User with this email address not found. Make sure they have signed up first.";
        } else if (res.status === 409) {
          errorMessage = "User is already enrolled in this course.";
        } else if (errorData.error) {
          errorMessage = errorData.error;
        }

        throw new Error(errorMessage);
      }

      // Refresh users list
      const usersRes = await authenticatedFetch(
        `/api/admin/course-users/${courseId}`,
        {
          headers: { Accept: "application/json" },
        }
      );
      if (usersRes.ok) {
        const json = await usersRes.json();
        setUsers(json.users);
      }

      setShowAddForm(false);
    } catch (error) {
      console.error("Failed to add user:", error);
      const message =
        error instanceof Error ? error.message : "Failed to add user";
      alert(message);
    } finally {
      setAdding(false);
    }
  };

  const handleRemoveUser = async (userId: string, courseId: number) => {
    if (
      !confirm("Are you sure you want to remove this user from the course?")
    ) {
      return;
    }

    setRemoving(userId);
    try {
      const res = await authenticatedFetch(
        `/api/admin/course-users/${courseId}/users/${userId}`,
        {
          method: "DELETE",
        }
      );

      if (!res.ok) {
        const errorData = await res.json().catch(() => ({}));
        let errorMessage = "Failed to remove user";

        if (res.status === 404) {
          errorMessage = "User not found or not enrolled in this course.";
        } else if (errorData.error) {
          errorMessage = errorData.error;
        }

        throw new Error(errorMessage);
      }

      // Remove user from local state
      setUsers((prev) => prev.filter((user) => user.id !== userId));
    } catch (error) {
      console.error("Failed to remove user:", error);
      const message =
        error instanceof Error ? error.message : "Failed to remove user";
      alert(message);
    } finally {
      setRemoving(null);
    }
  };

  const filteredUsers = users;

  if (loading) {
    return (
      <main className="mx-auto max-w-6xl px-4 py-6">
        <div className="text-center py-8">Loading users...</div>
      </main>
    );
  }

  return (
    <main className="mx-auto max-w-6xl px-4 py-6">
      <div className="flex flex-col gap-6">
        <div className="flex items-end justify-between gap-4">
          <div>
            <h1 className="text-3xl font-semibold tracking-tight">
              User Management
            </h1>
            <p className="text-sm text-slate-500">
              Manage users by course enrollment
            </p>
          </div>
          <Button
            onClick={() => setShowAddForm(true)}
            icon={<span className="material-icons text-base">person_add</span>}
          >
            Add User to Course
          </Button>
        </div>

        <div className="flex items-center gap-4">
          <label className="text-sm font-medium text-slate-700">Course:</label>
          <select
            value={selectedCourse || ""}
            onChange={(e) => setSelectedCourse(Number(e.target.value))}
            className="rounded-md border border-slate-300 bg-white px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
          >
            <option value="">All Courses</option>
            {courses.map((course) => (
              <option key={course.course_id} value={course.course_id}>
                {course.name} ({course.course_number})
              </option>
            ))}
          </select>
        </div>

        <div className="grid gap-4">
          {filteredUsers.length === 0 ? (
            <Card className="p-8 text-center">
              <p className="text-slate-500">No users found for this course.</p>
            </Card>
          ) : (
            filteredUsers.map((user) => (
              <Card key={user.id} className="p-4">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-4">
                    <div>
                      <h3 className="font-medium text-slate-900">
                        {user.name}
                      </h3>
                      <p className="text-sm text-slate-500">{user.email}</p>
                      <p className="text-xs text-slate-400">
                        Enrolled:{" "}
                        {new Date(user.enrolled_at).toLocaleDateString()}
                      </p>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <Badge
                      variant={
                        user.role === "super_admin"
                          ? "danger"
                          : user.role === "course_admin"
                            ? "warning"
                            : "default"
                      }
                    >
                      {user.role.replace("_", " ")}
                    </Badge>
                    {selectedCourse && (
                      <Button
                        onClick={() =>
                          handleRemoveUser(user.id, selectedCourse)
                        }
                        variant="outline"
                        size="sm"
                        icon={
                          <span className="material-icons text-sm">remove</span>
                        }
                        disabled={removing === user.id}
                      >
                        {removing === user.id ? "Removing..." : "Remove"}
                      </Button>
                    )}
                  </div>
                </div>
              </Card>
            ))
          )}
        </div>
      </div>

      {showAddForm && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4">
          <Card className="w-full max-w-md p-6">
            <h2 className="text-lg font-semibold mb-4">Add User to Course</h2>
            <form
              onSubmit={(e) => {
                e.preventDefault();
                const formData = new FormData(e.currentTarget);
                const email = formData.get("email") as string;
                const courseId = Number(formData.get("courseId"));
                if (email && courseId) {
                  handleAddUser(email, courseId);
                }
              }}
              className="space-y-4"
            >
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">
                  Email Address
                </label>
                <input
                  type="email"
                  name="email"
                  required
                  className="w-full rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                  placeholder="user@example.com"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">
                  Course
                </label>
                <select
                  name="courseId"
                  required
                  className="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                >
                  <option value="">Select a course</option>
                  {courses.map((course) => (
                    <option key={course.course_id} value={course.course_id}>
                      {course.name} ({course.course_number})
                    </option>
                  ))}
                </select>
              </div>
              <div className="flex gap-2 pt-2">
                <Button type="submit" className="flex-1" disabled={adding}>
                  {adding ? "Adding..." : "Add User"}
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => setShowAddForm(false)}
                  className="flex-1"
                  disabled={adding}
                >
                  Cancel
                </Button>
              </div>
            </form>
          </Card>
        </div>
      )}
    </main>
  );
}
