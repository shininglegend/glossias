// src/components/Story.tsx
import React, { useEffect, useState } from "react";
import { StoryLine } from "../types";
import Line from "./Line.tsx";
import {
  createAnnotationRequest,
  AnnotationType,
  ApiResponse,
  ApiError,
} from "../types/api.ts";

export default function Story({ storyId }: { storyId: number }) {
  const [lines, setLines] = useState<StoryLine[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchStory = async () => {
      try {
        const response = await fetch("/admin/stories/api/" + storyId);
        if (!response.ok) throw new Error("Failed to fetch story");
        const data = await response.json();
        setLines(data.content.lines);
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
      const response = await fetch("/admin/stories/api/" + storyId, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(request),
      });

      if (!response.ok) {
        const error = (await response.json()) as ApiError;
        throw new Error(error.error);
      }

      // Refresh data
      const refreshed = await fetch("/admin/stories/api/" + storyId);
      const data = (await refreshed.json()) as ApiResponse;
      setLines(data.content.lines);
    } catch (err) {
      alert("Failed to save annotation");
      console.error(err);
    }
  };

  if (loading) return <div>Loading...</div>;
  if (error) return <div>Error: {error}</div>;

  return (
    <div className="story-container">
      {lines.map((line) => (
        <Line key={line.lineNumber} line={line} onSelect={handleAnnotation} />
      ))}
      {/* [+] Add footnotes section */}
      <hr />
      <div className="footnotes-section">
        <h3>Footnotes</h3>
        {lines.map((line) =>
          line.footnotes.map((footnote, index) => (
            <div key={`${line.lineNumber}-${index}`} className="footnote">
              <div className="footnote-line">Line {line.lineNumber}</div>
              <div className="footnote-text">{footnote.text}</div>
              {footnote.references && footnote.references.length > 0 && (
                <div className="footnote-refs">
                  References: {footnote.references.join(", ")}
                </div>
              )}
            </div>
          )),
        )}
      </div>
    </div>
  );
}
