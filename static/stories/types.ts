interface StoryMetadata {
  storyId: number;
  weekNumber: number;
  dayLetter: string;
  title: {
    [languageCode: string]: string; // ISO 639-1 language codes
  };
  author: {
    id: string;
    name: string;
  };
  grammarPoint: string;
  description: {
    language: string; // ISO 639-1 language code
    text: string;
  };
  lastRevision: string; // ISO-8601 timestamp
}

interface VocabularyItem {
  word: string;
  lexicalForm: string;
  position: [number, number]; // [start, end] character positions
}

interface GrammarItem {
  text: string;
  position: [number, number]; // [start, end] character positions
}

interface Footnote {
  id: number;
  text: string;
  references?: string[];
}

interface StoryLine {
  lineNumber: number;
  text: string;
  vocabulary: VocabularyItem[];
  grammar: GrammarItem[];
  audioFile?: string; // optional, filename only
  footnotes: Footnote[];
}

interface Story {
  metadata: StoryMetadata;
  content: {
    lines: StoryLine[];
  };
}
