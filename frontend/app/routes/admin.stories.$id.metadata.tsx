import { useParams } from "react-router";
import React from "react";
import type { StoryMetadata } from "../types/admin";
import { useAdminApi } from "../services/adminApi";
import MetadataForm from "../components/Admin/MetadataForm";
import AdminStoryPage from "../components/Admin/AdminStoryPage";
import { useUnsavedChangesGuard } from "../hooks/useUnsavedChangesGuard";
import { pageMeta } from "~/lib/pageTitle";

export function meta() {
  return pageMeta("Edit Metadata");
}

export default function EditMetadata() {
  const { id } = useParams();
  const adminApi = useAdminApi();
  const [metadata, setMetadata] = React.useState<StoryMetadata | null>(null);

  const [loading, setLoading] = React.useState(true);
  const [saving, setSaving] = React.useState(false);
  const [hasUnsavedChanges, setHasUnsavedChanges] = React.useState(false);
  const [justSaved, setJustSaved] = React.useState(false);

  useUnsavedChangesGuard(hasUnsavedChanges);

  React.useEffect(() => {
    async function fetchMetadata() {
      try {
        const data = await adminApi.getMetadata(Number(id));
        setMetadata(data.story.metadata as StoryMetadata);
      } catch (error) {
        console.error("Failed to fetch metadata:", error);
      } finally {
        setLoading(false);
      }
    }
    fetchMetadata();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id]);
  const status = (
    <>
      {saving && <span className="text-sm text-slate-500">Saving…</span>}
      {!saving && hasUnsavedChanges && (
        <span className="text-sm text-orange-600">Unsaved changes</span>
      )}
      {!saving && !hasUnsavedChanges && justSaved && (
        <span className="text-sm text-green-600">Saved!</span>
      )}
    </>
  );

  return (
    <AdminStoryPage
      storyId={id!}
      title="Metadata"
      description="Title, description, language and other story-level details."
      actions={status}
    >
      {loading && <div className="text-center py-8">Loading metadata...</div>}
      {!loading && !metadata && (
        <div className="text-center py-8">Failed to load metadata</div>
      )}
      {metadata && (
        <MetadataForm
          value={metadata}
          onHasChanges={setHasUnsavedChanges}
          onResetSaveStatus={() => setJustSaved(false)}
          onSubmit={async (m) => {
            setSaving(true);
            setHasUnsavedChanges(false);
            try {
              await adminApi.updateMetadata(Number(id), m);
              setMetadata(m);
              setJustSaved(true);
              setTimeout(() => setJustSaved(false), 2000);
            } catch (error) {
              console.error("Failed to save metadata:", error);
            } finally {
              setSaving(false);
            }
          }}
        />
      )}
    </AdminStoryPage>
  );
}
