// Admin API client aligned to backend routes under /admin/stories

import type { Story, StoryMetadata, StoryContent } from "../types/admin";

type Json<T> = Promise<T>;

async function request<T>(
  path: string,
  init?: RequestInit,
  baseUrl?: string
): Json<T> {
  const url = baseUrl ? `${baseUrl}/api/admin${path}` : `/api/admin${path}`;
  const res = await fetch(url, {
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
}

// GET stories/:id -> { Story, Success }
export async function getStoryForEdit(
  id: number,
  baseUrl?: string
): Json<{ Story: Story; Success: boolean } | Story> {
  const data = await request<any>(
    `/stories/${id}`,
    {
      headers: { Accept: "application/json" },
    },
    baseUrl
  );
  // Tolerate {Story,Success} or raw Story
  return data.Story ? data : { Story: data as Story, Success: true };
}

// PUT stories/:id expects full Story JSON
export async function updateStory(
  id: number,
  story: Story,
  baseUrl?: string
): Json<{ Success: boolean; Story: Story }> {
  return request<{ Success: boolean; Story: Story }>(
    `/stories/${id}`,
    {
      method: "PUT",
      body: JSON.stringify(story),
    },
    baseUrl
  );
}

// GET stories/:id/metadata -> { Story }
export async function getMetadata(
  id: number,
  baseUrl?: string
): Json<{ story: Story; Success: boolean }> {
  const data = await request<any>(
    `/stories/${id}/metadata`,
    {
      headers: { Accept: "application/json" },
    },
    baseUrl
  );
  return data;
}

// PUT stories/:id/metadata expects StoryMetadata
export async function updateMetadata(
  id: number,
  metadata: StoryMetadata,
  baseUrl?: string
): Json<{ success: boolean }> {
  return request<{ success: boolean }>(
    `/stories/${id}/metadata`,
    {
      method: "PUT",
      body: JSON.stringify(metadata),
    },
    baseUrl
  );
}

// GET /stories/:id -> { content }
export async function getStoryContent(
  id: number,
  baseUrl?: string
): Json<{ content: StoryContent }> {
  return request<{ content: StoryContent }>(
    `/stories/${id}`,
    undefined,
    baseUrl
  );
}

// PUT /stories/:id with one of vocabulary | grammar | footnote
export interface AnnotationRequest {
  lineNumber: number;
  vocabulary?: Story["content"]["lines"][number]["vocabulary"][number];
  grammar?: Story["content"]["lines"][number]["grammar"][number];
  footnote?: Story["content"]["lines"][number]["footnotes"][number];
}

export async function addAnnotation(
  id: number,
  req: AnnotationRequest,
  baseUrl?: string
): Json<{ success: boolean }> {
  return request<{ success: boolean }>(
    `/stories/${id}`,
    {
      method: "PUT",
      body: JSON.stringify(req),
    },
    baseUrl
  );
}

// DELETE /stories/:id/annotations -> should delete all annotations on this story
export async function clearAnnotations(
  id: number,
  baseUrl?: string
): Json<{ success: boolean }> {
  return request<{ success: boolean }>(
    `/stories/${id}/annotations`,
    {
      method: "DELETE",
    },
    baseUrl
  );
}

// POST /stories/add for new story
export interface AddStoryPayload {
  titleEn: string;
  languageCode: string;
  authorName: string;
  weekNumber: number;
  dayLetter: string;
  storyText: string; // newline-separated lines
  descriptionText?: string;
}

export async function addStory(
  payload: AddStoryPayload,
  baseUrl?: string
): Json<{ success: boolean; storyId: number }> {
  return request<{ success: boolean; storyId: number }>(
    `/stories`,
    {
      method: "POST",
      body: JSON.stringify(payload),
    },
    baseUrl
  );
}

// DELETE /stories/:id -> Deletes the story
export async function deleteStory(
  id: number,
  baseUrl?: string
): Json<{ success: boolean }> {
  return request<{ success: boolean }>(
    `/stories/${id}`,
    {
      method: "DELETE",
    },
    baseUrl
  );
}
