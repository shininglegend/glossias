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
  languageCode?: string;
  courseId?: number;
  videoUrl?: string;
  lastRevision?: string; // RFC3339 string required by backend on update
}

export interface Author {
  id: string;
  name: string;
}

export interface Description {
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

// Summer 2026 phase authoring (T7).
//
// The path/bucket pairs on a target word or recall sentence are the source of
// truth for which asset belongs to it; the *Url fields are short-lived signed
// read URLs the backend adds for preview and are not sent back on save.

export interface TargetVocabulary {
  id: number;
  storyId: number;
  lexicalForm: string;
  audioPath?: string;
  audioBucket?: string;
  correctImagePath?: string;
  imageBucket?: string;
  audioUrl?: string;
  imageUrl?: string;
}

export interface LexicalFormCount {
  lexicalForm: string;
  occurrences: number;
}

export interface ContentIssue {
  field?: string;
  message: string;
}

export interface PhaseReadiness {
  phase: string;
  ready: boolean;
  issues: ContentIssue[];
}

export interface StoryContentReadiness {
  video: PhaseReadiness;
  identify: PhaseReadiness;
  produce: PhaseReadiness;
  recall: PhaseReadiness;
}

export interface TargetVocabularyPage {
  words: TargetVocabulary[];
  candidates: LexicalFormCount[];
  readiness: PhaseReadiness;
  required: number;
  minOccurrences: number;
}

export interface ProduceSegment {
  id: number;
  storyId: number;
  segmentOrder: number;
  englishText: string;
  referenceHebrew: string;
  grammarPointId?: number;
  grammarPointName?: string;
}

export interface ProducePage {
  segments: ProduceSegment[];
  explanation: string;
  grammarPoints: GrammarPoint[];
  readiness: PhaseReadiness;
  required: number;
}

export interface RecallSentence {
  id: number;
  storyId: number;
  sequenceOrder: number;
  hebrewText: string;
  targetVocabId?: number;
  imagePath?: string;
  imageBucket?: string;
  imageUrl?: string;
}

export interface RecallPage {
  sentences: RecallSentence[];
  targetVocabulary: TargetVocabulary[];
  readiness: PhaseReadiness;
  required: number;
}

export type PhaseAssetKind =
  "target_vocab_image" | "target_vocab_audio" | "recall_image";
