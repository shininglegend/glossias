// src/types/api.ts

// Main API response interfaces
export interface ApiResponse {
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
  audioFile?: string;
}

// Annotation items
export interface VocabularyItem {
  word: string;
  lexicalForm: string;
  position: [number, number]; // [start, end]
}

export interface GrammarItem {
  text: string;
  position: [number, number]; // [start, end]
}

export interface Footnote {
  id: number;
  text: string;
  references?: string[];
}

// API request interfaces
export interface AnnotationRequest {
  lineNumber: number;
  vocabulary?: VocabularyItem;
  grammar?: GrammarItem;
  footnote?: Footnote;
}

// Helper type for annotation types
export type AnnotationType = "vocab" | "grammar" | "footnote";

// API error response
export interface ApiError {
  error: string;
}

// Ensure request matches your Go struct
export const createAnnotationRequest = (
  lineNumber: number,
  type: AnnotationType,
  text: string,
  start: number,
  end: number,
  data?: { text?: string; lexicalForm?: string },
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
      };
      break;
    case "footnote":
      request.footnote = {
        id: 0, // Server will assign
        text: data?.text || "",
        references: [text],
      };
      break;
    default:
      throw new Error(`Invalid annotation type: ${type}`);
  }

  return request;
};
