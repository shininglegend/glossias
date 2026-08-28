// API service for connecting to backend endpoints

import { useCallback, useRef, useMemo } from "react";
import { useAuthenticatedFetch } from "../lib/authFetch";
import type {
  NavigationGuidanceResponse,
  ResetPhase,
  ResetProgressResult,
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

export interface IdentifyLine {
  text: TextSegment[];
  target_vocab_ids: number[];
}

export interface IdentifyTargetWord {
  id: number;
  lexical_form: string;
  audio_url?: string;
  image_url?: string;
}

export interface IdentifyData {
  story_id: string;
  story_title: string;
  language: string;
  lines: IdentifyLine[];
  target_words: IdentifyTargetWord[];
  /** Signed narration URLs keyed by 1-based line number. */
  audio_urls: { [key: string]: string };
  /** Quizzes already answered correctly on earlier visits (0-based lines). */
  correct_picks: { line_index: number; target_vocab_id: number }[];
  /** The student finished this phase on an earlier visit. */
  completed: boolean;
}

export interface RecallCard {
  id: number;
  hebrew_text: string;
  image_url?: string;
}

export interface RecallData {
  story_id: string;
  story_title: string;
  language: string;
  /** Number of narration lines; the phase is audio-only so no text is sent. */
  line_count: number;
  /** Signed narration URLs keyed by 1-based line number. */
  audio_urls: { [key: string]: string };
  /** The story's sentences, shuffled server-side with the order withheld. */
  sentences: RecallCard[];
  /** Orderings already submitted on earlier visits. */
  attempts: number;
  /** The student finished this phase on an earlier visit. */
  completed: boolean;
}

export interface CheckRecallResult {
  /** Correctness per submitted position. */
  results: boolean[];
  all_correct: boolean;
}

export interface ProduceSlot {
  /** 0-based story line range the segment belongs to (inclusive). */
  line_index: number;
  line_end: number;
  /**
   * True when the reference was found verbatim on one line of the range, so
   * `start`/`end` (code-point offsets, within that line) can be blanked out.
   * False marks every line in the range.
   */
  exact: boolean;
  start: number;
  end: number;
}

export interface ProduceAttemptStartView {
  segment_id: number;
  /** Countdown remaining as of the response. */
  seconds_left: number;
}

export interface ProduceSegmentView {
  id: number;
  segment_order: number;
  english_text: string;
  grammar_point_name?: string;
  /** Where the reference sits in the story text; absent if not found verbatim. */
  slot?: ProduceSlot;
}

export interface ProduceSubmissionView {
  segment_id: number;
  student_text: string;
  reference_hebrew: string;
}

export interface ProduceData {
  story_id: string;
  story_title: string;
  language: string;
  lines: { text: string }[];
  segments: ProduceSegmentView[];
  /** Authored contrastive grammar explanation; empty if none yet. */
  explanation: string;
  /** Attempts already stored on earlier visits, in segment order. */
  submissions: ProduceSubmissionView[];
  /** Segments started but not yet submitted, with time remaining. */
  starts: ProduceAttemptStartView[];
  /** Every segment has a submission. */
  completed: boolean;
  time_limit_seconds: number;
}

export interface SubmitProduceResponse {
  submission: ProduceSubmissionView;
  completed: boolean;
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

export interface APIResponse<T = unknown> {
  success: boolean;
  data?: T;
  error?: string;
}

interface StoriesResponse {
  stories: Story[];
}

export function useApiService() {
  const authenticatedFetch = useAuthenticatedFetch();
  const pendingRequests = useRef<Map<string, Promise<APIResponse<unknown>>>>(
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

      getStoryIdentify: (id: string): Promise<APIResponse<IdentifyData>> => {
        return fetchAPI<IdentifyData>(`/stories/${id}/identify`);
      },

      checkIdentify: (
        id: string,
        lineIndex: number,
        targetVocabId: number,
        selectedTargetVocabId: number,
      ): Promise<APIResponse<{ correct: boolean }>> => {
        return fetchAPI(`/stories/${id}/check-identify`, {
          method: "POST",
          body: JSON.stringify({
            line_index: lineIndex,
            target_vocab_id: targetVocabId,
            selected_target_vocab_id: selectedTargetVocabId,
          }),
        });
      },

      getStoryRecall: (id: string): Promise<APIResponse<RecallData>> => {
        return fetchAPI<RecallData>(`/stories/${id}/recall`);
      },

      checkRecall: (
        id: string,
        orderedSentenceIds: number[],
      ): Promise<APIResponse<CheckRecallResult>> => {
        return fetchAPI(`/stories/${id}/check-recall`, {
          method: "POST",
          body: JSON.stringify({ ordered_sentence_ids: orderedSentenceIds }),
        });
      },

      getStoryProduce: (id: string): Promise<APIResponse<ProduceData>> => {
        return fetchAPI<ProduceData>(`/stories/${id}/produce`);
      },

      startProduce: (
        id: string,
        segmentId: number,
      ): Promise<APIResponse<ProduceAttemptStartView>> => {
        return fetchAPI<ProduceAttemptStartView>(
          `/stories/${id}/produce/start`,
          {
            method: "POST",
            body: JSON.stringify({ segment_id: segmentId }),
          },
        );
      },

      submitProduce: (
        id: string,
        segmentId: number,
        studentText: string,
      ): Promise<APIResponse<SubmitProduceResponse>> => {
        return fetchAPI<SubmitProduceResponse>(`/stories/${id}/produce`, {
          method: "POST",
          body: JSON.stringify({
            segment_id: segmentId,
            student_text: studentText,
          }),
        });
      },

      checkVocab: (
        id: string,
        answers: unknown[],
      ): Promise<APIResponse<unknown>> => {
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

      getStoryScore: (id: string): Promise<APIResponse<unknown>> => {
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
        status?: string,
      ): Promise<APIResponse<unknown>> => {
        const queryParams = status
          ? `?status=${encodeURIComponent(status)}`
          : "";
        return fetchAPI(`/admin/stories/${storyId}/students${queryParams}`);
      },

      resetStudentProgress: (
        storyId: string,
        userId: string,
        phase: ResetPhase,
      ): Promise<APIResponse<ResetProgressResult>> => {
        return fetchAPI<ResetProgressResult>(
          `/admin/stories/${storyId}/students/${encodeURIComponent(userId)}/progress?phase=${phase}`,
          { method: "DELETE" },
        );
      },
    }),
    [fetchAPI],
  );
}
