import { useNavigate } from "react-router";
import { useUser } from "@clerk/react-router";
import { useAdminApi } from "../services/adminApi";
import Input from "~/components/ui/Input";
import Textarea from "~/components/ui/Textarea";
import Label from "~/components/ui/Label";
import Button from "~/components/ui/Button";
import CourseSelector from "~/components/ui/CourseSelector";
import Asterisk from "~/components/ui/Asterisk";
import React from "react";

export default function AddStory() {
  const navigate = useNavigate();
  const { user } = useUser();
  const adminApi = useAdminApi();
  const [submitting, setSubmitting] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);
  const [selectedCourseId, setSelectedCourseId] = React.useState<
    number | undefined
  >();
  const [authorName, setAuthorName] = React.useState("");

  // Fill author name when user data becomes available
  React.useEffect(() => {
    if (user && !authorName) {
      setAuthorName(user.fullName || user.firstName || "");
    }
  }, [user, authorName]);

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setSubmitting(true);
    setError(null);
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
        courseId: selectedCourseId,
      });
      navigate("/admin");
    } catch (error) {
      console.error("Failed to add story:", error);
      setError(error instanceof Error ? error.message : "Failed to add story");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <main className="container mx-auto p-6">
      <h1 className="text-2xl font-bold mb-4">Add Story</h1>
      {error && (
        <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded mb-4">
          {error}
        </div>
      )}
      <form onSubmit={handleSubmit} className="space-y-4">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <Label htmlFor="titleEn">
              Title (en)
              <Asterisk />
            </Label>
            <Input
              id="titleEn"
              name="titleEn"
              placeholder="Title (en)"
              required
            />
          </div>
          <div>
            <Label htmlFor="languageCode">
              Language code
              <Asterisk />
            </Label>
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
            <Label htmlFor="authorName">
              Author name
              <Asterisk />
            </Label>
            <Input
              id="authorName"
              name="authorName"
              placeholder="Author name"
              value={authorName}
              onChange={(e) => setAuthorName(e.target.value)}
              required
            />
          </div>
          <div>
            <Label htmlFor="weekNumber">Week number</Label>
            <Input
              id="weekNumber"
              name="weekNumber"
              type="number"
              placeholder="Optional Week number"
            />
          </div>
          <div>
            <Label htmlFor="dayLetter">Day letter</Label>
            <Input
              id="dayLetter"
              name="dayLetter"
              placeholder="Optional Day Letter (a-e)"
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
          <div>
            <CourseSelector
              value={selectedCourseId}
              onChange={setSelectedCourseId}
              required
            />
          </div>
        </div>
        <div>
          <Label htmlFor="storyText">
            Story text
            <Asterisk />
          </Label>
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
