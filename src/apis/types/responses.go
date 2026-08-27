package types

import (
	"glossias/src/pkg/models"
)

// APIResponse wraps all API responses with consistent structure
type APIResponse struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

// Story represents a story in API responses
type Story struct {
	ID         int    `json:"id"`
	Title      string `json:"title"`
	WeekNumber int    `json:"week_number"`
	DayLetter  string `json:"day_letter"`
	CourseID   *int   `json:"course_id,omitempty"`
	// MissingPhases lists the Summer 2026 phases ("identify", "produce",
	// "recall") whose content is not fully authored. Populated only for admin
	// callers; omitted when the story is complete.
	MissingPhases []string `json:"missing_phases,omitempty"`
}

// StoriesResponse contains array of stories
type StoriesResponse struct {
	Stories []Story `json:"stories"`
}

// AudioFile represents an audio file in API responses
type AudioFile struct {
	ID         int    `json:"id"`
	FilePath   string `json:"filePath"`
	FileBucket string `json:"fileBucket"`
	Label      string `json:"label"`
}

// TextSegment represents a segment of text in a line
type TextSegment struct {
	Text          string `json:"text"`
	Type          string `json:"type"`                      // "text", "blank", "completed", "target"
	VocabKey      string `json:"vocab_key,omitempty"`       // For blanks: "lineIndex-vocabIndex"
	TargetVocabID int    `json:"target_vocab_id,omitempty"` // For targets: the target_vocabulary row
}

// IdentifyLine is a story line for the Identify phase: its text with target
// words marked, and the target words whose picture quiz opens after it plays.
type IdentifyLine struct {
	Text           []TextSegment `json:"text"`
	TargetVocabIDs []int         `json:"target_vocab_ids"`
}

// IdentifyTargetWord is one of the story's target words with signed asset URLs.
type IdentifyTargetWord struct {
	ID          int    `json:"id"`
	LexicalForm string `json:"lexical_form"`
	AudioURL    string `json:"audio_url,omitempty"`
	ImageURL    string `json:"image_url,omitempty"`
}

// IdentifyPageData is the payload for the Identify phase.
type IdentifyPageData struct {
	PageData
	Lines       []IdentifyLine       `json:"lines"`
	TargetWords []IdentifyTargetWord `json:"target_words"`
	// Signed narration URLs keyed by 1-based line number, as useAudioPlayer expects.
	AudioURLs map[int]string `json:"audio_urls"`
	// CorrectPicks are the (line, word) quizzes the user has already answered
	// correctly, so a reload resumes at the right line and skips them.
	CorrectPicks []IdentifyPick `json:"correct_picks"`
	// Completed is true once every target-word occurrence has a correct pick;
	// the page then shows the finished state instead of replaying.
	Completed bool `json:"completed"`
}

// IdentifyPick is one correctly answered Identify quiz.
type IdentifyPick struct {
	LineIndex     int `json:"line_index"` // 0-based, matching Lines
	TargetVocabID int `json:"target_vocab_id"`
}

// CheckIdentifyRequest is a picture pick in the Identify phase.
type CheckIdentifyRequest struct {
	LineIndex             int `json:"line_index"` // 0-based, matching Lines
	TargetVocabID         int `json:"target_vocab_id"`
	SelectedTargetVocabID int `json:"selected_target_vocab_id"`
}

// CheckIdentifyResponse reports whether the picked picture was the right one.
type CheckIdentifyResponse struct {
	Correct bool `json:"correct"`
}

// Line represents a story line in API responses
type Line struct {
	Text            []string       `json:"text"`
	AudioFiles      []AudioFile    `json:"audio_files"`
	SignedAudioURLs map[int]string `json:"signed_audio_urls,omitempty"`
}

// VocabLine represents a story line with vocabulary segments
type VocabLine struct {
	Text            []TextSegment  `json:"text"`
	AudioFiles      []AudioFile    `json:"audio_files"`
	SignedAudioURLs map[int]string `json:"signed_audio_urls,omitempty"`
}

// LineText represents line text without anything else
type LineText struct {
	Text string `json:"text"`
}

// PageData represents common page data structure
type PageData struct {
	StoryID    string `json:"story_id"`
	StoryTitle string `json:"story_title"`
	Language   string `json:"language"`
}

// AudioPageData extends PageData with lines containing audio
type AudioPageData struct {
	PageData
	Lines []Line `json:"lines"`
}

// VocabPageData extends PageData with vocabulary bank
type VocabPageData struct {
	PageData
	Lines     []VocabLine `json:"lines"`
	VocabBank []string    `json:"vocab_bank"`
}

// GrammarPageData extends PageData with grammar point
type GrammarPageData struct {
	PageData
	Lines              []LineText        `json:"lines"`
	LanguageCode       string            `json:"languageCode"`
	GrammarPointID     int               `json:"grammar_point_id"`
	GrammarPoint       string            `json:"grammar_point"`
	GrammarDescription string            `json:"grammar_description"`
	InstancesCount     int               `json:"instances_count"`
	FoundInstances     []GrammarInstance `json:"found_instances"`
	IncorrectInstances []UserSelection   `json:"incorrect_instances"`
	NextGrammarPoint   *int              `json:"next_grammar_point"`
}

// TranslationPageData extends PageData with the user's translate-phase progress
type TranslationPageData struct {
	PageData
	Lines []LineTranslation `json:"lines"`
	// RequestedLines are the 0-based line indices already translated (saved
	// after each reveal), so a reload can resume.
	RequestedLines []int `json:"requested_lines"`
	// Completed is true once the phase was finished; the page then shows the
	// finished state instead of replaying.
	Completed bool `json:"completed"`
}

// ProduceSegmentView is one Produce segment as shown to the student. The
// reference Hebrew is deliberately absent: it is returned by the submit
// endpoint (or in Submissions for segments already answered) so it cannot be
// read before the attempt is made.
type ProduceSegmentView struct {
	ID               int    `json:"id"`
	SegmentOrder     int    `json:"segment_order"`
	EnglishText      string `json:"english_text"`
	GrammarPointName string `json:"grammar_point_name,omitempty"`
	// Slot locates the reference sentence inside the story text so the page
	// can show the surrounding Hebrew with a blank where the segment goes.
	// Nil when the reference does not appear verbatim in any line.
	Slot *ProduceSlot `json:"slot,omitempty"`
}

// ProduceSlot is a rune range within a 0-based story line.
type ProduceSlot struct {
	LineIndex int `json:"line_index"`
	Start     int `json:"start"`
	End       int `json:"end"`
}

// ProduceSubmissionView is the student's stored attempt at a segment, with the
// reference revealed since the attempt is over.
type ProduceSubmissionView struct {
	SegmentID       int    `json:"segment_id"`
	StudentText     string `json:"student_text"`
	ReferenceHebrew string `json:"reference_hebrew"`
}

// ProducePageData is the payload for the Produce phase.
type ProducePageData struct {
	PageData
	// Lines is the story text, for context around each segment's slot.
	Lines    []LineText           `json:"lines"`
	Segments []ProduceSegmentView `json:"segments"`
	// Explanation is the authored contrastive grammar explanation shown after
	// both segments; empty when none has been authored.
	Explanation string `json:"explanation"`
	// Submissions are the student's latest attempts so far, so a reload
	// resumes at the first unanswered segment.
	Submissions []ProduceSubmissionView `json:"submissions"`
	// Completed is true once every segment has a submission.
	Completed bool `json:"completed"`
	// TimeLimitSeconds is the per-segment writing limit.
	TimeLimitSeconds int `json:"time_limit_seconds"`
}

// SubmitProduceRequest is a student's attempt at one Produce segment. An
// empty StudentText is valid — the timer may have run out first.
type SubmitProduceRequest struct {
	SegmentID   int    `json:"segment_id"`
	StudentText string `json:"student_text"`
}

// SubmitProduceResponse returns the stored attempt with the reference
// revealed, and whether the phase is now complete.
type SubmitProduceResponse struct {
	Submission ProduceSubmissionView `json:"submission"`
	Completed  bool                  `json:"completed"`
}

// SaveTranslationRequest saves the translation to the database
type SaveTranslationRequest struct {
	LineIndexes []int `json:"line_numbers"`
}

// CheckVocabRequest represents the request body for vocab checking
type CheckVocabRequest struct {
	VocabKey string `json:"vocab_key"`
	Answer   string `json:"answer"`
}

// GrammarAnswer represents grammar answer from client
type GrammarAnswer struct {
	LineNumber int   `json:"line_number"`
	Positions  []int `json:"positions"`
}

// CheckGrammarRequest represents the request body for grammar checking
type CheckGrammarRequest struct {
	GrammarPointID int             `json:"grammar_point_id"`
	Answers        []GrammarAnswer `json:"answers"`
}

// CheckSingleGrammarRequest represents the request body for checking a single grammar selection
type CheckSingleGrammarRequest struct {
	GrammarPointID int `json:"grammar_point_id"`
	LineNumber     int `json:"line_number"`
	Position       int `json:"position"`
}

// CheckVocabResponse represents the response for vocab checking
type CheckVocabResponse struct {
	Correct      bool    `json:"correct"`                 // Whether the user's answer is correct for this vocab item
	LineComplete bool    `json:"line_complete"`           // Whether all items on this line have been answered
	OriginalLine *string `json:"original_line,omitempty"` // Original line text when the line is complete
}

// GrammarInstance represents a grammar point instance in the story
type GrammarInstance struct {
	LineNumber int    `json:"line_number"`
	Position   [2]int `json:"position"`
	Text       string `json:"text"`
}

// UserSelection represents a user's selection with correctness
type UserSelection struct {
	LineNumber int    `json:"line_number"`
	Position   [2]int `json:"position"`
	Text       string `json:"text"`
	Correct    bool   `json:"correct"`
}

// CheckGrammarResponse represents the response for grammar checking
type CheckGrammarResponse struct {
	Correct            int               `json:"correct"`
	Wrong              int               `json:"wrong"`
	TotalAnswers       int               `json:"total_answers"`
	GrammarInstances   []GrammarInstance `json:"grammar_instances"`
	UserSelections     []UserSelection   `json:"user_selections"`
	NextGrammarPointID *int              `json:"next_grammar_point_id"`
}

// CheckSingleGrammarResponse represents the response for checking a single grammar selection
type CheckSingleGrammarResponse struct {
	Correct          bool   `json:"correct"`
	MatchedPosition  [2]int `json:"matched_position"`   // Full position range of the match (if correct)
	TotalInstances   int    `json:"total_instances"`    // Total number of instances to find
	NextGrammarPoint *int   `json:"next_grammar_point"` // ID of next grammar point if all found
}

// LineValidationError represents validation error with expected answer counts
type LineValidationError struct {
	Message         string      `json:"message"`
	ExpectedAnswers map[int]int `json:"expected_answers"` // line number -> expected count
}

// LineTranslation extends LineText with translation
type LineTranslation struct {
	LineText
	Translation *string `json:"translation,omitempty"`
	LineNumber  int     `json:"line_number"`
}

// ConvertStoryToAPI converts models.Story to API Story format
func ConvertStoryToAPI(dbStory models.Story) Story {
	if dbStory.Metadata.Title["en"] == "" && dbStory.Metadata.Title[""] != "" {
		dbStory.Metadata.Title["en"] = dbStory.Metadata.Title[""] // "" might hold default title
	}
	return Story{
		ID:         dbStory.Metadata.StoryID,
		Title:      dbStory.Metadata.Title["en"], // Using English title if possible
		WeekNumber: dbStory.Metadata.WeekNumber,
		DayLetter:  dbStory.Metadata.DayLetter,
		CourseID:   dbStory.Metadata.CourseID,
	}
}

// ConvertStoriesToAPI converts slice of models.Story to API format
func ConvertStoriesToAPI(dbStories []models.Story) []Story {
	stories := make([]Story, 0, len(dbStories))
	for _, dbStory := range dbStories {
		stories = append(stories, ConvertStoryToAPI(dbStory))
	}
	return stories
}
