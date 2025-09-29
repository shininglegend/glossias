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

export interface Description {
  language: string;
  text: string;
}

export interface StoryMetadata {
  storyId: number;
  weekNumber: number;
  dayLetter: string;
  title?: string | { [key: string]: string };
  description?: Description;
  videoUrl?: string;
}

export interface AudioFile {
  id: number;
  filePath: string;
  fileBucket: string;
  label: string;
}

export interface Line {
  text: string[];
  english_translation?: string;
  audio_files: AudioFile[];
  signed_audio_urls?: { [key: number]: string };
}

export interface GrammarLine {
  text: string;
  english_translation?: string;
}

export interface PageData {
  story_id: string;
  story_title: string;
  lines: Line[];
  language: string;
}

export interface GrammarPageData {
  story_id: string;
  story_title: string;
  lines: GrammarLine[];
  language: string;
}

export interface VocabData extends PageData {
  vocab_bank: string[];
}

export interface GrammarData extends GrammarPageData {
  grammar_point_id: number;
  grammar_point: string;
  grammar_description?: string;
  instances_count: number;
}

export interface TranslationLine {
  text: string;
  translation: string;
}

export interface TranslateData {
  story_id: string;
  story_title: string;
  language: string;
  lines: TranslationLine[];
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

    getStoryWithAudio: useCallback(
      (id: string): Promise<APIResponse<PageData>> => {
        return fetchAPI<PageData>(`/stories/${id}/story-with-audio`);
      },
      [fetchAPI],
    ),

    getSignedAudioURLs: useCallback(
      (
        storyId: string,
        label?: string,
      ): Promise<APIResponse<{ [key: number]: string }>> => {
        const params = label ? `?label=${encodeURIComponent(label)}` : "";
        return fetchAPI<{ [key: number]: string }>(
          `/stories/${storyId}/audio/signed${params}`,
        );
      },
      [fetchAPI],
    ),

    getStoryVocab: useCallback(
      (id: string): Promise<APIResponse<VocabData>> => {
        return fetchAPI<VocabData>(`/stories/${id}/vocab`);
      },
      [fetchAPI],
    ),

    getStoryGrammar: useCallback(
      (
        id: string,
        grammarPointId?: string,
      ): Promise<APIResponse<GrammarData>> => {
        const url = grammarPointId
          ? `/stories/${id}/grammar?grammar_point_id=${grammarPointId}`
          : `/stories/${id}/grammar`;
        return fetchAPI<GrammarData>(url);
      },
      [fetchAPI],
    ),

    getStoryMetadata: useCallback(
      (id: string): Promise<APIResponse<StoryMetadata>> => {
        return fetchAPI<StoryMetadata>(`/stories/${id}/metadata`);
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

    checkVocabLine: useCallback(
      (
        id: string,
        lineNumber: number,
        answer: string,
      ): Promise<APIResponse<{ correct: boolean }>> => {
        return fetchAPI(`/stories/${id}/check-vocab`, {
          method: "POST",
          body: JSON.stringify({
            answers: [{ line_number: lineNumber, answers: [answer] }],
          }),
        });
      },
      [fetchAPI],
    ),

    checkGrammar: useCallback(
      (
        id: string,
        grammarPointId: number,
        answers: Array<{ line_number: number; positions: number[] }>,
      ): Promise<APIResponse<any>> => {
        return fetchAPI(`/stories/${id}/check-grammar`, {
          method: "POST",
          body: JSON.stringify({
            grammar_point_id: grammarPointId,
            answers,
          }),
        });
      },
      [fetchAPI],
    ),

    getTranslations: useCallback(
      (
        id: string,
        lineNumbers: number[],
      ): Promise<APIResponse<TranslateData>> => {
        const lines = lineNumbers.map((n) => n + 1).join(","); // Convert to 1-based indexing
        return fetchAPI<TranslateData>(
          `/stories/${id}/translate?lines=[${lines}]`,
          {
            method: "POST",
          },
        );
      },
      [fetchAPI],
    ),

    getStoryScore: useCallback(
      (id: string): Promise<APIResponse<any>> => {
        return fetchAPI(`/stories/${id}/scores`);
      },
      [fetchAPI],
    ),
  };
}
