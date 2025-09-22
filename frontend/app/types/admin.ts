// Types mirroring backend models for admin editors

export interface Story {
  metadata: StoryMetadata;
  content: StoryContent;
}

export interface StoryMetadata {
  storyId: number;
  weekNumber: number;
  dayLetter: string; // a-e
  title: Record<string, string>; // ISO 639-1 -> title
  author: Author;
  grammarPoints: GrammarPoint[];
  description: Description;
  courseId?: number;
  videoUrl?: string;
  lastRevision?: string; // RFC3339 string required by backend on update
}

export interface Author {
  id: string;
  name: string;
}

export interface Description {
  language: string;
  text: string;
}

export interface StoryContent {
  lines: StoryLine[];
}

export interface StoryLine {
  lineNumber: number;
  text: string;
  vocabulary: VocabularyItem[];
  grammar: GrammarItem[];
  audioFile?: string | null;
  footnotes: Footnote[];
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
