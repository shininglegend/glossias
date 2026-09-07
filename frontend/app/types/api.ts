// [moved from annotator/src/types/api.ts]
// API types for the Admin Annotator. Keep endpoints under /admin.

export interface ApiResponse {
  story: Story;
  success: boolean;
  error?: string;
}

export interface StoryMetadata {
  storyId: number;
  weekNumber: number;
  dayLetter: string;
  title?: string | { [key: string]: string }; // Can be a string or language map
  author?: Author;
  grammarPoints?: GrammarPoint[];
  description?: Description;
  videoUrl?: string;
  languageCode?: string; // 2-letter Unicode language code
}

export interface Author {
  id: string;
  name: string;
}

export interface Description {
  text: string;
}

export interface Story {
  metadata: StoryMetadata;
  content: StoryContent;
}

export interface StoryContent {
  lines: StoryLine[];
}

export interface StoryLine {
  lineNumber: number;
  text: string;
  vocabulary: VocabularyItem[];
  grammar: GrammarItem[];
  footnotes: Footnote[];
  audioFiles: AudioFile[];
  storyId?: number;
}

export interface AudioFile {
  id: number;
  filePath: string;
  fileBucket: string;
  label: string;
}

export interface VocabularyItem {
  word: string;
  lexicalForm: string;
  position: [number, number];
}

export interface GrammarItem {
  grammarPointId?: number;
  text: string;
  position: [number, number];
}

export interface GrammarPoint {
  id: number;
  name: string;
  description?: string;
}

export interface Footnote {
  id: number;
  text: string;
  references?: string[];
}

export interface AnnotationRequest {
  lineNumber: number;
  vocabulary?: VocabularyItem;
  grammar?: GrammarItem;
  footnote?: Footnote;
}

export type AnnotationType = "vocab" | "grammar" | "footnote";

export interface ApiError {
  error: string;
}

export interface NavigationGuidanceRequest {
  storyId: string;
  userId: string;
  currentPage: PageType;
}

export interface NavigationGuidanceResponse {
  nextPage: PageType;
  displayName: string;
  /** Archived runs of the story; finishing one clears the live progress. */
  completedAttempts: number;
}

/**
 * Scope of an admin progress reset. Everything but "all" acts on the
 * student's current attempt; "all" also deletes archived attempts.
 */
export type ResetPhase =
  | "all"
  | "exercises"
  | "video"
  | "identify"
  | "translate"
  | "produce"
  | "recall"
  | "vocab"
  | "grammar";

export interface ResetProgressResult {
  phase: ResetPhase;
  /** Rows removed, keyed by table name plus "time_tracking". */
  deleted: Record<string, number>;
}

/** One phase of a student's live attempt that holds answers or time. */
export interface AttemptStage {
  phase: ResetPhase;
  detail: string;
  seconds: number;
}

/**
 * Mirrors models.AttemptSummary: an archived attempt with its frozen score,
 * or the live current attempt (is_current) with its deletable stages.
 */
export interface StudentAttemptSummary {
  number: number;
  is_current: boolean;
  completed_at?: string;
  score?: {
    overall_accuracy: number;
    total_time_seconds: number;
    produce_segments_submitted: number;
    produce_segments_graded: number;
  };
  /** Only on the current attempt; empty means not started. */
  stages?: AttemptStage[];
}

export type PageType =
  | "list"
  | "video"
  | "vocab"
  | "identify"
  | "translate"
  | "produce"
  | "recall"
  | "grammar"
  | "score";

// TextSegment represents a segment of text in a vocab line
export interface TextSegment {
  text: string;
  type: "text" | "blank" | "completed" | "target";
  vocab_key?: string; // For blanks: "lineIndex-vocabIndex"
  target_vocab_id?: number; // For targets: the target_vocabulary row
}

// VocabLine represents a story line with vocabulary segments
export interface VocabLine {
  text: TextSegment[];
  audio_files: AudioFile[];
  signed_audio_urls?: { [key: number]: string };
}

export const createAnnotationRequest = (
  lineNumber: number,
  type: AnnotationType,
  text: string,
  start: number,
  end: number,
  data?: { text?: string; lexicalForm?: string; grammarPointId?: number },
): AnnotationRequest => {
  const request: AnnotationRequest = {
    lineNumber,
  };

  switch (type) {
    case "vocab":
      request.vocabulary = {
        word: text,
        lexicalForm: data?.lexicalForm || "",
        position: [start, end],
      };
      break;
    case "grammar":
      request.grammar = {
        text,
        position: [start, end],
        grammarPointId: data?.grammarPointId,
      };
      break;
    case "footnote":
      request.footnote = {
        id: 0,
        text: data?.text || "",
        references: [text],
      };
      break;
    default:
      throw new Error(`Invalid annotation type: ${type}`);
  }

  return request;
};
