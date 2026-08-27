// Admin API client aligned to backend routes under /admin/stories

import { useCallback } from "react";
import { useAuthenticatedFetch } from "../lib/authFetch";
import type {
  Story,
  StoryMetadata,
  TargetVocabulary,
  TargetVocabularyPage,
  ProducePage,
  ProduceSegment,
  RecallPage,
  RecallSentence,
  StoryContentReadiness,
} from "../types/admin";

type Json<T> = Promise<T>;

// Cache for pending requests to prevent duplicates
const pendingRequests = new Map<string, Promise<unknown>>();

export function useAdminApi() {
  const authenticatedFetch = useAuthenticatedFetch();

  const request = useCallback(
    async <T>(path: string, init?: RequestInit, baseUrl?: string): Json<T> => {
      const url = baseUrl ? `${baseUrl}/api/admin${path}` : `/api/admin${path}`;
      const method = init?.method || "GET";
      const cacheKey = `${method}:${url}`;

      // For GET requests, check if there's already a pending request
      if (method === "GET" && pendingRequests.has(cacheKey)) {
        return pendingRequests.get(cacheKey) as Promise<T>;
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

  // The phase-authoring deletes answer 204 No Content, which request() cannot
  // parse as JSON.
  const requestNoContent = useCallback(
    async (
      path: string,
      init?: RequestInit,
      baseUrl?: string,
    ): Promise<void> => {
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
    },
    [authenticatedFetch],
  );

  return {
    // GET stories/:id -> { Story, Success }
    getStoryForEdit: useCallback(
      async (id: number, baseUrl?: string): Json<Story | undefined> => {
        const data = await request<{ story: Story }>(
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
        const data = await request<{ story: Story; success: boolean }>(
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

    // GET /stories/:id -> { story: { metadata, content } }
    getStoryContent: useCallback(
      async (id: number, baseUrl?: string): Json<{ story: Story }> => {
        return request<{ story: Story }>(`/stories/${id}`, undefined, baseUrl);
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

    // Summer 2026 phase authoring (T7).
    //
    // Assets are uploaded with usePhaseAssetUploader (lib/phaseAssets.ts), which
    // returns a storage path; the path is then attached here by saving the target
    // word or recall sentence that owns it. Passing an empty path clears the
    // asset and deletes the stored file; omitting the field leaves it alone.

    // GET /stories/:id/content-readiness
    getContentReadiness: useCallback(
      async (id: number, baseUrl?: string): Json<StoryContentReadiness> => {
        return request<StoryContentReadiness>(
          `/stories/${id}/content-readiness`,
          undefined,
          baseUrl,
        );
      },
      [request],
    ),

    // GET /stories/:id/target-vocabulary
    getTargetVocabulary: useCallback(
      async (id: number, baseUrl?: string): Json<TargetVocabularyPage> => {
        return request<TargetVocabularyPage>(
          `/stories/${id}/target-vocabulary`,
          undefined,
          baseUrl,
        );
      },
      [request],
    ),

    // POST /stories/:id/target-vocabulary
    addTargetWord: useCallback(
      async (
        id: number,
        lexicalForm: string,
        baseUrl?: string,
      ): Json<TargetVocabulary> => {
        return request<TargetVocabulary>(
          `/stories/${id}/target-vocabulary`,
          { method: "POST", body: JSON.stringify({ lexicalForm }) },
          baseUrl,
        );
      },
      [request],
    ),

    // PUT /stories/:id/target-vocabulary/:wordId
    saveTargetWord: useCallback(
      async (
        id: number,
        wordId: number,
        changes: {
          lexicalForm?: string;
          audioPath?: string;
          imagePath?: string;
        },
        baseUrl?: string,
      ): Json<TargetVocabulary> => {
        return request<TargetVocabulary>(
          `/stories/${id}/target-vocabulary/${wordId}`,
          { method: "PUT", body: JSON.stringify(changes) },
          baseUrl,
        );
      },
      [request],
    ),

    // DELETE /stories/:id/target-vocabulary/:wordId
    deleteTargetWord: useCallback(
      async (id: number, wordId: number, baseUrl?: string): Promise<void> => {
        await requestNoContent(
          `/stories/${id}/target-vocabulary/${wordId}`,
          { method: "DELETE" },
          baseUrl,
        );
      },
      [requestNoContent],
    ),

    // GET /stories/:id/produce
    getProduce: useCallback(
      async (id: number, baseUrl?: string): Json<ProducePage> => {
        return request<ProducePage>(
          `/stories/${id}/produce`,
          undefined,
          baseUrl,
        );
      },
      [request],
    ),

    // PUT /stories/:id/produce/segments/:order
    saveProduceSegment: useCallback(
      async (
        id: number,
        order: number,
        segment: {
          englishText: string;
          referenceHebrew: string;
          grammarPointId?: number;
        },
        baseUrl?: string,
      ): Json<ProduceSegment> => {
        return request<ProduceSegment>(
          `/stories/${id}/produce/segments/${order}`,
          { method: "PUT", body: JSON.stringify(segment) },
          baseUrl,
        );
      },
      [request],
    ),

    // DELETE /stories/:id/produce/segments/:order
    deleteProduceSegment: useCallback(
      async (id: number, order: number, baseUrl?: string): Promise<void> => {
        await requestNoContent(
          `/stories/${id}/produce/segments/${order}`,
          { method: "DELETE" },
          baseUrl,
        );
      },
      [requestNoContent],
    ),

    // PUT /stories/:id/produce/explanation
    saveProduceExplanation: useCallback(
      async (
        id: number,
        explanation: string,
        baseUrl?: string,
      ): Json<{ explanation: string }> => {
        return request<{ explanation: string }>(
          `/stories/${id}/produce/explanation`,
          { method: "PUT", body: JSON.stringify({ explanation }) },
          baseUrl,
        );
      },
      [request],
    ),

    // DELETE /stories/:id/produce/explanation
    deleteProduceExplanation: useCallback(
      async (id: number, baseUrl?: string): Promise<void> => {
        await requestNoContent(
          `/stories/${id}/produce/explanation`,
          { method: "DELETE" },
          baseUrl,
        );
      },
      [requestNoContent],
    ),

    // GET /stories/:id/recall
    getRecall: useCallback(
      async (id: number, baseUrl?: string): Json<RecallPage> => {
        return request<RecallPage>(`/stories/${id}/recall`, undefined, baseUrl);
      },
      [request],
    ),

    // PUT /stories/:id/recall/sentences/:order
    saveRecallSentence: useCallback(
      async (
        id: number,
        order: number,
        sentence: {
          hebrewText: string;
          targetVocabId?: number;
          imagePath?: string;
        },
        baseUrl?: string,
      ): Json<RecallSentence> => {
        return request<RecallSentence>(
          `/stories/${id}/recall/sentences/${order}`,
          { method: "PUT", body: JSON.stringify(sentence) },
          baseUrl,
        );
      },
      [request],
    ),

    // DELETE /stories/:id/recall/sentences/:order
    deleteRecallSentence: useCallback(
      async (id: number, order: number, baseUrl?: string): Promise<void> => {
        await requestNoContent(
          `/stories/${id}/recall/sentences/${order}`,
          { method: "DELETE" },
          baseUrl,
        );
      },
      [requestNoContent],
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
