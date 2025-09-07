import { useParams, Link, useNavigate } from "react-router";
import React from "react";
import type { Story } from "../types/admin";
import { useAdminApi } from "../services/adminApi";
import StoryJSONEditor from "../components/Admin/StoryJSONEditor";
import Button from "~/components/ui/Button";

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
  }, [id, adminApi]);

  if (loading) {
    return (
      <main className="container mx-auto p-6">
        <div className="text-center py-8">Loading story...</div>
      </main>
    );
  }

  if (!story) {
    return (
      <main className="container mx-auto p-6">
        <div className="text-center py-8">Failed to load story</div>
      </main>
    );
  }

  return (
    <main className="container mx-auto p-6">
      <h1 className="text-2xl font-bold mb-4">Edit Story #{id}</h1>
      <div className="mb-4 flex gap-3">
        <Link to={`/admin/stories/${id}/annotate`}>
          <Button variant="outline" size="sm">
            Annotate
          </Button>
        </Link>
        <Link to={`/admin/stories/${id}/metadata`}>
          <Button variant="outline" size="sm">
            Metadata
          </Button>
        </Link>
      </div>

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
    </main>
  );
}
