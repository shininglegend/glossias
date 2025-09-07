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

// Line represents a story line in API responses
type Line struct {
	Text              []string `json:"text"`
	AudioURL          *string  `json:"audio_url,omitempty"`
	HasVocabOrGrammar bool     `json:"has_vocab_or_grammar"`
}

// PageData represents common page data structure
type PageData struct {
	StoryID    string `json:"story_id"`
	StoryTitle string `json:"story_title"`
	Lines      []Line `json:"lines"`
}

// Page2Data extends PageData with vocabulary bank
type Page2Data struct {
	PageData
	VocabBank []string `json:"vocab_bank"`
}

// Page3Data extends PageData with grammar point
type Page3Data struct {
	PageData
	GrammarPoint string `json:"grammar_point"`
}

// Page3Data extends PageData with  translation
type Page4Data struct {
	PageData
	Translation string `json:"translation"`
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

// VocabResult represents individual vocabulary check result
type VocabResult struct {
	Correct       bool   `json:"correct"`
	UserAnswer    string `json:"user_answer,omitempty"`
	CorrectAnswer string `json:"correct_answer,omitempty"`
	Line          int    `json:"line"`
}

// CheckVocabResponse represents the response for vocab checking
type CheckVocabResponse struct {
	Answers []VocabResult `json:"answers"`
}

// LineValidationError represents validation error with expected answer counts
type LineValidationError struct {
	Message         string      `json:"message"`
	ExpectedAnswers map[int]int `json:"expected_answers"` // line number -> expected count
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
