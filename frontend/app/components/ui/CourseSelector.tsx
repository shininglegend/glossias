import React from "react";
import { useCoursesApi, type Course } from "~/services/coursesApi";
import Label from "./Label";

interface CourseSelectorProps {
  value?: number;
  onChange: (courseId: number | undefined) => void;
  label?: string;
  required?: boolean;
  disabled?: boolean;
  placeholder?: string;
  className?: string;
}

export default function CourseSelector({
  value,
  onChange,
  label = "Course",
  required = false,
  disabled = false,
  placeholder = "Select a course",
  className = "",
}: CourseSelectorProps) {
  const coursesApi = useCoursesApi();
  const [courses, setCourses] = React.useState<Course[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);

  React.useEffect(() => {
    let cancelled = false;

    const loadCourses = async () => {
      try {
        setLoading(true);
        setError(null);
        const response = await coursesApi.getCourses();
        if (!cancelled) {
          setCourses(response.courses);
        }
      } catch (err) {
        if (!cancelled) {
          console.error("Failed to load courses:", err);
          setError("Failed to load courses");
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    };

    loadCourses();

    return () => {
      cancelled = true;
    };
  }, []);

  const handleChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const selectedValue = e.target.value;
    onChange(selectedValue ? parseInt(selectedValue, 10) : undefined);
  };

  return (
    <div className={className}>
      {label && (
        <Label htmlFor="courseSelector" className="mb-2">
          {label}
          {required && <span className="text-red-500 ml-1">*</span>}
        </Label>
      )}
      <select
        id="courseSelector"
        value={value || ""}
        onChange={handleChange}
        disabled={disabled || loading}
        required={required}
        className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 disabled:bg-gray-100 disabled:cursor-not-allowed"
      >
        <option value="">{loading ? "Loading courses..." : placeholder}</option>
        {courses.map((course) => (
          <option key={course.course_id} value={course.course_id}>
            {course.course_number} - {course.name}
          </option>
        ))}
      </select>
      {error && <p className="mt-1 text-sm text-red-600">{error}</p>}
    </div>
  );
}
