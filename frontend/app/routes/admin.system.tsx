import React from "react";
import { Link } from "react-router";
import Button from "~/components/ui/Button";
import { Card } from "~/components/ui/Card";
import { useUserContext } from "../contexts/UserContext";
import { useAuthenticatedFetch } from "../lib/authFetch";

export default function AdminSystem() {
  const { userInfo } = useUserContext();
  const authenticatedFetch = useAuthenticatedFetch();
  const [clearingCache, setClearingCache] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);
  const [success, setSuccess] = React.useState<string | null>(null);

  // Check if user is super admin
  const isSuperAdmin = userInfo?.is_super_admin || false;

  const handleClearCache = async () => {
    if (!confirm("Are you sure you want to clear all cache? This will affect performance temporarily.")) {
      return;
    }

    setClearingCache(true);
    setError(null);
    setSuccess(null);

    try {
      const response = await authenticatedFetch("/api/admin/cache/clear", {
        method: "POST",
      });

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      setSuccess("Cache cleared successfully");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to clear cache");
    } finally {
      setClearingCache(false);
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
            Super admin privileges required to access system management.
          </p>
          <Link
            to="/admin"
            className="text-primary-600 hover:underline mt-4 inline-block"
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
              System Management
            </h1>
            <p className="text-sm text-slate-500">
              System administration tools for super admins
            </p>
          </div>
          <Link to="/admin">
            <Button variant="outline">← Back to Dashboard</Button>
          </Link>
        </div>

        {error && (
          <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded">
            {error}
          </div>
        )}

        {success && (
          <div className="bg-green-50 border border-green-200 text-green-700 px-4 py-3 rounded">
            {success}
          </div>
        )}

        {/* Cache Management Section */}
        <Card className="p-6">
          <div className="flex items-start justify-between gap-4">
            <div className="min-w-0 flex-1">
              <h2 className="text-xl font-semibold mb-2">Cache Management</h2>
              <p className="text-slate-600 text-sm mb-4">
                Clear all cached data including story content, user access permissions,
                and vocabulary scores. This will temporarily impact performance as data
                is re-cached on demand.
              </p>
              <div className="text-xs text-slate-500">
                <strong>Warning:</strong> This action cannot be undone. All cached data
                will be permanently removed and will need to be regenerated.
              </div>
            </div>
            <Button
              onClick={handleClearCache}
              variant="danger"
              disabled={clearingCache}
              icon={
                <span className="material-icons text-sm">
                  {clearingCache ? "hourglass_empty" : "clear_all"}
                </span>
              }
            >
              {clearingCache ? "Clearing..." : "Clear All Cache"}
            </Button>
          </div>
        </Card>

        {/* System Info Section */}
        <Card className="p-6">
          <h2 className="text-xl font-semibold mb-4">System Information</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
            <div>
              <div className="font-medium text-slate-700 mb-1">User Role</div>
              <div className="text-slate-600">Super Administrator</div>
            </div>
            <div>
              <div className="font-medium text-slate-700 mb-1">User ID</div>
              <div className="text-slate-600 font-mono text-xs">
                {userInfo.user_id}
              </div>
            </div>
          </div>
        </Card>
      </div>
    </main>
  );
}
