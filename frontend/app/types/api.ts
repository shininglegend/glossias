// [moved from annotator/src/types/api.ts]
// API types for the Admin Annotator. Keep endpoints under /admin.

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

export interface VocabularyItem {
  word: string;
  lexicalForm: string;
  position: [number, number];
}

export interface GrammarItem {
  text: string;
  position: [number, number];
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


