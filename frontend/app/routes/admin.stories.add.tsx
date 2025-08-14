import { Form, redirect, useNavigation } from "react-router";
import { type ActionFunctionArgs } from "react-router";
import { addStory } from "../services/adminApi";
import Input from "~/components/ui/Input";
import Textarea from "~/components/ui/Textarea";
import Label from "~/components/ui/Label";
import Button from "~/components/ui/Button";

export async function action({ request }: ActionFunctionArgs) {
  const formData = await request.formData();
  const payload = Object.fromEntries(formData.entries());
  const url = new URL(request.url);
  const baseUrl = `${url.protocol}//${url.host}`;
  await addStory(
    {
      titleEn: String(payload.titleEn),
      languageCode: String(payload.languageCode),
      authorName: String(payload.authorName),
      weekNumber: Number(payload.weekNumber),
      dayLetter: String(payload.dayLetter),
      storyText: String(payload.storyText),
      descriptionText: String(payload.descriptionText || ""),
    },
    baseUrl
  );
  return redirect("/admin");
}

export default function AddStory() {
  const nav = useNavigation();
  const submitting = nav.state === "submitting";
  return (
    <main className="container mx-auto p-6">
      <h1 className="text-2xl font-bold mb-4">Add Story</h1>
      <Form method="post" className="space-y-4">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <Label htmlFor="titleEn">Title (en)</Label>
            <Input
              id="titleEn"
              name="titleEn"
              placeholder="Title (en)"
              required
            />
          </div>
          <div>
            <Label htmlFor="languageCode">Language code</Label>
            <Input
              id="languageCode"
              name="languageCode"
              placeholder="e.g. en, es, he"
              required
              pattern="^[a-z]{2}$"
              title="Two-letter ISO 639-1 code"
            />
          </div>
          <div>
            <Label htmlFor="authorName">Author name</Label>
            <Input
              id="authorName"
              name="authorName"
              placeholder="Author name"
              required
            />
          </div>
          <div>
            <Label htmlFor="weekNumber">Week number</Label>
            <Input
              id="weekNumber"
              name="weekNumber"
              type="number"
              placeholder="Week number"
              required
            />
          </div>
          <div>
            <Label htmlFor="dayLetter">Day letter</Label>
            <Input
              id="dayLetter"
              name="dayLetter"
              placeholder="a-e"
              required
              pattern="^[a-e]$"
              title="Single letter a-e"
            />
          </div>
          <div>
            <Label htmlFor="descriptionText">Short description</Label>
            <Input
              id="descriptionText"
              name="descriptionText"
              placeholder="Optional short description"
            />
          </div>
        </div>
        <div>
          <Label htmlFor="storyText">Story text</Label>
          <Textarea
            id="storyText"
            name="storyText"
            placeholder="One line per line of the story"
            rows={12}
            required
          />
        </div>
        <Button type="submit" disabled={submitting}>
          {submitting ? "Saving…" : "Save"}
        </Button>
      </Form>
    </main>
  );
}
