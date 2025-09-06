// Admin API client aligned to backend routes under /admin/stories

import { useCallback } from "react";
import { useAuthenticatedFetch } from "../lib/authFetch";
import type { Story, StoryMetadata, StoryContent } from "../types/admin";

type Json<T> = Promise<T>;

export function useAdminApi() {
  const authenticatedFetch = useAuthenticatedFetch();

  const request = useCallback(
    async <T>(path: string, init?: RequestInit, baseUrl?: string): Json<T> => {
      const url = baseUrl ? `${baseUrl}/api/admin${path}` : `/api/admin${path}`;
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
    },
    [authenticatedFetch],
  );

  return {
    // GET stories/:id -> { Story, Success }
    getStoryForEdit: useCallback(
      async (
        id: number,
        baseUrl?: string,
      ): Json<{ Story: Story; Success: boolean } | Story> => {
        const data = await request<any>(
          `/stories/${id}`,
          {
            headers: { Accept: "application/json" },
          },
          baseUrl,
        );
        // Tolerate {Story,Success} or raw Story
        return data.Story ? data : { Story: data as Story, Success: true };
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
      ): Json<{ story: Story; Success: boolean }> => {
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
  };
}
