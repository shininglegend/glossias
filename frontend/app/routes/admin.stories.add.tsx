import { Form, redirect } from "react-router";
import { getAdminBase } from "../config";

export async function action({ request }: { request: Request }) {
  const formData = await request.formData();
  const payload = Object.fromEntries(formData.entries());
  const res = await fetch(`${getAdminBase()}/admin/stories/add`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      titleEn: payload.titleEn,
      languageCode: payload.languageCode,
      authorName: payload.authorName,
      weekNumber: Number(payload.weekNumber),
      dayLetter: payload.dayLetter,
      storyText: payload.storyText,
      descriptionText: payload.descriptionText || "",
    }),
  });
  if (!res.ok) throw new Error("Failed to add story");
  const json = await res.json();
  return redirect(`/admin`);
}

export default function AddStory() {
  return (
    <main className="container mx-auto p-6">
      <h1 className="text-2xl font-bold mb-4">Add Story</h1>
      <Form method="post">
        <div className="grid grid-cols-2 gap-4">
          <input name="titleEn" placeholder="Title (en)" className="border p-2" required />
          <input name="languageCode" placeholder="Language code" className="border p-2" required />
          <input name="authorName" placeholder="Author name" className="border p-2" required />
          <input name="weekNumber" type="number" placeholder="Week number" className="border p-2" required />
          <input name="dayLetter" placeholder="Day letter (a-e)" className="border p-2" required />
          <input name="descriptionText" placeholder="Description" className="border p-2" />
        </div>
        <textarea name="storyText" placeholder="Story text (one line per line)" className="border p-2 w-full mt-4" rows={10} required />
        <button type="submit" className="mt-4 px-4 py-2 bg-blue-600 text-white rounded">Save</button>
      </Form>
    </main>
  );
}


