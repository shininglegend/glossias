// Courses API client for course management operations

import { useCallback } from "react";
import { useAuthenticatedFetch } from "../lib/authFetch";

export interface Course {
  course_id: number;
  course_number: string;
  name: string;
  description?: string;
  created_at: string;
  updated_at: string;
}

export interface CourseAdmin {
  user_id: string;
  email: string;
  name: string;
  assigned_at: string;
}

export interface CreateCourseRequest {
  course_number: string;
  name: string;
  description?: string;
}

export interface UpdateCourseRequest {
  course_number: string;
  name: string;
  description?: string;
}

export interface AddCourseAdminRequest {
  user_id: string;
}

type Json<T> = Promise<T>;

export function useCoursesApi() {
  const authenticatedFetch = useAuthenticatedFetch();

  const request = useCallback(
    async <T>(path: string, init?: RequestInit): Json<T> => {
      const url = `/api/admin/courses${path}`;
      const res = await authenticatedFetch(url, {
        headers: {
          Accept: "application/json",
          "Content-Type": "application/json",
          ...(init?.headers || {}),
        },
        ...init,
      });
      if (!res.ok) {
        const text = await res.text();
        throw new Error(`HTTP ${res.status}: ${text || res.statusText}`);
      }
      return res.json();
    },
    [authenticatedFetch],
  );

  return {
    // GET /courses - List all courses
    getCourses: useCallback(async (): Json<{ courses: Course[] }> => {
      return request<{ courses: Course[] }>("");
    }, [request]),

    // POST /courses - Create course
    createCourse: useCallback(
      async (courseData: CreateCourseRequest): Json<{ course: Course }> => {
        return request<{ course: Course }>("", {
          method: "POST",
          body: JSON.stringify(courseData),
        });
      },
      [request],
    ),

    // GET /courses/{id} - Get specific course
    getCourse: useCallback(
      async (id: number): Json<{ course: Course }> => {
        return request<{ course: Course }>(`/${id}`);
      },
      [request],
    ),

    // PUT /courses/{id} - Update course
    updateCourse: useCallback(
      async (
        id: number,
        courseData: UpdateCourseRequest,
      ): Json<{ course: Course }> => {
        return request<{ course: Course }>(`/${id}`, {
          method: "PUT",
          body: JSON.stringify(courseData),
        });
      },
      [request],
    ),

    // DELETE /courses/{id} - Delete course
    deleteCourse: useCallback(
      async (id: number): Json<{ success: boolean }> => {
        return request<{ success: boolean }>(`/${id}`, {
          method: "DELETE",
        });
      },
      [request],
    ),

    // GET /courses/{id}/admins - List course admins
    getCourseAdmins: useCallback(
      async (id: number): Json<{ admins: CourseAdmin[] }> => {
        return request<{ admins: CourseAdmin[] }>(`/${id}/admins`);
      },
      [request],
    ),

    // POST /courses/{id}/admins - Add course admin
    addCourseAdmin: useCallback(
      async (
        id: number,
        adminData: AddCourseAdminRequest,
      ): Json<{ success: boolean }> => {
        return request<{ success: boolean }>(`/${id}/admins`, {
          method: "POST",
          body: JSON.stringify(adminData),
        });
      },
      [request],
    ),

    // DELETE /courses/{id}/admins/{user_id} - Remove course admin
    removeCourseAdmin: useCallback(
      async (courseId: number, userId: string): Json<{ success: boolean }> => {
        return request<{ success: boolean }>(`/${courseId}/admins/${userId}`, {
          method: "DELETE",
        });
      },
      [request],
    ),
  };
}
