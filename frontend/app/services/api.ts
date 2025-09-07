// API service for connecting to backend endpoints

import { useCallback } from "react";
import { useAuthenticatedFetch } from "../lib/authFetch";

const API_BASE = "/api";

export interface Story {
  id: number;
  title: string;
  week_number: number;
  day_letter: string;
}

export interface Line {
  text: string[];
  audio_url?: string;
  has_vocab_or_grammar: boolean;
}

export interface PageData {
  story_id: string;
  story_title: string;
  lines: Line[];
}

export interface Page2Data extends PageData {
  vocab_bank: string[];
}

export interface Page3Data extends PageData {
  grammar_point: string;
}

export interface Page4Data extends PageData {
  translation: string;
}

export interface APIResponse<T = any> {
  success: boolean;
  data?: T;
  error?: string;
}

interface StoriesResponse {
  stories: Story[];
}

export function useApiService() {
  const authenticatedFetch = useAuthenticatedFetch();

  const fetchAPI = useCallback(
    async <T>(
      endpoint: string,
      options?: RequestInit,
    ): Promise<APIResponse<T>> => {
      try {
        const response = await authenticatedFetch(`${API_BASE}${endpoint}`, {
          headers: {
            "Content-Type": "application/json",
            ...options?.headers,
          },
          ...options,
        });

        if (!response.ok) {
          throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }

        return await response.json();
      } catch (error) {
        console.error("API request failed:", error);
        return {
          success: false,
          error: error instanceof Error ? error.message : "Unknown error",
        };
      }
    },
    [authenticatedFetch],
  );

  return {
    getStories: useCallback((): Promise<APIResponse<StoriesResponse>> => {
      return fetchAPI<StoriesResponse>("/stories");
    }, [fetchAPI]),

    getStoryPage1: useCallback(
      (id: string): Promise<APIResponse<PageData>> => {
        return fetchAPI<PageData>(`/stories/${id}/page1`);
      },
      [fetchAPI],
    ),

    getStoryPage2: useCallback(
      (id: string): Promise<APIResponse<Page2Data>> => {
        return fetchAPI<Page2Data>(`/stories/${id}/page2`);
      },
      [fetchAPI],
    ),

    getStoryPage3: useCallback(
      (id: string): Promise<APIResponse<Page3Data>> => {
        return fetchAPI<Page3Data>(`/stories/${id}/page3`);
      },
      [fetchAPI],
    ),

    getStoryPage4: useCallback(
      (id: string): Promise<APIResponse<Page4Data>> => {
        return fetchAPI<Page4Data>(`/stories/${id}/page4`);
      },
      [fetchAPI],
    ),

    checkVocab: useCallback(
      (id: string, answers: any[]): Promise<APIResponse<any>> => {
        return fetchAPI(`/stories/${id}/check-vocab`, {
          method: "POST",
          body: JSON.stringify({ answers }),
        });
      },
      [fetchAPI],
    ),
  };
}
