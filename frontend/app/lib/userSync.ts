import { useAuth, useUser } from "@clerk/react-router";
import { useCallback, useEffect, useState } from "react";
import { useAuthenticatedFetch } from "./authFetch";

export interface UserInfo {
  user_id: string;
  email: string;
  name: string;
  is_super_admin: boolean;
  course_admin_rights: {
    course_id: number;
    course_number: string;
    course_name: string;
    assigned_at: string;
  }[];
}

export function useUserSync() {
  const { isLoaded, isSignedIn } = useAuth();
  const { user } = useUser();
  const authenticatedFetch = useAuthenticatedFetch();
  const [userInfo, setUserInfo] = useState<UserInfo | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const syncUser = useCallback(async () => {
    if (!isSignedIn || !user) return;

    setLoading(true);
    setError(null);

    try {
      const response = await authenticatedFetch("/api/me");
      if (!response.ok) {
        throw new Error(`Failed to sync user: ${response.status}`);
      }
      const resp = await response.json();
      const userData = resp.data;
      setUserInfo(userData);
    } catch (err) {
      const errorMessage =
        err instanceof Error ? err.message : "Failed to sync user";
      setError(errorMessage);
      console.error("User sync error:", err);
    } finally {
      setLoading(false);
    }
  }, [isSignedIn, user, authenticatedFetch]);

  // Auto-sync when user signs in
  useEffect(() => {
    if (isLoaded && isSignedIn && user && !userInfo && !loading) {
      syncUser();
    }
  }, [isLoaded, isSignedIn, user?.id, userInfo, loading]);

  return {
    userInfo,
    loading,
    error,
    syncUser,
    isLoaded,
    isSignedIn,
  };
}
