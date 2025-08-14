// Admin annotator route with toolbar actions
import { useParams, Link } from "react-router";
import React from "react";
import Story from "../components/Annotator/Story";
import { clearAnnotations } from "../services/adminApi";
import Button from "~/components/ui/Button";

export default function AdminAnnotateRoute() {
  const params = useParams();
  const id = Number(params.id);
  const [refreshKey, setRefreshKey] = React.useState(0);
  const [busy, setBusy] = React.useState(false);
  if (!id) return <div>Invalid story ID</div>;

  const handleClear = async () => {
    setBusy(true);
    try {
      await clearAnnotations(id);
      setRefreshKey((k) => k + 1);
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <div className="flex gap-2">
          <Link to={`/admin/stories/${id}`}>
            <Button variant="outline" size="sm">Back to Edit</Button>
          </Link>
          <Link to={`/admin/stories/${id}/metadata`}>
            <Button variant="outline" size="sm">Metadata</Button>
          </Link>
        </div>
        <Button variant="danger" onClick={handleClear} disabled={busy}>
          {busy ? "Clearing…" : "Clear All Annotations"}
        </Button>
      </div>
      <Story key={refreshKey} storyId={id} />
    </div>
  );
}


