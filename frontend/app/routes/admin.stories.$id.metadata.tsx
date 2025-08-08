import { useLoaderData, Form, redirect, useParams } from "react-router";

type Metadata = any;

export async function loader({ params }: { params: { id: string } }) {
  const res = await fetch(`/admin/stories/${params.id}/metadata`, { headers: { Accept: "application/json" } });
  if (!res.ok) throw new Error("Failed to load metadata");
  const json = await res.json();
  return json.Story?.Metadata || json.metadata || json; // tolerate variants
}

export async function action({ request, params }: { request: Request; params: { id: string } }) {
  const formData = await request.formData();
  const payload = Object.fromEntries(formData.entries());
  const res = await fetch(`/admin/stories/${params.id}/metadata`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  if (!res.ok) throw new Error("Failed to save metadata");
  return null;
}

export default function EditMetadata() {
  const metadata = useLoaderData() as Metadata;
  const { id } = useParams();
  return (
    <main className="container mx-auto p-6">
      <h1 className="text-2xl font-bold mb-4">Edit Metadata #{id}</h1>
      <Form method="post">
        <textarea name="Metadata" defaultValue={JSON.stringify(metadata, null, 2)} className="border p-2 w-full" rows={16} />
        <button type="submit" className="mt-4 px-4 py-2 bg-blue-600 text-white rounded">Save</button>
      </Form>
    </main>
  );
}


