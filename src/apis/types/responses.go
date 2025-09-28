package types

import (
	"glossias/src/pkg/models"
)

// APIResponse wraps all API responses with consistent structure
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// Story represents a story in API responses
type Story struct {
	ID         int    `json:"id"`
	Title      string `json:"title"`
	WeekNumber int    `json:"week_number"`
	DayLetter  string `json:"day_letter"`
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

// Line represents a story line in API responses
type Line struct {
	Text            []string       `json:"text"`
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
	Lines     []Line   `json:"lines"`
	VocabBank []string `json:"vocab_bank"`
}

// GrammarPageData extends PageData with grammar point
type GrammarPageData struct {
	PageData
	Lines              []LineText `json:"lines"`
	LanguageCode       string     `json:"languageCode"`
	GrammarPointID     int        `json:"grammar_point_id"`
	GrammarPoint       string     `json:"grammar_point"`
	GrammarDescription string     `json:"grammar_description"`
	InstancesCount     int        `json:"instances_count"`
}

// TranslationPageData extends PageData with translation field
type TranslationPageData struct {
	PageData
	Lines []LineTranslation `json:"lines"`
}

// VocabAnswer represents vocabulary answer from client
type VocabAnswer struct {
	LineNumber int      `json:"line_number"`
	Answers    []string `json:"answers"`
}

// CheckVocabRequest represents the request body for vocab checking
type CheckVocabRequest struct {
	Answers []VocabAnswer `json:"answers"`
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

// VocabResult represents individual vocabulary check result
type VocabResult struct {
	Correct       bool   `json:"correct"`
	UserAnswer    string `json:"user_answer,omitempty"`
	CorrectAnswer string `json:"correct_answer,omitempty"`
	Line          int    `json:"line"`
}

// CheckVocabResponse represents the response for vocab checking
type CheckVocabResponse struct {
	Correct bool `json:"correct"`
}

// GrammarInstance represents a grammar point instance in the story
type GrammarInstance struct {
	LineNumber   int    `json:"line_number"`
	Position     [2]int `json:"position"`
	Text         string `json:"text"`
	UserSelected bool   `json:"user_selected"`
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

// LineValidationError represents validation error with expected answer counts
type LineValidationError struct {
	Message         string      `json:"message"`
	ExpectedAnswers map[int]int `json:"expected_answers"` // line number -> expected count
}

// LineTranslation extends LineText with translation
type LineTranslation struct {
	LineText
	Translation *string `json:"translation,omitempty"`
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
