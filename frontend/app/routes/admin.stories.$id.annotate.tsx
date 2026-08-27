// Admin annotator route with toolbar actions
import { useParams } from "react-router";
import React from "react";
import Story from "../components/Annotator/Story";
import { useAdminApi } from "../services/adminApi";
import Button from "~/components/ui/Button";
import AdminStoryPage from "../components/Admin/AdminStoryPage";
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
    <AdminStoryPage
      storyId={id}
      title="Annotations"
      description="Mark vocabulary and grammar in the story text; footnotes appear below."
      actions={
        <Button
          variant="danger"
          onClick={() => setShowConfirmDialog(true)}
          disabled={busy}
        >
          {busy ? "Clearing…" : "Clear All Annotations"}
        </Button>
      }
    >
      <Story key={refreshKey} storyId={id} />
      <ConfirmDialog
        isOpen={showConfirmDialog}
        onClose={() => setShowConfirmDialog(false)}
        onConfirm={handleClear}
        variant="clear"
        message="This will permanently remove all annotations from this story. This action cannot be undone."
        loading={busy}
      />
    </AdminStoryPage>
  );
}
