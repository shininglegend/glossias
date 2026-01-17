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
  status: "active" | "past" | "future";
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
  const [selectedUsers, setSelectedUsers] = React.useState<Set<string>>(
    new Set()
  );
  const [statusFilter, setStatusFilter] = React.useState<string>("all");
  const [updatingStatus, setUpdatingStatus] = React.useState<string | null>(null);
  const [showBulkStatusModal, setShowBulkStatusModal] = React.useState(false);
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

  const handleAddUser = async (emails: string[], courseId: number) => {
    setAdding(true);
    try {
      const res = await authenticatedFetch(
        `/api/admin/course-users/${courseId}`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ emails }),
        }
      );

      if (!res.ok) {
        const errorData = await res.json().catch(() => ({}));
        const errorMessage = errorData.error || "Failed to add users";
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
      setSelectedUsers((prev) => {
        const next = new Set(prev);
        next.delete(userId);
        return next;
      });
    } catch (error) {
      console.error("Failed to remove user:", error);
      const message =
        error instanceof Error ? error.message : "Failed to remove user";
      alert(message);
    } finally {
      setRemoving(null);
    }
  };

  const handleRemoveSelected = async () => {
    if (selectedUsers.size === 0 || !selectedCourse) return;

    if (
      !confirm(
        `Are you sure you want to remove ${selectedUsers.size} user(s) from the course?`
      )
    ) {
      return;
    }

    const userIds = Array.from(selectedUsers);
    const results = await Promise.allSettled(
      userIds.map((userId) =>
        authenticatedFetch(
          `/api/admin/course-users/${selectedCourse}/users/${userId}`,
          { method: "DELETE" }
        )
      )
    );

    const successful = results.filter((r) => r.status === "fulfilled").length;
    const failed = results.length - successful;

    if (failed > 0) {
      alert(
        `Removed ${successful} user(s). Failed to remove ${failed} user(s).`
      );
    }

    // Refresh users list
    const usersRes = await authenticatedFetch(
      `/api/admin/course-users/${selectedCourse}`,
      { headers: { Accept: "application/json" } }
    );
    if (usersRes.ok) {
      const json = await usersRes.json();
      setUsers(json.users);
    }
    setSelectedUsers(new Set());
  };

  const toggleUserSelection = (userId: string) => {
    setSelectedUsers((prev) => {
      const next = new Set(prev);
      if (next.has(userId)) {
        next.delete(userId);
      } else {
        next.add(userId);
      }
      return next;
    });
  };

  const selectAll = () => {
    setSelectedUsers(new Set(filteredUsers.map((u) => u.id)));
  };

  const selectNone = () => {
    setSelectedUsers(new Set());
  };

  const handleStatusChange = async (
    userId: string,
    newStatus: "active" | "past" | "future"
  ) => {
    if (!selectedCourse) return;

    setUpdatingStatus(userId);
    try {
      const res = await authenticatedFetch(
        `/api/admin/course-users/${selectedCourse}/status?status=${newStatus}`,
        {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ user_ids: [userId] }),
        }
      );

      if (!res.ok) {
        const errorData = await res.json().catch(() => ({}));
        throw new Error(errorData.error || "Failed to update status");
      }

      // Update local state
      setUsers((prev) =>
        prev.map((u) => (u.id === userId ? { ...u, status: newStatus } : u))
      );
    } catch (error) {
      console.error("Failed to update status:", error);
      const message =
        error instanceof Error ? error.message : "Failed to update status";
      alert(message);
    } finally {
      setUpdatingStatus(null);
    }
  };

  const handleBulkStatusChange = async (
    newStatus: "active" | "past" | "future"
  ) => {
    if (selectedUsers.size === 0 || !selectedCourse) return;

    setUpdatingStatus("bulk");
    try {
      const res = await authenticatedFetch(
        `/api/admin/course-users/${selectedCourse}/status?status=${newStatus}`,
        {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ user_ids: Array.from(selectedUsers) }),
        }
      );

      if (!res.ok) {
        const errorData = await res.json().catch(() => ({}));
        throw new Error(errorData.error || "Failed to update status");
      }

      // Update local state for all selected users
      setUsers((prev) =>
        prev.map((u) =>
          selectedUsers.has(u.id) ? { ...u, status: newStatus } : u
        )
      );
      setSelectedUsers(new Set());
      setShowBulkStatusModal(false);
    } catch (error) {
      console.error("Failed to update status:", error);
      const message =
        error instanceof Error ? error.message : "Failed to update status";
      alert(message);
    } finally {
      setUpdatingStatus(null);
    }
  };

  const filteredUsers = users
    .filter((user) => {
      if (statusFilter !== "all" && user.status !== statusFilter) {
        return false;
      }
      return true;
    })
    .sort((a, b) => {
      // Sort order: active (0), future (1), past (2)
      const statusOrder = { active: 0, future: 1, past: 2 };
      return statusOrder[a.status] - statusOrder[b.status];
    });

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
            Add Users to Course
          </Button>
        </div>

        <div className="flex items-center gap-4 text-xs text-slate-600">
          <span className="font-medium">Status:</span>
          <div className="flex items-center gap-1">
            <div className="w-4 h-4 rounded bg-green-100 border border-green-200"></div>
            <span>Active</span>
          </div>
          <div className="flex items-center gap-1">
            <div className="w-4 h-4 rounded bg-blue-100 border border-blue-200"></div>
            <span>Future</span>
          </div>
          <div className="flex items-center gap-1">
            <div className="w-4 h-4 rounded bg-gray-100 border border-gray-200"></div>
            <span>Past</span>
          </div>
        </div>

        <div className="flex items-center gap-4 flex-wrap">
          <div className="flex items-center gap-2">
            <label className="text-sm font-medium text-slate-700">
              Course:
            </label>
            <select
              value={selectedCourse || ""}
              onChange={(e) => setSelectedCourse(Number(e.target.value))}
              className="rounded-md border border-slate-300 bg-white px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            >
              {courses.map((course) => (
                <option key={course.course_id} value={course.course_id}>
                  {course.name} ({course.course_number})
                </option>
              ))}
            </select>
          </div>

          <div className="flex items-center gap-2">
            <label className="text-sm font-medium text-slate-700">
              Status:
            </label>
            <select
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value)}
              className="rounded-md border border-slate-300 bg-white px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            >
              <option value="all">All Statuses</option>
              <option value="active">Active</option>
              <option value="future">Future</option>
              <option value="past">Past</option>
            </select>
          </div>

          {filteredUsers.length > 0 && (
            <div className="flex items-center gap-2 ml-auto">
              {selectedUsers.size > 0 && (
                <>
                  <Button
                    onClick={() => setShowBulkStatusModal(true)}
                    variant="outline"
                    size="sm"
                    icon={<span className="material-icons text-sm">sync</span>}
                    disabled={updatingStatus !== null}
                  >
                    Change Status ({selectedUsers.size})
                  </Button>
                  <Button
                    onClick={handleRemoveSelected}
                    variant="outline"
                    size="sm"
                    icon={<span className="material-icons text-sm">delete</span>}
                    disabled={removing !== null}
                  >
                    Remove Selected ({selectedUsers.size})
                  </Button>
                </>
              )}
              <Button onClick={selectAll} variant="outline" size="sm">
                Select All
              </Button>
              <Button onClick={selectNone} variant="outline" size="sm">
                Select None
              </Button>
            </div>
          )}
        </div>

        <div className="grid gap-2">
          {filteredUsers.length === 0 ? (
            <Card className="p-8 text-center">
              <p className="text-slate-500">No users found for this course, or filters are hiding all users.</p>
            </Card>
          ) : (
            filteredUsers.map((user) => {
              const statusColors = {
                active: "!bg-green-50 !border-green-200",
                past: "!bg-gray-50 !border-gray-200",
                future: "!bg-blue-50 !border-blue-200",
              };
              return (
              <Card key={user.id} className={`p-3 ${statusColors[user.status]}`}>
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-4">
                    <input
                      type="checkbox"
                      checked={selectedUsers.has(user.id)}
                      onChange={() => toggleUserSelection(user.id)}
                      className="h-4 w-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500"
                    />
                    <div className="flex items-center gap-6">
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
                    <select
                      value={user.status}
                      onChange={(e) =>
                        handleStatusChange(
                          user.id,
                          e.target.value as "active" | "past" | "future"
                        )
                      }
                      disabled={updatingStatus === user.id}
                      className="rounded border border-slate-300 bg-white px-2 py-1 text-xs focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                    >
                      <option value="active">Active</option>
                      <option value="future">Future</option>
                      <option value="past">Past</option>
                    </select>
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
            );
            })
          )}
        </div>
      </div>

      {showAddForm && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4">
          <Card className="w-full max-w-md p-6">
            <h2 className="text-lg font-semibold mb-4">Add Users to Course</h2>
            <form
              onSubmit={(e) => {
                e.preventDefault();
                const formData = new FormData(e.currentTarget);
                const emailsText = formData.get("emails") as string;
                const courseId = Number(formData.get("courseId"));
                
                // Split by newlines, commas, or spaces and filter out empty strings
                const emails = emailsText
                  .split(/[\n,\s]+/)
                  .map(email => email.trim())
                  .filter(email => email.length > 0);

                // Basic email format validation before submitting to backend
                const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
                const invalidEmails = emails.filter(email => !emailRegex.test(email));
                if (invalidEmails.length > 0) {
                  alert(
                    `The following email address(es) are invalid:\n\n${invalidEmails.join(
                      "\n"
                    )}\n\nPlease correct them and try again.`
                  );
                  return;
                }
                
                if (emails.length > 0 && courseId) {
                  handleAddUser(emails, courseId);
                }
              }}
              className="space-y-4"
            >
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">
                  Email Addresses (one per line)
                </label>
                <textarea
                  name="emails"
                  required
                  rows={4}
                  className="w-full rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                  placeholder="user1@example.com&#10;user2@example.com&#10;user3@example.com"
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
                  {adding ? "Adding..." : "Add Users"}
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

      {showBulkStatusModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4">
          <Card className="w-full max-w-md p-6">
            <h2 className="text-lg font-semibold mb-4">
              Change Status for {selectedUsers.size} User{selectedUsers.size !== 1 ? 's' : ''}
            </h2>
            <div className="space-y-4">
              <p className="text-sm text-slate-600">
                Select the new status for all selected users:
              </p>
              <div className="flex gap-2">
                <Button
                  onClick={() => handleBulkStatusChange("active")}
                  className="flex-1"
                  disabled={updatingStatus !== null}
                >
                  Active
                </Button>
                <Button
                  onClick={() => handleBulkStatusChange("future")}
                  className="flex-1"
                  disabled={updatingStatus !== null}
                >
                  Future
                </Button>
                <Button
                  onClick={() => handleBulkStatusChange("past")}
                  className="flex-1"
                  disabled={updatingStatus !== null}
                >
                  Past
                </Button>
              </div>
              <Button
                type="button"
                variant="outline"
                onClick={() => setShowBulkStatusModal(false)}
                className="w-full"
                disabled={updatingStatus !== null}
              >
                Cancel
              </Button>
            </div>
          </Card>
        </div>
      )}
    </main>
  );
}
