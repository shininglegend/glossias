// API service for connecting to backend endpoints

import { useCallback, useRef, useMemo } from "react";
import { useAuthenticatedFetch } from "../lib/authFetch";
import type {
  NavigationGuidanceResponse,
  Story as CourseStory,
  TextSegment,
} from "../types/api";

const API_BASE = "/api";

export interface Story {
  id: number;
  title: string;
  week_number: number;
  day_letter: string;
  course_id?: number;
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

export interface VocabLine {
  text: TextSegment[];
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

export interface VocabData {
  story_id: string;
  story_title: string;
  lines: VocabLine[];
  language: string;
  vocab_bank: string[];
}

export interface GrammarData extends GrammarPageData {
  grammar_point_id: number;
  grammar_point: string;
  grammar_description?: string;
  instances_count: number;
  found_instances?: Array<{
    line_number: number;
    position: [number, number];
    text: string;
  }>;
  incorrect_instances?: Array<{
    line_number: number;
    position: [number, number];
    text: string;
    correct: boolean;
  }>;
  next_grammar_point?: number;
}

export interface TranslationLine {
  text: string;
  translation: string;
  line_number: number;
}

export interface TranslateData {
  story_id: string;
  story_title: string;
  language: string;
  lines: TranslationLine[];
  returned_lines: number[];
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
  const pendingRequests = useRef<Map<string, Promise<APIResponse<any>>>>(
    new Map(),
  );

  const fetchAPI = useCallback(
    async <T>(
      endpoint: string,
      options?: RequestInit,
    ): Promise<APIResponse<T>> => {
      const requestKey = `${endpoint}:${JSON.stringify(options || {})}`;

      // Check for pending request
      const pending = pendingRequests.current.get(requestKey);
      if (pending) {
        return (await pending) as APIResponse<T>;
      }

      // Create new request
      const requestPromise = (async (): Promise<APIResponse<T>> => {
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
        } finally {
          pendingRequests.current.delete(requestKey);
        }
      })();

      pendingRequests.current.set(requestKey, requestPromise);
      return await requestPromise;
    },
    [authenticatedFetch],
  );

  return useMemo(
    () => ({
      getStories: (): Promise<APIResponse<StoriesResponse>> => {
        return fetchAPI<StoriesResponse>("/stories");
      },

      getStoryWithAudio: (id: string): Promise<APIResponse<PageData>> => {
        return fetchAPI<PageData>(`/stories/${id}/story-with-audio`);
      },

      getSignedAudioURLs: (
        storyId: string,
        label?: string,
      ): Promise<APIResponse<{ [key: number]: string }>> => {
        const params = label ? `?label=${encodeURIComponent(label)}` : "";
        return fetchAPI<{ [key: number]: string }>(
          `/stories/${storyId}/audio/signed${params}`,
        );
      },

      getStoryVocab: (id: string): Promise<APIResponse<VocabData>> => {
        return fetchAPI<VocabData>(`/stories/${id}/vocab`);
      },

      getStoryGrammar: (
        id: string,
        grammarPointId?: string,
      ): Promise<APIResponse<GrammarData>> => {
        const url = grammarPointId
          ? `/stories/${id}/grammar?grammar_point_id=${grammarPointId}`
          : `/stories/${id}/grammar`;
        return fetchAPI<GrammarData>(url);
      },

      getStoryMetadata: (id: string): Promise<APIResponse<StoryMetadata>> => {
        return fetchAPI<StoryMetadata>(`/stories/${id}/metadata`);
      },

      checkVocab: (id: string, answers: any[]): Promise<APIResponse<any>> => {
        return fetchAPI(`/stories/${id}/check-vocab`, {
          method: "POST",
          body: JSON.stringify({ answers }),
        });
      },

      checkIndividualVocab: (
        id: string,
        vocabKey: string,
        answer: string,
      ): Promise<
        APIResponse<{
          correct: boolean;
          line_complete: boolean;
          original_line?: string;
        }>
      > => {
        return fetchAPI(`/stories/${id}/check-vocab`, {
          method: "POST",
          body: JSON.stringify({
            vocab_key: vocabKey,
            answer: answer,
          }),
        });
      },

      checkGrammar: (
        id: string,
        grammarPointId: number,
        request: {
          grammar_point_id: number;
          line_number: number;
          position: number;
        },
      ): Promise<
        APIResponse<{
          correct: boolean;
          matched_position?: [number, number];
          total_instances: number;
          next_grammar_point: number | null;
        }>
      > => {
        return fetchAPI(`/stories/${id}/check-grammar`, {
          method: "POST",
          body: JSON.stringify(request),
        });
      },

      getTranslations: (
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

      getStoryScore: (id: string): Promise<APIResponse<any>> => {
        return fetchAPI(`/stories/${id}/scores`);
      },

      getNavigationGuidance: (
        storyId: string,
        currentPage: string,
      ): Promise<APIResponse<NavigationGuidanceResponse>> => {
        return fetchAPI<NavigationGuidanceResponse>(
          `/stories/${storyId}/next`,
          {
            method: "POST",
            body: JSON.stringify({ currentPage }),
          },
        );
      },

      // Admin endpoints
      getCourseStories: (
        courseId: string,
      ): Promise<APIResponse<CourseStory[]>> => {
        return fetchAPI<CourseStory[]>(`/stories/by-course/${courseId}`);
      },

      getStoryStudentPerformance: (
        storyId: string,
      ): Promise<APIResponse<any>> => {
        return fetchAPI(`/admin/courses/${storyId}/student-performance`);
      },
    }),
    [fetchAPI],
  );
}
