// Admin annotator route with toolbar actions
import { useParams } from "react-router";
import React from "react";
import Story from "../components/Annotator/Story";
import { useAdminApi } from "../services/adminApi";
import Button from "~/components/ui/Button";
import AdminStoryNavigation from "../components/Admin/AdminStoryNavigation";
import ConfirmDialog from "~/components/ui/ConfirmDialog";

export default function AdminAnnotateRoute() {
  const params = useParams();
  const id = Number(params.id);
  const adminApi = useAdminApi();
  const [refreshKey, setRefreshKey] = React.useState(0);
  const [busy, setBusy] = React.useState(false);
  const [showConfirmDialog, setShowConfirmDialog] = React.useState(false);
  if (!id) return <div>Invalid story ID</div>;

  const handleClear = async () => {
    setBusy(true);
    try {
      await adminApi.clearAnnotations(id);
      setRefreshKey((k) => k + 1);
    } finally {
      setBusy(false);
      setShowConfirmDialog(false);
    }
  };

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Edit Annotations</h1>
        <Button
          variant="danger"
          onClick={() => setShowConfirmDialog(true)}
          disabled={busy}
        >
          {busy ? "Clearing…" : "Clear All Annotations"}
        </Button>
      </div>
      <AdminStoryNavigation storyId={id} />
      <h3>Story</h3>
      <Story key={refreshKey} storyId={id} />
      <ConfirmDialog
        isOpen={showConfirmDialog}
        onClose={() => setShowConfirmDialog(false)}
        onConfirm={handleClear}
        variant="clear"
        message="This will permanently remove all annotations from this story. This action cannot be undone."
        loading={busy}
      />
    </div>
  );
}
