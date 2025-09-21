// Admin API client aligned to backend routes under /admin/stories

import { useCallback } from "react";
import { useAuthenticatedFetch } from "../lib/authFetch";
import type {
  Story,
  StoryMetadata,
  StoryContent,
  GrammarPoint,
} from "../types/admin";

type Json<T> = Promise<T>;

// Cache for pending requests to prevent duplicates
const pendingRequests = new Map<string, Promise<any>>();

export function useAdminApi() {
  const authenticatedFetch = useAuthenticatedFetch();

  const request = useCallback(
    async <T>(path: string, init?: RequestInit, baseUrl?: string): Json<T> => {
      const url = baseUrl ? `${baseUrl}/api/admin${path}` : `/api/admin${path}`;
      const method = init?.method || "GET";
      const cacheKey = `${method}:${url}`;

      // For GET requests, check if there's already a pending request
      if (method === "GET" && pendingRequests.has(cacheKey)) {
        return pendingRequests.get(cacheKey);
      }

      const requestPromise = (async () => {
        const res = await authenticatedFetch(url, {
          headers: {
            Accept: "application/json",
            "Content-Type": "application/json",
            ...(init?.headers || {}),
          },
          ...init,
        });
        if (!res.ok) {
          const text = await res.text();
          throw new Error(`HTTP ${res.status}: ${text || res.statusText}`);
        }
        return res.json();
      })();

      // Cache GET requests
      if (method === "GET") {
        pendingRequests.set(cacheKey, requestPromise);

        // Clean up cache when request completes
        requestPromise.finally(() => {
          pendingRequests.delete(cacheKey);
        });
      }

      return requestPromise;
    },
    [authenticatedFetch],
  );

  return {
    // GET stories/:id -> { Story, Success }
    getStoryForEdit: useCallback(
      async (id: number, baseUrl?: string): Json<Story | undefined> => {
        const data = await request<any>(
          `/stories/${id}`,
          {
            headers: { Accept: "application/json" },
          },
          baseUrl,
        );
        const story = data.story;
        return story;
      },
      [request],
    ),

    // PUT stories/:id expects full Story JSON
    updateStory: useCallback(
      async (
        id: number,
        story: Story,
        baseUrl?: string,
      ): Json<{ Success: boolean; Story: Story }> => {
        return request<{ Success: boolean; Story: Story }>(
          `/stories/${id}`,
          {
            method: "PUT",
            body: JSON.stringify(story),
          },
          baseUrl,
        );
      },
      [request],
    ),

    // GET stories/:id/metadata -> { Story }
    getMetadata: useCallback(
      async (
        id: number,
        baseUrl?: string,
      ): Json<{
        story: Story;
        success: boolean;
      }> => {
        const data = await request<any>(
          `/stories/${id}/metadata`,
          {
            headers: { Accept: "application/json" },
          },
          baseUrl,
        );
        return data;
      },
      [request],
    ),

    // PUT stories/:id/metadata expects StoryMetadata
    updateMetadata: useCallback(
      async (
        id: number,
        metadata: StoryMetadata,
        baseUrl?: string,
      ): Json<{ success: boolean }> => {
        return request<{ success: boolean }>(
          `/stories/${id}/metadata`,
          {
            method: "PUT",
            body: JSON.stringify(metadata),
          },
          baseUrl,
        );
      },
      [request],
    ),

    // GET /stories/:id -> { content }
    getStoryContent: useCallback(
      async (id: number, baseUrl?: string): Json<{ content: StoryContent }> => {
        return request<{ content: StoryContent }>(
          `/stories/${id}`,
          undefined,
          baseUrl,
        );
      },
      [request],
    ),

    // PUT /stories/:id with one of vocabulary | grammar | footnote
    addAnnotation: useCallback(
      async (
        id: number,
        req: {
          lineNumber: number;
          vocabulary?: Story["content"]["lines"][number]["vocabulary"][number];
          grammar?: Story["content"]["lines"][number]["grammar"][number];
          footnote?: Story["content"]["lines"][number]["footnotes"][number];
        },
        baseUrl?: string,
      ): Json<{ success: boolean }> => {
        return request<{ success: boolean }>(
          `/stories/${id}`,
          {
            method: "PUT",
            body: JSON.stringify(req),
          },
          baseUrl,
        );
      },
      [request],
    ),

    // DELETE /stories/:id/annotations -> should delete all annotations on this story
    clearAnnotations: useCallback(
      async (id: number, baseUrl?: string): Json<{ success: boolean }> => {
        return request<{ success: boolean }>(
          `/stories/${id}/annotations`,
          {
            method: "DELETE",
          },
          baseUrl,
        );
      },
      [request],
    ),

    // POST /stories/add for new story
    addStory: useCallback(
      async (
        payload: {
          titleEn: string;
          languageCode: string;
          authorName: string;
          weekNumber: number;
          dayLetter: string;
          storyText: string; // newline-separated lines
          descriptionText?: string;
          courseId?: number;
        },
        baseUrl?: string,
      ): Json<{ success: boolean; storyId: number }> => {
        return request<{ success: boolean; storyId: number }>(
          `/stories`,
          {
            method: "POST",
            body: JSON.stringify(payload),
          },
          baseUrl,
        );
      },
      [request],
    ),

    // DELETE /stories/:id -> Deletes the story
    deleteStory: useCallback(
      async (id: number, baseUrl?: string): Json<{ success: boolean }> => {
        return request<{ success: boolean }>(
          `/stories/${id}`,
          {
            method: "DELETE",
          },
          baseUrl,
        );
      },
      [request],
    ),

    // GET /stories/:id/translations/lang/:lang -> Translation[]
    getTranslations: useCallback(
      async (
        id: number,
        languageCode: string = "en",
        baseUrl?: string,
      ): Json<
        Array<{
          storyId: number;
          lineNumber: number;
          languageCode: string;
          translationText: string;
        }>
      > => {
        return request<
          Array<{
            storyId: number;
            lineNumber: number;
            languageCode: string;
            translationText: string;
          }>
        >(
          `/stories/${id}/translations/lang/${languageCode}`,
          undefined,
          baseUrl,
        );
      },
      [request],
    ),

    // PUT /stories/:id/translations/line
    saveTranslation: useCallback(
      async (
        id: number,
        lineNumber: number,
        translation: string,
        languageCode: string = "en",
        baseUrl?: string,
      ): Json<{ success: boolean }> => {
        return request<{ success: boolean }>(
          `/stories/${id}/translations/line`,
          {
            method: "PUT",
            body: JSON.stringify({
              lineNumber,
              languageCode,
              translation,
            }),
          },
          baseUrl,
        );
      },
      [request],
    ),

    // PUT /stories/:id/translations
    saveAllTranslations: useCallback(
      async (
        id: number,
        translations: Array<{ lineNumber: number; translation: string }>,
        languageCode: string = "en",
        baseUrl?: string,
      ): Json<{ success: boolean }> => {
        return request<{ success: boolean }>(
          `/stories/${id}/translations`,
          {
            method: "PUT",
            body: JSON.stringify({
              languageCode,
              translations,
            }),
          },
          baseUrl,
        );
      },
      [request],
    ),
  };
}
