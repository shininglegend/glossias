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
  title?: string; // map[string]string
  author: Author;
  grammarPoints: GrammarPoint[];
  description: Description;
}

export interface Author {
  id: string;
  name: string;
}

export interface Description {
  language: string; // 2-letter Unicode language code
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
  languageCode?: string;
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
