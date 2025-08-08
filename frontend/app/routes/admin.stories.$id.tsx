import { useLoaderData, Form, redirect, useParams, Link } from "react-router";

type Story = any;

export async function loader({ params }: { params: { id: string } }) {
  const res = await fetch(`/admin/stories/${params.id}`, { headers: { Accept: "application/json" } });
  if (!res.ok) throw new Error("Failed to load story");
  const json = await res.json();
  return json.Story || json.story || json; // depending on server response shape
}

export async function action({ request, params }: { request: Request; params: { id: string } }) {
  const formData = await request.formData();
  const payload = Object.fromEntries(formData.entries());
  const res = await fetch(`/admin/stories/${params.id}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  if (!res.ok) throw new Error("Failed to save story");
  return redirect(`/admin`);
}

export default function EditStory() {
  const story = useLoaderData() as Story;
  const { id } = useParams();
  return (
    <main className="container mx-auto p-6">
      <h1 className="text-2xl font-bold mb-4">Edit Story #{id}</h1>
      <div className="mb-4">
        <Link to={`/admin/stories/${id}/annotate`} className="text-blue-600">Grammar & Vocabulary</Link>
      </div>
      <Form method="post">
        <textarea name="Content" defaultValue={JSON.stringify(story, null, 2)} className="border p-2 w-full" rows={20} />
        <button type="submit" className="mt-4 px-4 py-2 bg-blue-600 text-white rounded">Save</button>
      </Form>
    </main>
  );
}


