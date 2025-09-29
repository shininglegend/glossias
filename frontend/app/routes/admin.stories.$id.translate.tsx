import { useParams } from "react-router";
import React, { useState, useEffect, useRef } from "react";
import type { Story, StoryLine } from "../types/admin";
import { useAdminApi } from "../services/adminApi";
import AdminStoryNavigation from "../components/Admin/AdminStoryNavigation";
import Button from "~/components/ui/Button";

interface TranslationLine {
  lineNumber: number;
  hebrew: string;
  english: string;
  hasChanges: boolean;
}

export default function TranslateStory() {
  const { id } = useParams();
  const adminApi = useAdminApi();
  const [story, setStory] = useState<Story | null>(null);
  const [translations, setTranslations] = useState<TranslationLine[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState<Set<number>>(new Set());
  const [bulkSaving, setBulkSaving] = useState(false);

  useEffect(() => {
    async function fetchData() {
      try {
        const [storyData, translationsData] = await Promise.all([
          adminApi.getStoryForEdit(Number(id)),
          adminApi.getTranslations(Number(id), "en"),
        ]);

        if (storyData) {
          setStory(storyData);

          const translationMap = new Map(
            translationsData.map((t) => [t.lineNumber, t.translationText]),
          );

          setTranslations(
            storyData.content.lines.map((line) => ({
              lineNumber: line.lineNumber,
              hebrew: line.text,
              english: translationMap.get(line.lineNumber) || "",
              hasChanges: false,
            })),
          );
        }
      } catch (error) {
        console.error("Failed to fetch data:", error);
      } finally {
        setLoading(false);
      }
    }
    fetchData();
  }, [id]);

  const updateTranslation = (lineNumber: number, english: string) => {
    setTranslations((prev) =>
      prev.map((t) =>
        t.lineNumber === lineNumber ? { ...t, english, hasChanges: true } : t,
      ),
    );
  };

  const saveTranslation = async (lineNumber: number) => {
    setSaving((prev) => new Set(prev).add(lineNumber));
    try {
      const translation = translations.find((t) => t.lineNumber === lineNumber);
      if (translation) {
        await adminApi.saveTranslation(
          Number(id),
          lineNumber,
          translation.english,
        );

        setTranslations((prev) =>
          prev.map((t) =>
            t.lineNumber === lineNumber ? { ...t, hasChanges: false } : t,
          ),
        );
      }
    } catch (error) {
      console.error("Failed to save translation:", error);
    } finally {
      setSaving((prev) => {
        const newSet = new Set(prev);
        newSet.delete(lineNumber);
        return newSet;
      });
    }
  };

  const saveAllTranslations = async () => {
    setBulkSaving(true);
    try {
      const translationsToSave = translations
        .filter((t) => t.hasChanges)
        .map((t) => ({ lineNumber: t.lineNumber, translation: t.english }));

      if (translationsToSave.length > 0) {
        await adminApi.saveAllTranslations(Number(id), translationsToSave);
      }

      setTranslations((prev) => prev.map((t) => ({ ...t, hasChanges: false })));
    } catch (error) {
      console.error("Failed to save all translations:", error);
    } finally {
      setBulkSaving(false);
    }
  };

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

  const hasAnyChanges = translations.some((t) => t.hasChanges);

  return (
    <main className="container mx-auto p-6">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-4">
        <h1 className="text-2xl font-bold">Translate Story  for "{story.metadata.title["en"]}"</h1>
        <Button
          onClick={saveAllTranslations}
          disabled={!hasAnyChanges || bulkSaving}
          className="w-full sm:w-auto"
        >
          {bulkSaving ? "Saving..." : "Save All Changes"}
        </Button>
      </div>

      <AdminStoryNavigation storyId={id!} />

      <div className="space-y-6">
        {translations.map((translation) => (
          <TranslationLineEditor
            key={translation.lineNumber}
            translation={translation}
            onUpdate={updateTranslation}
            onSave={saveTranslation}
            isSaving={saving.has(translation.lineNumber)}
          />
        ))}
      </div>
    </main>
  );
}

interface TranslationLineEditorProps {
  translation: TranslationLine;
  onUpdate: (lineNumber: number, english: string) => void;
  onSave: (lineNumber: number) => void;
  isSaving: boolean;
}

function TranslationLineEditor({
  translation,
  onUpdate,
  onSave,
  isSaving,
}: TranslationLineEditorProps) {
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const adjustHeight = () => {
    if (textareaRef.current) {
      textareaRef.current.style.height = "auto";
      textareaRef.current.style.height =
        textareaRef.current.scrollHeight + "px";
    }
  };

  useEffect(() => {
    adjustHeight();
  }, [translation.english]);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.ctrlKey && e.key === "Enter") {
      e.preventDefault();
      onSave(translation.lineNumber);
    }
  };

  return (
    <div className="border-b pb-2 mb-2">
      <div className="grid grid-cols-1 md:grid-cols-[1fr_2fr_auto] gap-2 items-start">
        {/* Hebrew text (left side) */}
        <div>
          <div
            className="p-2 bg-gray-50 rounded text-md min-h-[32px] text-right"
            dir="rtl"
          >
            {translation.hebrew}
          </div>
        </div>

        {/* English translation (middle) */}
        <div>
          <textarea
            ref={textareaRef}
            value={translation.english}
            onChange={(e) => onUpdate(translation.lineNumber, e.target.value)}
            onInput={adjustHeight}
            onKeyDown={handleKeyDown}
            className="w-full p-2 border border-gray-300 rounded text-sm resize-none overflow-hidden focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            placeholder="Enter English translation..."
            rows={1}
          />
        </div>

        {/* Save button (right side) */}
        <div className="flex items-center gap-2">
          <span className="text-xs text-gray-400">
            #{translation.lineNumber}
          </span>
          <Button
            onClick={() => onSave(translation.lineNumber)}
            disabled={!translation.hasChanges || isSaving}
            size="sm"
            variant="outline"
            className="text-xs px-2 py-1"
          >
            {isSaving ? "..." : translation.hasChanges ? "Save" : "✓"}
          </Button>
        </div>
      </div>
    </div>
  );
}
