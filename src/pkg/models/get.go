// story_data.go
package models

import (
	"context"
	"database/sql"
	"fmt"
	"glossias/src/pkg/generated/db"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/lib/pq"
)

func GetStoryData(ctx context.Context, id int) (*Story, error) {
	story := NewStory()

	// Get main story data
	dbStory, err := queries.GetStory(ctx, int32(id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Convert DB story to model story
	story.Metadata.StoryID = int(dbStory.StoryID)
	story.Metadata.WeekNumber = int(dbStory.WeekNumber)
	story.Metadata.DayLetter = dbStory.DayLetter
	if dbStory.GrammarPoint.Valid {
		story.Metadata.GrammarPoint = dbStory.GrammarPoint.String
	}
	if dbStory.LastRevision.Valid {
		story.Metadata.LastRevision = dbStory.LastRevision.Time
	}
	story.Metadata.Author.ID = dbStory.AuthorID
	story.Metadata.Author.Name = dbStory.AuthorName

	// Get titles
	titles, err := queries.GetStoryTitles(ctx, int32(id))
	if err != nil {
		return nil, err
	}
	for _, title := range titles {
		story.Metadata.Title[title.LanguageCode] = title.Title
	}

	// Get description
	storyWithDesc, err := queries.GetStoryWithDescription(ctx, int32(id))
	if err == nil {
		if storyWithDesc.LanguageCode.Valid && storyWithDesc.DescriptionText.Valid {
			story.Metadata.Description.Language = storyWithDesc.LanguageCode.String
			story.Metadata.Description.Text = storyWithDesc.DescriptionText.String
		}
	}

	// Get lines with their components
	lines, err := getStoryLines(ctx, id)
	if err != nil {
		return nil, err
	}
	story.Content.Lines = lines

	return story, nil
}

func getStoryLines(ctx context.Context, storyID int) ([]StoryLine, error) {

	dbLines, err := queries.GetStoryLines(ctx, int32(storyID))
	if err != nil {
		return nil, err
	}

	var lines []StoryLine
	for _, dbLine := range dbLines {
		line := StoryLine{
			LineNumber: int(dbLine.LineNumber),
			Text:       dbLine.Text,
			Vocabulary: []VocabularyItem{}, // Init with empty arrays
			Grammar:    []GrammarItem{},
			Footnotes:  []Footnote{},
		}

		if dbLine.AudioFile.Valid {
			s := dbLine.AudioFile.String
			line.AudioFile = &s
		}

		// Get vocabulary items for this line
		vocabItems, err := queries.GetAllVocabularyForStory(ctx, pgtype.Int4{Int32: int32(storyID), Valid: true})
		if err != nil {
			return nil, err
		}
		for _, vocab := range vocabItems {
			if int(vocab.LineNumber.Int32) == int(dbLine.LineNumber) {
				line.Vocabulary = append(line.Vocabulary, VocabularyItem{
					Word:        vocab.Word,
					LexicalForm: vocab.LexicalForm,
					Position:    [2]int{int(vocab.PositionStart), int(vocab.PositionEnd)},
				})
			}
		}

		// Get grammar items for this line
		grammarItems, err := queries.GetAllGrammarForStory(ctx, pgtype.Int4{Int32: int32(storyID), Valid: true})
		if err != nil {
			return nil, err
		}
		for _, grammar := range grammarItems {
			if int(grammar.LineNumber.Int32) == int(dbLine.LineNumber) {
				line.Grammar = append(line.Grammar, GrammarItem{
					Text:     grammar.Text,
					Position: [2]int{int(grammar.PositionStart), int(grammar.PositionEnd)},
				})
			}
		}

		// Get footnotes for this line
		footnotes, err := queries.GetAllFootnotesForStory(ctx, pgtype.Int4{Int32: int32(storyID), Valid: true})
		if err != nil {
			return nil, err
		}
		for _, fn := range footnotes {
			if int(fn.LineNumber.Int32) == int(dbLine.LineNumber) {
				refs, err := queries.GetFootnoteReferences(ctx, fn.ID)
				if err != nil {
					return nil, err
				}
				line.Footnotes = append(line.Footnotes, Footnote{
					ID:         int(fn.ID),
					Text:       fn.FootnoteText,
					References: refs,
				})
			}
		}

		lines = append(lines, line)
	}
	return lines, nil
}

// GetLineAnnotations retrieves all annotations for a specific line
func GetLineAnnotations(ctx context.Context, storyID int, lineNumber int) (*StoryLine, error) {

	line := &StoryLine{
		LineNumber: lineNumber,
		Vocabulary: []VocabularyItem{}, // init as empty arrays
		Grammar:    []GrammarItem{},
		Footnotes:  []Footnote{},
	}

	// Get vocabulary items for this line
	vocabItems, err := queries.GetAllVocabularyForStory(ctx, pgtype.Int4{Int32: int32(storyID), Valid: true})
	if err != nil {
		return nil, err
	}
	for _, vocab := range vocabItems {
		if int(vocab.LineNumber.Int32) == lineNumber {
			line.Vocabulary = append(line.Vocabulary, VocabularyItem{
				Word:        vocab.Word,
				LexicalForm: vocab.LexicalForm,
				Position:    [2]int{int(vocab.PositionStart), int(vocab.PositionEnd)},
			})
		}
	}

	// Get grammar items for this line
	grammarItems, err := queries.GetAllGrammarForStory(ctx, pgtype.Int4{Int32: int32(storyID), Valid: true})
	if err != nil {
		return nil, err
	}
	for _, grammar := range grammarItems {
		if int(grammar.LineNumber.Int32) == lineNumber {
			line.Grammar = append(line.Grammar, GrammarItem{
				Text:     grammar.Text,
				Position: [2]int{int(grammar.PositionStart), int(grammar.PositionEnd)},
			})
		}
	}

	// Get footnotes for this line
	footnotes, err := queries.GetAllFootnotesForStory(ctx, pgtype.Int4{Int32: int32(storyID), Valid: true})
	if err != nil {
		return nil, err
	}
	for _, fn := range footnotes {
		if int(fn.LineNumber.Int32) == lineNumber {
			refs, err := queries.GetFootnoteReferences(ctx, fn.ID)
			if err != nil {
				return nil, err
			}
			line.Footnotes = append(line.Footnotes, Footnote{
				ID:         int(fn.ID),
				Text:       fn.FootnoteText,
				References: refs,
			})
		}
	}

	return line, nil
}

// GetStoryAnnotations retrieves all annotations for a story grouped by line
func GetStoryAnnotations(ctx context.Context, storyID int) (map[int]*StoryLine, error) {

	// Verify story exists
	exists, err := queries.StoryExists(ctx, int32(storyID))
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrNotFound
	}

	lines := make(map[int]*StoryLine)

	// Get all vocabulary items
	vocabItems, err := queries.GetAllVocabularyForStory(ctx, pgtype.Int4{Int32: int32(storyID), Valid: true})
	if err != nil {
		return nil, err
	}
	for _, vocab := range vocabItems {
		lineNumber := int(vocab.LineNumber.Int32)
		if lines[lineNumber] == nil {
			lines[lineNumber] = &StoryLine{
				LineNumber: lineNumber,
				Vocabulary: []VocabularyItem{},
				Grammar:    []GrammarItem{},
				Footnotes:  []Footnote{},
			}
		}
		lines[lineNumber].Vocabulary = append(lines[lineNumber].Vocabulary, VocabularyItem{
			Word:        vocab.Word,
			LexicalForm: vocab.LexicalForm,
			Position:    [2]int{int(vocab.PositionStart), int(vocab.PositionEnd)},
		})
	}

	// Get all grammar items
	grammarItems, err := queries.GetAllGrammarForStory(ctx, pgtype.Int4{Int32: int32(storyID), Valid: true})
	if err != nil {
		return nil, err
	}
	for _, grammar := range grammarItems {
		lineNumber := int(grammar.LineNumber.Int32)
		if lines[lineNumber] == nil {
			lines[lineNumber] = &StoryLine{
				LineNumber: lineNumber,
				Vocabulary: []VocabularyItem{},
				Grammar:    []GrammarItem{},
				Footnotes:  []Footnote{},
			}
		}
		lines[lineNumber].Grammar = append(lines[lineNumber].Grammar, GrammarItem{
			Text:     grammar.Text,
			Position: [2]int{int(grammar.PositionStart), int(grammar.PositionEnd)},
		})
	}

	// Get all footnotes
	footnotes, err := queries.GetAllFootnotesForStory(ctx, pgtype.Int4{Int32: int32(storyID), Valid: true})
	if err != nil {
		return nil, err
	}
	for _, fn := range footnotes {
		lineNumber := int(fn.LineNumber.Int32)
		if lines[lineNumber] == nil {
			lines[lineNumber] = &StoryLine{
				LineNumber: lineNumber,
				Vocabulary: []VocabularyItem{},
				Grammar:    []GrammarItem{},
				Footnotes:  []Footnote{},
			}
		}
		refs, err := queries.GetFootnoteReferences(ctx, fn.ID)
		if err != nil {
			return nil, err
		}
		lines[lineNumber].Footnotes = append(lines[lineNumber].Footnotes, Footnote{
			ID:         int(fn.ID),
			Text:       fn.FootnoteText,
			References: refs,
		})
	}

	return lines, nil
}

// GetLineText retrieves the text content of a specific line
func GetLineText(ctx context.Context, storyID int, lineNumber int) (string, error) {

	text, err := queries.GetLineText(ctx, db.GetLineTextParams{
		StoryID:    int32(storyID),
		LineNumber: int32(lineNumber),
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return "", ErrInvalidLineNumber
		}
		return "", err
	}
	return text, nil
}

// withTransaction executes a function within a database transaction
func withTransaction(fn func(*sql.Tx) error) error {
	// Check if we're using pgxpool
	if pool, ok := rawConn.(*pgxpool.Pool); ok {
		// Use pgx transaction
		ctx := context.Background()
		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx)

		// Create new queries instance with transaction
		oldQueries := queries
		queries = queries.WithTx(tx)

		// Execute function (tx parameter is ignored for SQLC)
		err = fn(nil)

		// Restore original queries
		queries = oldQueries

		if err != nil {
			return err
		}

		return tx.Commit(ctx)
	}

	// Check if we're using *sql.DB
	if sqlDB, ok := rawConn.(*sql.DB); ok {
		tx, err := sqlDB.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		if err := fn(tx); err != nil {
			return err
		}

		return tx.Commit()
	}

	// Fallback for other connection types
	fmt.Println("# Connection type not recognized. Transactions disabled.")
	return fn(nil)
}

func GetAllStories(ctx context.Context, language string) ([]Story, error) {

	basicStories, err := queries.GetAllStoriesBasic(ctx, language)
	if err != nil {
		return nil, err
	}

	var stories []Story
	for _, basicStory := range basicStories {
		story := Story{
			Metadata: StoryMetadata{
				StoryID:    int(basicStory.StoryID),
				WeekNumber: int(basicStory.WeekNumber),
				DayLetter:  basicStory.DayLetter,
				Title:      map[string]string{language: basicStory.Title},
			},
		}
		stories = append(stories, story)
	}
	return stories, nil
}
