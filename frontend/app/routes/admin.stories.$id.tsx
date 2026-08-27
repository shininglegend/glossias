import { useParams, useNavigate } from "react-router";
import React from "react";
import type { Story } from "../types/admin";
import { useAdminApi } from "../services/adminApi";
import StoryJSONEditor from "../components/Admin/StoryJSONEditor";
import AdminStoryPage from "../components/Admin/AdminStoryPage";

function Section({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  return (
    <section className="mb-6">
      <h2 className="text-lg font-semibold mb-2">{title}</h2>
      {children}
    </section>
  );
}

export default function EditStory() {
  const { id } = useParams();
  const navigate = useNavigate();
  const adminApi = useAdminApi();
  const [story, setStory] = React.useState<Story | null>(null);
  const [loading, setLoading] = React.useState(true);

  React.useEffect(() => {
    async function fetchStory() {
      try {
        const data = await adminApi.getStoryForEdit(Number(id));
        setStory(data ? data : null);
      } catch (error) {
        console.error("Failed to fetch story:", error);
      } finally {
        setLoading(false);
      }
    }
    fetchStory();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id]);

  return (
    <AdminStoryPage
      storyId={id!}
      title="Raw JSON"
      description="Raw JSON for the whole story. Prefer the editors above for routine changes."
    >
      {loading && <div className="text-center py-8">Loading story...</div>}
      {!loading && !story && (
        <div className="text-center py-8">Failed to load story</div>
      )}
      {story && (
        <Section title="Raw JSON">
          <StoryJSONEditor
            value={story}
            onSubmit={async (s) => {
              try {
                await adminApi.updateStory(Number(id), s);
                navigate("/admin");
              } catch (error) {
                console.error("Failed to update story:", error);
              }
            }}
          />
        </Section>
      )}
    </AdminStoryPage>
  );
}
