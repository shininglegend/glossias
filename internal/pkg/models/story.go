package models

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"glossias/internal/pkg/database"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/lib/pq"
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

var store database.Store

func SetDB(d any) {
	if d == nil {
		panic("database connection is nil")
	}
	if pool, ok := d.(*pgxpool.Pool); ok {
		store = &poolWrapper{pool}
	} else if s, ok := d.(database.Store); ok {
		store = s
	} else {
		panic("unsupported database type")
	}
}

// poolWrapper wraps pgxpool.Pool to implement database.Store interface
type poolWrapper struct {
	pool *pgxpool.Pool
}

func (w *poolWrapper) DB() database.DB {
	return &poolDB{w.pool}
}

func (w *poolWrapper) Close() error {
	w.pool.Close()
	return nil
}

// poolDB wraps pgxpool.Pool to implement database.DB interface
type poolDB struct {
	pool *pgxpool.Pool
}

func (db *poolDB) Exec(query string, args ...any) (sql.Result, error) {
	tag, err := db.pool.Exec(context.Background(), query, args...)
	if err != nil {
		return nil, err
	}
	return &poolResult{tag}, nil
}

func (db *poolDB) Query(query string, args ...any) (database.Rows, error) {
	rows, err := db.pool.Query(context.Background(), query, args...)
	if err != nil {
		return nil, err
	}
	return &poolRows{rows}, nil
}

func (db *poolDB) QueryRow(query string, args ...any) database.Row {
	return &poolRow{db.pool.QueryRow(context.Background(), query, args...)}
}

func (db *poolDB) Begin() (*sql.Tx, error) {
	return nil, errors.New("transactions not implemented for pgxpool")
}

func (db *poolDB) Close() error {
	db.pool.Close()
	return nil
}

func (db *poolDB) Ping() error {
	return db.pool.Ping(context.Background())
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
	GrammarPoint string            `json:"grammarPoint"`
	Description  Description       `json:"description"`
	LastRevision time.Time         `json:"lastRevision,omitempty"`
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
	LineNumber int              `json:"lineNumber"`
	Text       string           `json:"text"`
	Vocabulary []VocabularyItem `json:"vocabulary"`
	Grammar    []GrammarItem    `json:"grammar"`
	AudioFile  *string          `json:"audioFile,omitempty"` // Using pointer for optional field
	Footnotes  []Footnote       `json:"footnotes"`
}

type VocabularyItem struct {
	Word        string `json:"word"`
	LexicalForm string `json:"lexicalForm"`
	Position    [2]int `json:"position"` // Fixed-size array for [start, end]
}

type GrammarItem struct {
	Text     string `json:"text"`
	Position [2]int `json:"position"` // Fixed-size array for [start, end]
}

type Footnote struct {
	ID         int      `json:"id"`
	Text       string   `json:"text"`
	References []string `json:"references,omitempty"` // Optional field
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

	parsedTime, err := time.Parse(time.RFC3339, aux.LastRevision)
	if err != nil {
		return err
	}
	sm.LastRevision = parsedTime
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

// poolResult wraps pgx result
type poolResult struct {
	tag any
}

func (r *poolResult) LastInsertId() (int64, error) {
	return 0, errors.New("LastInsertId not supported by pgx")
}

func (r *poolResult) RowsAffected() (int64, error) {
	if tag, ok := r.tag.(interface{ RowsAffected() int64 }); ok {
		return tag.RowsAffected(), nil
	}
	return 0, nil
}

// poolRows wraps pgx rows
type poolRows struct {
	rows any
}

func (r *poolRows) Close() error {
	if rows, ok := r.rows.(interface{ Close() }); ok {
		rows.Close()
	}
	return nil
}

func (r *poolRows) Next() bool {
	if rows, ok := r.rows.(interface{ Next() bool }); ok {
		return rows.Next()
	}
	return false
}

func (r *poolRows) Scan(dest ...any) error {
	if rows, ok := r.rows.(interface{ Scan(dest ...any) error }); ok {
		return rows.Scan(dest...)
	}
	return errors.New("scan not supported")
}

func (r *poolRows) Columns() ([]string, error) {
	return nil, errors.New("columns not implemented for pgx")
}

// poolRow wraps pgx row
type poolRow struct {
	row any
}

func (r *poolRow) Scan(dest ...any) error {
	if row, ok := r.row.(interface{ Scan(dest ...any) error }); ok {
		return row.Scan(dest...)
	}
	return errors.New("scan not supported")
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
