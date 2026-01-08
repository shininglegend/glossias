package models

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"glossias/src/pkg/cache"
	"glossias/src/pkg/database"
	"glossias/src/pkg/generated/db"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	_ "github.com/lib/pq"
	storage_go "github.com/supabase-community/storage-go"
)

const (
	minTitleLength = 3
)

// Common validation errors
var (
	ErrMissingStoryID       = errors.New("missing story ID")
	ErrInvalidWeekNumber    = errors.New("invalid week number")
	ErrMissingDayLetter     = errors.New("missing day letter")
	ErrTitleTooShort        = errors.New("title too short")
	ErrMissingAuthorID      = errors.New("missing author ID")
	ErrMissingGrammarPoints = errors.New("at least one grammar point is required")
	ErrNotFound             = errors.New("story not found")
)

var queries *db.Queries
var rawConn any
var storageClient *storage_go.Client
var storageBaseURL string
var storageAPIKey string
var cacheInstance *cache.Cache
var keyBuilder *cache.KeyBuilder

func SetDB(d any) {
	if d == nil {
		panic("database connection is nil")
	}
	rawConn = d
	if conn, ok := d.(db.DBTX); ok {
		queries = db.New(conn)
	} else if mockConn, ok := d.(*database.MockDBTX); ok {
		queries = db.New(mockConn)
	} else if reconnectConn, ok := d.(*database.ReconnectableDBTX); ok {
		// ReconnectableDBTX implements DBTX with reconnection logic
		queries = db.New(reconnectConn)
	} else {
		// For testing - allow nil queries when no real DB connection
		queries = nil
	}
}

// SetStorageClient initializes the Supabase storage client
func SetStorageClient(url, apiKey string) {
	if url == "" || apiKey == "" {
		fmt.Println("Storage credentials missing - operations will fail")
		storageClient = nil
		storageBaseURL = ""
		storageAPIKey = ""
		return
	}

	storageClient = storage_go.NewClient(url, apiKey, nil)
	storageBaseURL = url
	storageAPIKey = apiKey
	// Test the connection by listing buckets with retry on 'tx closed'
	const maxRetries = 3
	var err error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		_, err = storageClient.ListBuckets()
		if err == nil {
			break
		}
		if database.IsConnectionError(err) {
			fmt.Printf("Attempt %d: tx closed error received, reconnecting...\n", attempt)
			storageClient = storage_go.NewClient(url, apiKey, nil)
			time.Sleep(1 * time.Second)
		} else {
			break
		}
	}
	if err != nil {
		fmt.Printf("Failed to connect to storage: %v\n", err)
		storageClient = nil
		storageBaseURL = ""
		storageAPIKey = ""
		return
	}
	fmt.Printf("Storage client initialized with URL: %s\n", url)
}

// TestDBConnection tests the database connection with minimal query
func TestDBConnection(ctx context.Context) error {
	if rawConn == nil {
		return errors.New("database not initialized")
	}
	
	// Use the simplest possible query to test connection
	if conn, ok := rawConn.(db.DBTX); ok {
		var result int
		err := conn.QueryRow(ctx, "SELECT 1").Scan(&result)
		return err
	}
	
	return errors.New("unable to test database connection")
}

// SetCache initializes the cache instance
func SetCache() error {
	cacheConfig := cache.DefaultConfig()
	var err error
	cacheInstance, err = cache.New(cacheConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize cache: %w", err)
	}
	keyBuilder = cache.NewKeyBuilder()
	fmt.Println("Cache initialized successfully")
	return nil
}

// storageRetry executes a storage operation with retry on connection errors
func storageRetry(operation func() error) error {
	const maxRetries = 3
	var err error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		err = operation()
		if err == nil {
			return nil
		}

		// Check if it's a connection error
		if database.IsConnectionError(err) {

			fmt.Printf("Storage connection error (attempt %d/%d): %v\n", attempt, maxRetries, err)

			if attempt < maxRetries {
				time.Sleep(1 * time.Second)
				// Reinitialize storage client
				if storageBaseURL != "" && storageAPIKey != "" {
					storageClient = storage_go.NewClient(storageBaseURL, storageAPIKey, nil)
				}
			}
		} else {
			// Non-connection error, don't retry
			return err
		}
	}

	return fmt.Errorf("storage operation failed after %d attempts: %w", maxRetries, err)
}

type Story struct {
	Metadata StoryMetadata `json:"metadata"`
	Content  StoryContent  `json:"content"`
}

type StoryMetadata struct {
	StoryID       int               `json:"storyId"`
	WeekNumber    int               `json:"weekNumber"`
	DayLetter     string            `json:"dayLetter"`
	Title         map[string]string `json:"title"` // ISO 639-1 language codes
	Author        Author            `json:"author"`
	VideoURL      string            `json:"videoUrl,omitempty"`
	Description   Description       `json:"description"`
	CourseID      *int              `json:"courseId,omitempty"`
	LastRevision  *time.Time        `json:"lastRevision,omitempty"`
	GrammarPoints []GrammarPoint    `json:"grammarPoints"`
	Language      string            `json:"languageCode,omitempty"`
}

type Author struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Description struct {
	Text string `json:"text"`
}

type StoryContent struct {
	Lines []StoryLine `json:"lines"`
}

type StoryLine struct {
	LineNumber int              `json:"lineNumber"`
	Text       string           `json:"text"`
	Vocabulary []VocabularyItem `json:"vocabulary"`
	Grammar    []GrammarItem    `json:"grammar"`
	AudioFiles []AudioFile      `json:"audioFiles"`
	Footnotes  []Footnote       `json:"footnotes"`
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
	StoryID     int    `json:"story_id"`
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
	var lastRevision string
	if sm.LastRevision != nil {
		lastRevision = sm.LastRevision.Format(time.RFC3339)
	}
	return json.Marshal(&struct {
		LastRevision string `json:"lastRevision"`
		*Alias
	}{
		LastRevision: lastRevision,
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
			Title:         make(map[string]string),
			GrammarPoints: make([]GrammarPoint, 0),
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
	if len(s.Metadata.GrammarPoints) == 0 {
		return ErrMissingGrammarPoints
	}
	return nil
}

// convertToPGInt converts various integer types to pgtype.Int4
func convertToPGInt(value any) pgtype.Int4 {
	switch v := value.(type) {
	case int:
		return pgtype.Int4{Int32: int32(v), Valid: true}
	case int32:
		return pgtype.Int4{Int32: v, Valid: true}
	case int64:
		return pgtype.Int4{Int32: int32(v), Valid: true}
	case nil:
		return pgtype.Int4{Valid: false}
	default:
		return pgtype.Int4{Valid: false}
	}
}
