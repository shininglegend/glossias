import { useLoaderData, useParams, useNavigation } from "react-router";
import { type LoaderFunctionArgs, type ActionFunctionArgs } from "react-router";
import type { StoryMetadata } from "../types/admin";
import { getMetadata, updateMetadata } from "../services/adminApi";
import MetadataForm from "../components/Admin/MetadataForm";
import Button from "~/components/ui/Button";

export async function loader({ params, request }: LoaderFunctionArgs) {
  const id = Number(params.id);
  const url = new URL(request.url);
  const baseUrl = `${url.protocol}//${url.host}`;
  const data = await getMetadata(id, baseUrl);
  const meta = data.story.metadata as StoryMetadata;
  return { metadata: meta };
}

export async function action({ request, params }: ActionFunctionArgs) {
  const id = Number(params.id);
  const meta: StoryMetadata = await request.json();
  const url = new URL(request.url);
  const baseUrl = `${url.protocol}//${url.host}`;
  await updateMetadata(id, meta, baseUrl);
  return { success: true } as const;
}

export default function EditMetadata() {
  const { metadata } = useLoaderData() as { metadata: StoryMetadata };
  const { id } = useParams();
  const nav = useNavigation();
  const saving = nav.state === "submitting";
  return (
    <main className="container mx-auto p-6">
      <div className="mb-4 flex items-center justify-between">
        <h1 className="text-2xl font-bold">Edit Metadata #{id}</h1>
        {saving && <span className="text-sm text-slate-500">Saving…</span>}
      </div>
      <MetadataForm
        value={metadata}
        onSubmit={async (m) => {
          await fetch(window.location.pathname, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(m),
          });
        }}
      />
    </main>
  );
}
