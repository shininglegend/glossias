package models

import (
	"encoding/json"
	"errors"
	"glossias/src/pkg/database"
	"glossias/src/pkg/generated/db"
	"time"

	_ "github.com/lib/pq"
	storage_go "github.com/supabase-community/storage-go"
)

const (
	minTitleLength = 3
)

// Common validation errors
var (
	ErrMissingStoryID    = errors.New("missing story ID")
	ErrInvalidWeekNumber = errors.New("invalid week number")
	ErrMissingDayLetter  = errors.New("missing day letter")
	ErrTitleTooShort     = errors.New("title too short")
	ErrMissingAuthorID   = errors.New("missing author ID")
	ErrNotFound          = errors.New("story not found")
)

var queries *db.Queries
var rawConn any
var storageClient *storage_go.Client

func SetDB(d any) {
	if d == nil {
		panic("database connection is nil")
	}
	rawConn = d
	if conn, ok := d.(db.DBTX); ok {
		queries = db.New(conn)
	} else if mockConn, ok := d.(*database.MockDBTX); ok {
		queries = db.New(mockConn)
	} else {
		// For testing - allow nil queries when no real DB connection
		queries = nil
	}
}

// SetStorageClient initializes the Supabase storage client
func SetStorageClient(url, apiKey string) {
	if url == "" || apiKey == "" {
		storageClient = nil
		return
	}
	storageClient = storage_go.NewClient(url, apiKey, nil)
}

type Story struct {
	Metadata StoryMetadata `json:"metadata"`
	Content  StoryContent  `json:"content"`
}

type StoryMetadata struct {
	StoryID      int               `json:"storyId"`
	WeekNumber   int               `json:"weekNumber"`
	DayLetter    string            `json:"dayLetter"`
	Title        map[string]string `json:"title"` // ISO 639-1 language codes
	Author       Author            `json:"author"`
	VideoURL     string            `json:"videoUrl,omitempty"`
	Description  Description       `json:"description"`
	CourseID     *int              `json:"courseId,omitempty"`
	LastRevision *time.Time        `json:"lastRevision,omitempty"`
}

type Author struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Description struct {
	Language string `json:"language"`
	Text     string `json:"text"`
}

type StoryContent struct {
	Lines []StoryLine `json:"lines"`
}

type StoryLine struct {
	LineNumber         int              `json:"lineNumber"`
	Text               string           `json:"text"`
	EnglishTranslation string           `json:"englishTranslation,omitempty"`
	Vocabulary         []VocabularyItem `json:"vocabulary"`
	Grammar            []GrammarItem    `json:"grammar"`
	AudioFiles         []AudioFile      `json:"audioFiles"`
	Footnotes          []Footnote       `json:"footnotes"`
}

type VocabularyItem struct {
	Word        string `json:"word"`
	LexicalForm string `json:"lexicalForm"`
	Position    [2]int `json:"position"` // Fixed-size array for [start, end]
}

type GrammarItem struct {
	GrammarPointID *int   `json:"grammarPointId,omitempty"`
	Text           string `json:"text"`
	Position       [2]int `json:"position"` // Fixed-size array for [start, end]
}

type Footnote struct {
	ID         int      `json:"id"`
	Text       string   `json:"text"`
	References []string `json:"references,omitempty"` // Optional field
}

// AudioFile represents an audio file attached to a line
type AudioFile struct {
	ID         int    `json:"id"`
	StoryID    int    `json:"storyId"`
	LineNumber int    `json:"lineNumber"`
	FilePath   string `json:"filePath"`
	FileBucket string `json:"fileBucket"`
	Label      string `json:"label"`
}

// GrammarPoint represents a grammar point definition
type GrammarPoint struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// ToJSON serializes a Story to JSON bytes
func (s *Story) ToJSON() ([]byte, error) {
	return json.Marshal(s)
}

// FromJSON deserializes JSON bytes into a Story
func (s *Story) FromJSON(data []byte) error {
	return json.Unmarshal(data, s)
}

// Custom marshaling for StoryMetadata to handle the ISO-8601 timestamp
func (sm StoryMetadata) MarshalJSON() ([]byte, error) {
	type Alias StoryMetadata // Avoid recursive MarshalJSON calls
	return json.Marshal(&struct {
		LastRevision string `json:"lastRevision"`
		*Alias
	}{
		LastRevision: sm.LastRevision.Format(time.RFC3339),
		Alias:        (*Alias)(&sm),
	})
}

// Custom unmarshaling for StoryMetadata to parse the ISO-8601 timestamp
func (sm *StoryMetadata) UnmarshalJSON(data []byte) error {
	type Alias StoryMetadata
	aux := &struct {
		LastRevision string `json:"lastRevision"`
		*Alias
	}{
		Alias: (*Alias)(sm),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.LastRevision == "" {
		return nil // No timestamp provided
	}
	parsedTime, err := time.Parse(time.RFC3339, aux.LastRevision)
	if err != nil {
		return err
	}
	sm.LastRevision = &parsedTime
	return nil
}

// NewStory creates a new Story instance with initialized maps and slices
func NewStory() *Story {
	return &Story{
		Metadata: StoryMetadata{
			Title: make(map[string]string),
		},
		Content: StoryContent{
			Lines: make([]StoryLine, 0),
		},
	}
}

// Validate performs basic validation of the Story struct
func (s *Story) Validate() error {
	if s.Metadata.WeekNumber < 0 {
		return ErrInvalidWeekNumber
	}
	if s.Metadata.DayLetter == "" {
		return ErrMissingDayLetter
	}
	if len(s.Metadata.Title) > minTitleLength {
		return ErrTitleTooShort
	}
	if s.Metadata.Author.ID == "" {
		return ErrMissingAuthorID
	}
	return nil
}
