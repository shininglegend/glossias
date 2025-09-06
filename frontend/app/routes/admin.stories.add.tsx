import { useNavigate } from "react-router";
import { useAdminApi } from "../services/adminApi";
import Input from "~/components/ui/Input";
import Textarea from "~/components/ui/Textarea";
import Label from "~/components/ui/Label";
import Button from "~/components/ui/Button";
import React from "react";

export default function AddStory() {
  const navigate = useNavigate();
  const adminApi = useAdminApi();
  const [submitting, setSubmitting] = React.useState(false);

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setSubmitting(true);
    try {
      const formData = new FormData(e.currentTarget);
      const payload = Object.fromEntries(formData.entries());
      await adminApi.addStory({
        titleEn: String(payload.titleEn),
        languageCode: String(payload.languageCode),
        authorName: String(payload.authorName),
        weekNumber: Number(payload.weekNumber),
        dayLetter: String(payload.dayLetter),
        storyText: String(payload.storyText),
        descriptionText: String(payload.descriptionText || ""),
      });
      navigate("/admin");
    } catch (error) {
      console.error("Failed to add story:", error);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <main className="container mx-auto p-6">
      <h1 className="text-2xl font-bold mb-4">Add Story</h1>
      <form onSubmit={handleSubmit} className="space-y-4">
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
      </form>
    </main>
  );
}
