import { useLoaderData, useParams, Link, useNavigation } from "react-router";
import {
  redirect,
  type ActionFunctionArgs,
  type LoaderFunctionArgs,
} from "react-router";
import type { Story } from "../types/admin";
import { getStoryForEdit, updateStory } from "../services/adminApi";
import StoryJSONEditor from "../components/Admin/StoryJSONEditor";
import Button from "~/components/ui/Button";

export async function loader({ params, request }: LoaderFunctionArgs) {
  const id = Number(params.id);
  const url = new URL(request.url);
  const baseUrl = `${url.protocol}//${url.host}`;
  const data = await getStoryForEdit(id, baseUrl);
  const story: Story = (data as any).Story || (data as Story);
  return { story };
}

export async function action({ request, params }: ActionFunctionArgs) {
  const id = Number(params.id);
  const formData = await request.formData();
  const payload = JSON.parse(String(formData.get("story")) || "{}");
  const url = new URL(request.url);
  const baseUrl = `${url.protocol}//${url.host}`;
  await updateStory(id, payload, baseUrl);
  return redirect(`/admin`);
}

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
  const { story } = useLoaderData() as { story: Story };
  const { id } = useParams();
  const nav = useNavigation();

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
            await fetch(window.location.pathname, {
              method: "POST",
              body: new URLSearchParams([["story", JSON.stringify(s)]]),
            });
          }}
        />
      </Section>
    </main>
  );
}
