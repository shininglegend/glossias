import { Link } from "react-router";
import React from "react";
import Button from "~/components/ui/Button";
import Input from "~/components/ui/Input";
import { Card } from "~/components/ui/Card";
import Badge from "~/components/ui/Badge";
import Textarea from "~/components/ui/Textarea";

import {
  useCoursesApi,
  type Course,
  type CourseAdmin,
} from "../services/coursesApi";
import { useUserContext } from "../contexts/UserContext";

export default function AdminCourses() {
  const coursesApi = useCoursesApi();
  const { userInfo } = useUserContext();
  const [courses, setCourses] = React.useState<Course[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [processing, setProcessing] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  // Form states
  const [showCreateForm, setShowCreateForm] = React.useState(false);
  const [editingCourse, setEditingCourse] = React.useState<Course | null>(null);
  const [managingAdmins, setManagingAdmins] = React.useState<Course | null>(
    null,
  );
  const [courseAdmins, setCourseAdmins] = React.useState<CourseAdmin[]>([]);

  // Form data
  const [formData, setFormData] = React.useState({
    course_number: "",
    name: "",
    description: "",
  });
  const [adminUserId, setAdminUserId] = React.useState("");

  // Check if user is super admin
  const isSuperAdmin = userInfo?.is_super_admin || false;

  React.useEffect(() => {
    fetchCourses();
  }, []);

  const fetchCourses = async () => {
    try {
      setLoading(true);
      const response = await coursesApi.getCourses();
      setCourses(response.courses || []);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to fetch courses");
      console.error("Failed to fetch courses:", err);
    } finally {
      setLoading(false);
    }
  };

  const resetForm = () => {
    setFormData({ course_number: "", name: "", description: "" });
    setShowCreateForm(false);
    setEditingCourse(null);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!isSuperAdmin) return;

    setProcessing(true);
    try {
      if (editingCourse) {
        const response = await coursesApi.updateCourse(
          editingCourse.course_id,
          formData,
        );
        setCourses((prev) =>
          prev.map((c) =>
            c.course_id === editingCourse.course_id ? response.course : c,
          ),
        );
      } else {
        const response = await coursesApi.createCourse(formData);
        setCourses((prev) => [...prev, response.course]);
      }
      resetForm();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to save course");
    } finally {
      setProcessing(false);
    }
  };

  const handleEdit = (course: Course) => {
    setFormData({
      course_number: course.course_number,
      name: course.name,
      description: course.description || "",
    });
    setEditingCourse(course);
    setShowCreateForm(true);
  };

  const handleDelete = async (course: Course) => {
    if (!isSuperAdmin) return;
    if (!confirm(`Delete course "${course.name}"? This cannot be undone.`))
      return;

    setProcessing(true);
    try {
      await coursesApi.deleteCourse(course.course_id);
      setCourses((prev) =>
        prev.filter((c) => c.course_id !== course.course_id),
      );
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to delete course");
    } finally {
      setProcessing(false);
    }
  };

  const fetchCourseAdmins = async (course: Course) => {
    if (!course?.course_id) {
      setError("Course ID is missing");
      return;
    }
    try {
      const response = await coursesApi.getCourseAdmins(course.course_id);
      setCourseAdmins(response.admins);
      setManagingAdmins(course);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to fetch course admins",
      );
    }
  };

  const handleAddAdmin = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!managingAdmins || !adminUserId.trim()) return;

    setProcessing(true);
    try {
      await coursesApi.addCourseAdmin(managingAdmins.course_id, {
        user_id: adminUserId.trim(),
      });
      await fetchCourseAdmins(managingAdmins); // Refresh the list
      setAdminUserId("");
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to add course admin",
      );
    } finally {
      setProcessing(false);
    }
  };

  const handleRemoveAdmin = async (userId: string) => {
    if (!managingAdmins) return;
    if (!confirm("Remove this admin from the course?")) return;

    setProcessing(true);
    try {
      await coursesApi.removeCourseAdmin(managingAdmins.course_id, userId);
      setCourseAdmins((prev) => prev.filter((a) => a.user_id !== userId));
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to remove course admin",
      );
    } finally {
      setProcessing(false);
    }
  };

  if (!userInfo) {
    return <div className="text-center py-8">Loading user permissions...</div>;
  }

  if (!isSuperAdmin) {
    return (
      <main className="mx-auto max-w-6xl px-4 py-6">
        <div className="text-center py-8">
          <h1 className="text-2xl font-semibold text-red-600 mb-2">
            Access Denied
          </h1>
          <p className="text-slate-600">
            Super admin privileges required to manage courses.
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

  return (
    <main className="mx-auto max-w-6xl px-4 py-6">
      <div className="flex flex-col gap-6">
        <div className="flex items-end justify-between gap-4">
          <div>
            <h1 className="text-3xl font-semibold tracking-tight">
              Course Management
            </h1>
            <p className="text-sm text-slate-500">
              Manage courses and assign course administrators
            </p>
          </div>
          <div className="flex gap-2">
            <Link to="/admin">
              <Button variant="outline">← Back to Dashboard</Button>
            </Link>
            <Button
              onClick={() => setShowCreateForm(true)}
              icon={<span className="material-icons text-base">add</span>}
            >
              Add Course
            </Button>
          </div>
        </div>

        {error && (
          <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded">
            {error}
          </div>
        )}

        {/* Create/Edit Form */}
        {showCreateForm && (
          <Card className="p-6">
            <h2 className="text-xl font-semibold mb-4">
              {editingCourse ? "Edit Course" : "Create New Course"}
            </h2>
            <form onSubmit={handleSubmit} className="space-y-4">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-1">
                    Course Number *
                  </label>
                  <Input
                    value={formData.course_number}
                    onChange={(e) =>
                      setFormData((prev) => ({
                        ...prev,
                        course_number: e.target.value,
                      }))
                    }
                    placeholder="e.g., GRK101"
                    required
                    disabled={processing}
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-1">
                    Course Name *
                  </label>
                  <Input
                    value={formData.name}
                    onChange={(e) =>
                      setFormData((prev) => ({
                        ...prev,
                        name: e.target.value,
                      }))
                    }
                    placeholder="e.g., Introduction to Greek"
                    required
                    disabled={processing}
                  />
                </div>
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">
                  Description
                </label>
                <Textarea
                  value={formData.description}
                  onChange={(e) =>
                    setFormData((prev) => ({
                      ...prev,
                      description: e.target.value,
                    }))
                  }
                  placeholder="Course description (optional)"
                  disabled={processing}
                  rows={3}
                />
              </div>
              <div className="flex gap-2 pt-2">
                <Button type="submit" disabled={processing}>
                  {processing
                    ? "Saving..."
                    : editingCourse
                      ? "Update Course"
                      : "Create Course"}
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  onClick={resetForm}
                  disabled={processing}
                >
                  Cancel
                </Button>
              </div>
            </form>
          </Card>
        )}

        {/* Admin Management Modal */}
        {managingAdmins && (
          <Card className="p-6">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-xl font-semibold mb-4">
                Manage Admins: {managingAdmins.name}
              </h2>
              <Button
                variant="outline"
                onClick={() => setManagingAdmins(null)}
                icon={<span className="material-icons text-base">close</span>}
              >
                Close
              </Button>
            </div>

            {/* Add Admin Form */}
            <form onSubmit={handleAddAdmin} className="mb-6">
              <div className="flex gap-2">
                <Input
                  value={adminUserId}
                  onChange={(e) => setAdminUserId(e.target.value)}
                  placeholder="Enter Clerk User ID"
                  disabled={processing}
                  className="flex-1"
                />
                <Button
                  type="submit"
                  disabled={processing || !adminUserId.trim()}
                >
                  {processing ? "Adding..." : "Add Admin"}
                </Button>
              </div>
              <p className="text-xs text-slate-500 mt-1">
                Enter the Clerk User ID of the user you want to make a course
                admin
              </p>
            </form>

            {/* Current Admins List */}
            <div>
              <h3 className="text-sm font-medium text-slate-700 mb-2">
                Current Course Admins
              </h3>
              {courseAdmins.length === 0 ? (
                <p className="text-slate-500 text-sm">
                  No course admins assigned
                </p>
              ) : (
                <div className="space-y-2">
                  {courseAdmins.map((admin) => (
                    <div
                      key={admin.user_id}
                      className="flex items-center justify-between p-3 bg-slate-50 rounded"
                    >
                      <div>
                        <div className="font-medium">{admin.name || admin.email || admin.user_id}</div>
                        <div className="text-sm text-slate-500">
                          {admin.email}
                        </div>
                        <div className="text-xs text-slate-400">
                          Assigned:{" "}
                          {new Date(admin.assigned_at).toLocaleDateString()}
                        </div>
                      </div>
                      <Button
                        onClick={() => handleRemoveAdmin(admin.user_id)}
                        variant="danger"
                        size="sm"
                        disabled={processing}
                        icon={
                          <span className="material-icons text-sm">remove</span>
                        }
                      >
                        Remove
                      </Button>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </Card>
        )}

        {/* Courses List */}
        {loading ? (
          <div className="text-center py-8">Loading courses...</div>
        ) : (
          <div className="grid gap-4">
            {courses.length === 0 ? (
              <div className="text-center py-8 text-slate-500">
                No courses found. Create your first course to get started.
              </div>
            ) : (
              courses.map((course, index) => (
                <Card
                  key={course.course_id || `course-${index}`}
                  className="p-4"
                >
                  <div className="flex items-start justify-between gap-4">
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2 mb-2">
                        <h3 className="font-semibold text-lg">{course.name}</h3>
                        <Badge>{course.course_number}</Badge>
                      </div>
                      {course.description && (
                        <p className="text-slate-600 text-sm mb-2">
                          {course.description}
                        </p>
                      )}
                      <div className="text-xs text-slate-500">
                        Created:{" "}
                        {new Date(course.created_at).toLocaleDateString()}
                        {course.updated_at !== course.created_at && (
                          <>
                            {" "}
                            • Updated:{" "}
                            {new Date(course.updated_at).toLocaleDateString()}
                          </>
                        )}
                      </div>
                    </div>
                    <div className="flex flex-wrap gap-2">
                      <Button
                        onClick={() => fetchCourseAdmins(course)}
                        variant="outline"
                        size="sm"
                        icon={
                          <span className="material-icons text-sm">group</span>
                        }
                      >
                        Admins
                      </Button>
                      <Button
                        onClick={() => handleEdit(course)}
                        variant="outline"
                        size="sm"
                        icon={
                          <span className="material-icons text-sm">edit</span>
                        }
                      >
                        Edit
                      </Button>
                      <Button
                        onClick={() => handleDelete(course)}
                        variant="danger"
                        size="sm"
                        disabled={processing}
                        icon={
                          <span className="material-icons text-sm">delete</span>
                        }
                      >
                        Delete
                      </Button>
                    </div>
                  </div>
                </Card>
              ))
            )}
          </div>
        )}
      </div>
    </main>
  );
}
