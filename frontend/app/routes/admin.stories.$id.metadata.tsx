import { useParams } from "react-router";
import React from "react";
import type { StoryMetadata } from "../types/admin";
import { getMetadata, updateMetadata } from "../services/adminApi";
import MetadataForm from "../components/Admin/MetadataForm";
import Button from "~/components/ui/Button";

export default function EditMetadata() {
  const { id } = useParams();
  const [metadata, setMetadata] = React.useState<StoryMetadata | null>(null);
  const [loading, setLoading] = React.useState(true);
  const [saving, setSaving] = React.useState(false);

  React.useEffect(() => {
    async function fetchMetadata() {
      try {
        const data = await getMetadata(Number(id));
        setMetadata(data.story.metadata as StoryMetadata);
      } catch (error) {
        console.error("Failed to fetch metadata:", error);
      } finally {
        setLoading(false);
      }
    }
    fetchMetadata();
  }, [id]);
  if (loading) {
    return (
      <main className="container mx-auto p-6">
        <div className="text-center py-8">Loading metadata...</div>
      </main>
    );
  }

  if (!metadata) {
    return (
      <main className="container mx-auto p-6">
        <div className="text-center py-8">Failed to load metadata</div>
      </main>
    );
  }

  return (
    <main className="container mx-auto p-6">
      <div className="mb-4 flex items-center justify-between">
        <h1 className="text-2xl font-bold">Edit Metadata #{id}</h1>
        {saving && <span className="text-sm text-slate-500">Saving…</span>}
      </div>
      <MetadataForm
        value={metadata}
        onSubmit={async (m) => {
          setSaving(true);
          try {
            await updateMetadata(Number(id), m);
            setMetadata(m);
          } catch (error) {
            console.error("Failed to save metadata:", error);
          } finally {
            setSaving(false);
          }
        }}
      />
    </main>
  );
}
