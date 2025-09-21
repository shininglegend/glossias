// [moved from annotator/src/components/Story.tsx]
import React, { useEffect, useState } from "react";
import Line from "./Line";
import { useAuthenticatedFetch } from "../../lib/authFetch";
import {
  createAnnotationRequest,
  type AnnotationType,
  type ApiError,
  type ApiResponse,
  type StoryLine,
  type StoryMetadata,
} from "../../types/api";

interface AudioURLsResponse {
  success: boolean;
  data: Record<string, string>; // lineNumber -> signedURL
}

export default function Story({ storyId }: { storyId: number }) {
  const authenticatedFetch = useAuthenticatedFetch();
  const [lines, setLines] = useState<StoryLine[]>([]);
  const [metadata, setMetaData] = useState<StoryMetadata | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [completeAudioURLs, setCompleteAudioURLs] = useState<
    Record<string, string>
  >({});
  const [incompleteAudioURLs, setIncompleteAudioURLs] = useState<
    Record<string, string>
  >({});

  const fetchAudioURLs = async (label: string) => {
    try {
      const response = await authenticatedFetch(
        `/api/stories/${storyId}/audio/signed?label=${label}`,
      );
      if (!response.ok) return {};
      const data: AudioURLsResponse = await response.json();
      return data.success ? data.data : {};
    } catch (e) {
      console.error(`Failed to fetch ${label} audio URLs:`, e);
      return {};
    }
  };

  useEffect(() => {
    const fetchStory = async () => {
      try {
        const response = await authenticatedFetch(
          `/api/admin/stories/${storyId}`,
        );
        if (!response.ok) throw new Error("Failed to fetch story");
        const data: ApiResponse = await response.json();

        // Map lines with empty audioFiles for now - backend doesn't return them in admin endpoint
        const linesWithAudio = data.story.content.lines.map((line) => ({
          ...line,
          audioFiles: [], // Will be populated by checking audio URLs
        }));

        setLines(linesWithAudio);
        setMetaData(data.metadata);

        // Fetch audio URLs for both labels
        const [completeURLs, incompleteURLs] = await Promise.all([
          fetchAudioURLs("complete"),
          fetchAudioURLs("incomplete"),
        ]);

        setCompleteAudioURLs(completeURLs);
        setIncompleteAudioURLs(incompleteURLs);
        setLoading(false);
      } catch (e) {
        setError(e instanceof Error ? e.message : "Unknown error");
        setLoading(false);
      }
    };
    fetchStory();
  }, [storyId]);

  const handleAnnotation = async (
    lineNumber: number,
    text: string,
    type: AnnotationType,
    start: number,
    end: number,
    data?: { text?: string; lexicalForm?: string },
  ) => {
    const request = createAnnotationRequest(
      lineNumber,
      type,
      text,
      start,
      end,
      data,
    );

    try {
      const response = await authenticatedFetch(
        `/api/admin/stories/${storyId}/annotations`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(request),
        },
      );

      if (!response.ok) {
        const err: ApiError = await response.json();
        throw new Error(err.error);
      }

      const refreshed = await authenticatedFetch(
        `/api/admin/stories/${storyId}`,
      );
      const data: ApiResponse = await refreshed.json();
      setLines(data.story.content.lines);
      setMetaData(data.metadata);
    } catch (err) {
      console.error(err);
      alert("Failed to save annotation");
    }
  };

  if (loading) return <div className="text-sm text-slate-600">Loading…</div>;
  if (error) return <div className="text-sm text-rose-700">Error: {error}</div>;

  return (
    <div className="story-container">
      {lines.map((line) => (
        <Line
          key={line.lineNumber}
          line={{ ...line, storyId: metadata?.storyId || storyId }}
          onSelect={handleAnnotation}
          completeAudioURL={completeAudioURLs[line.lineNumber.toString()]}
          incompleteAudioURL={incompleteAudioURLs[line.lineNumber.toString()]}
        />
      ))}
      <div className="mt-8 border-t pt-6">
        <h3 className="text-lg font-semibold mb-3">Footnotes</h3>
        <div className="grid gap-3">
          {lines.flatMap((line) =>
            line.footnotes.map((footnote, index) => (
              <div
                key={`${line.lineNumber}-${index}`}
                className="rounded-md border border-slate-200 bg-white p-3 shadow-sm"
              >
                <div className="text-xs text-slate-500 mb-1">
                  Line {line.lineNumber}
                </div>
                <div className="text-sm">{footnote.text}</div>
                {footnote.references && footnote.references.length > 0 && (
                  <div className="mt-1 text-xs text-slate-500">
                    References: {footnote.references.join(", ")}
                  </div>
                )}
              </div>
            )),
          )}
        </div>
      </div>
    </div>
  );
}
