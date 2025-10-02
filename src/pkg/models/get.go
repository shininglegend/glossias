// story_data.go
package models

import (
	"context"
	"database/sql"
	"fmt"
	"glossias/src/pkg/generated/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/lib/pq"
)

// checkUserAccessWithCache checks if a user has access to a story with caching
func checkUserAccessWithCache(userID string, storyID int, checkFunc func() (bool, error)) (bool, error) {
	if cacheInstance == nil || keyBuilder == nil {
		// Fallback to direct check if cache not available
		return checkFunc()
	}

	cacheKey := keyBuilder.UserAccess(userID, storyID)

	// Try to get from cache first
	if data, err := cacheInstance.Get(cacheKey); err == nil {
		// Cache hit - return cached access result
		return string(data) == "true", nil
	}

	// Cache miss - check access and cache result
	hasAccess, err := checkFunc()
	if err != nil {
		return false, err
	}

	// Cache the result (ignore cache errors)
	accessStr := "false"
	if hasAccess {
		accessStr = "true"
	}
	_ = cacheInstance.Set(cacheKey, []byte(accessStr))

	return hasAccess, nil
}

func GetStoryData(ctx context.Context, id int, userID string) (*Story, error) {
	// Check user access first (with caching)
	if cacheInstance != nil && keyBuilder != nil {
		hasAccess, err := checkUserAccessWithCache(userID, id, func() (bool, error) {
			// Check if user has access to this story
			dbStory, err := queries.GetStory(ctx, int32(id))
			if err != nil {
				if err == sql.ErrNoRows || err == pgx.ErrNoRows {
					return false, ErrNotFound
				}
				return false, err
			}

			// Check course access if story has course
			if dbStory.CourseID.Valid {
				courseID := int32(dbStory.CourseID.Int32)
				return CanUserAccessCourse(ctx, userID, courseID), nil
			}
			return true, nil // No course restriction
		})
		if err != nil {
			return nil, err
		}
		if !hasAccess {
			return nil, ErrNotFound
		}
	} else {
		// Fallback to direct access check
		dbStory, err := queries.GetStory(ctx, int32(id))
		if err != nil {
			if err == sql.ErrNoRows || err == pgx.ErrNoRows {
				return nil, ErrNotFound
			}
			return nil, err
		}

		// Check if user has access to this story
		if dbStory.CourseID.Valid {
			courseID := int32(dbStory.CourseID.Int32)
			if !CanUserAccessCourse(ctx, userID, courseID) {
				return nil, ErrNotFound
			}
		}
	}

	// Try cache for story data (no user ID in key)
	if cacheInstance != nil && keyBuilder != nil {
		cacheKey := keyBuilder.StoryData(id)
		var story Story
		err := cacheInstance.GetOrSetJSON(cacheKey, &story, func() (any, error) {
			return getStoryDataFromDB(ctx, id, userID)
		})
		if err != nil {
			return nil, err
		}
		return &story, nil
	}

	// Fallback to direct DB access
	return getStoryDataFromDB(ctx, id, userID)
}

// getStoryDataFromDB performs the actual database operations for GetStoryData
func getStoryDataFromDB(ctx context.Context, id int, userID string) (*Story, error) {
	story := NewStory()

	// Get main story data
	dbStory, err := queries.GetStory(ctx, int32(id))
	if err != nil {
		if err == sql.ErrNoRows || err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Check if user has access to this story
	if dbStory.CourseID.Valid {
		courseID := int32(dbStory.CourseID.Int32)
		if !CanUserAccessCourse(ctx, userID, courseID) {
			return nil, ErrNotFound
		}
	}

	// Convert DB story to model story
	story.Metadata.StoryID = int(dbStory.StoryID)
	story.Metadata.WeekNumber = int(dbStory.WeekNumber)
	story.Metadata.DayLetter = dbStory.DayLetter
	if dbStory.VideoUrl.Valid {
		story.Metadata.VideoURL = dbStory.VideoUrl.String
	}
	if dbStory.LastRevision.Valid {
		story.Metadata.LastRevision = &dbStory.LastRevision.Time
	}
	story.Metadata.Author.ID = dbStory.AuthorID
	story.Metadata.Author.Name = dbStory.AuthorName
	if dbStory.CourseID.Valid {
		courseID := int(dbStory.CourseID.Int32)
		story.Metadata.CourseID = &courseID
	}

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
			story.Metadata.Language = storyWithDesc.LanguageCode.String
			story.Metadata.Description.Text = storyWithDesc.DescriptionText.String
		}
	}

	// Get grammar points
	grammarPoints, err := GetStoryGrammarPoints(ctx, id)
	if err != nil {
		return nil, err
	}
	story.Metadata.GrammarPoints = grammarPoints

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
			AudioFiles: []AudioFile{},
			Footnotes:  []Footnote{},
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
				grammarItem := GrammarItem{
					Text:     grammar.Text,
					Position: [2]int{int(grammar.PositionStart), int(grammar.PositionEnd)},
				}
				if grammar.GrammarPointID.Valid {
					gpID := int(grammar.GrammarPointID.Int32)
					grammarItem.GrammarPointID = &gpID
				}
				line.Grammar = append(line.Grammar, grammarItem)
			}
		}

		// Get audio files for this line
		audioFiles, err := queries.GetLineAudioFiles(ctx, db.GetLineAudioFilesParams{
			StoryID:    pgtype.Int4{Int32: int32(storyID), Valid: true},
			LineNumber: pgtype.Int4{Int32: int32(dbLine.LineNumber), Valid: true},
		})
		if err != nil {
			return nil, err
		}
		for _, audio := range audioFiles {
			line.AudioFiles = append(line.AudioFiles, AudioFile{
				ID:         int(audio.AudioFileID),
				FilePath:   audio.FilePath,
				FileBucket: audio.FileBucket,
				Label:      audio.Label,
			})
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
	// Try cache first if available
	if cacheInstance != nil && keyBuilder != nil {
		cacheKey := keyBuilder.LineAnnotations(storyID, lineNumber)
		var line StoryLine
		err := cacheInstance.GetOrSetJSON(cacheKey, &line, func() (any, error) {
			return getLineAnnotationsFromDB(ctx, storyID, lineNumber)
		})
		if err != nil {
			return nil, err
		}
		return &line, nil
	}

	// Fallback to direct DB access
	return getLineAnnotationsFromDB(ctx, storyID, lineNumber)
}

// getLineAnnotationsFromDB performs the actual database operations for GetLineAnnotations
func getLineAnnotationsFromDB(ctx context.Context, storyID int, lineNumber int) (*StoryLine, error) {
	line := &StoryLine{
		LineNumber: lineNumber,
		Vocabulary: []VocabularyItem{}, // init as empty arrays
		Grammar:    []GrammarItem{},
		AudioFiles: []AudioFile{},
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
			grammarItem := GrammarItem{
				Text:     grammar.Text,
				Position: [2]int{int(grammar.PositionStart), int(grammar.PositionEnd)},
			}
			if grammar.GrammarPointID.Valid {
				gpID := int(grammar.GrammarPointID.Int32)
				grammarItem.GrammarPointID = &gpID
			}
			line.Grammar = append(line.Grammar, grammarItem)
		}
	}

	// Get audio files for this line
	audioFiles, err := queries.GetLineAudioFiles(ctx, db.GetLineAudioFilesParams{
		StoryID:    pgtype.Int4{Int32: int32(storyID), Valid: true},
		LineNumber: pgtype.Int4{Int32: int32(lineNumber), Valid: true},
	})
	if err != nil {
		return nil, err
	}
	for _, audio := range audioFiles {
		line.AudioFiles = append(line.AudioFiles, AudioFile{
			ID:         int(audio.AudioFileID),
			FilePath:   audio.FilePath,
			FileBucket: audio.FileBucket,
			Label:      audio.Label,
		})
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
	// Try cache first if available
	if cacheInstance != nil && keyBuilder != nil {
		cacheKey := keyBuilder.StoryAnnotations(storyID)
		var annotations map[int]*StoryLine
		err := cacheInstance.GetOrSetJSON(cacheKey, &annotations, func() (any, error) {
			return getStoryAnnotationsFromDB(ctx, storyID)
		})
		if err != nil {
			return nil, err
		}
		return annotations, nil
	}

	// Fallback to direct DB access
	return getStoryAnnotationsFromDB(ctx, storyID)
}

// getStoryAnnotationsFromDB performs the actual database operations for GetStoryAnnotations
func getStoryAnnotationsFromDB(ctx context.Context, storyID int) (map[int]*StoryLine, error) {
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
				AudioFiles: []AudioFile{},
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
				AudioFiles: []AudioFile{},
				Footnotes:  []Footnote{},
			}
		}
		grammarItem := GrammarItem{
			Text:     grammar.Text,
			Position: [2]int{int(grammar.PositionStart), int(grammar.PositionEnd)},
		}
		if grammar.GrammarPointID.Valid {
			gpID := int(grammar.GrammarPointID.Int32)
			grammarItem.GrammarPointID = &gpID
		}
		lines[lineNumber].Grammar = append(lines[lineNumber].Grammar, grammarItem)
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
				AudioFiles: []AudioFile{},
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

	// Get all audio files for the story and organize by line
	allAudioFiles, err := queries.GetAllStoryAudioFiles(ctx, pgtype.Int4{Int32: int32(storyID), Valid: true})
	if err != nil {
		return nil, err
	}
	for _, audio := range allAudioFiles {
		lineNumber := int(audio.LineNumber.Int32)
		if lines[lineNumber] == nil {
			lines[lineNumber] = &StoryLine{
				LineNumber: lineNumber,
				Vocabulary: []VocabularyItem{},
				Grammar:    []GrammarItem{},
				AudioFiles: []AudioFile{},
				Footnotes:  []Footnote{},
			}
		}
		lines[lineNumber].AudioFiles = append(lines[lineNumber].AudioFiles, AudioFile{
			ID:         int(audio.AudioFileID),
			FilePath:   audio.FilePath,
			FileBucket: audio.FileBucket,
			Label:      audio.Label,
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
		if err == sql.ErrNoRows || err == pgx.ErrNoRows {
			return "", ErrInvalidLineNumber
		}
		return "", err
	}
	return text, nil
}

// withTransaction executes a function within a database transaction
func withTransaction(fn func() error) error {
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
		err = fn()

		// Restore original queries
		queries = oldQueries

		if err != nil {
			return err
		}

		return tx.Commit(ctx)
	}

	// Fallback for other connection types
	fmt.Println("# Connection type not recognized. Transactions disabled.")
	return fn()
}

func GetAllStories(ctx context.Context, language string, userID string) ([]Story, error) {
	// Don't cache "all stories" - this is user-specific due to access controls
	// Individual stories are cached separately in GetStoryData
	return getAllStoriesFromDB(ctx, language, userID)
}

// getAllStoriesFromDB performs the actual database operations for GetAllStories
func getAllStoriesFromDB(ctx context.Context, language string, userID string) ([]Story, error) {
	basicStories, err := queries.GetAllStoriesForUser(ctx, db.GetAllStoriesForUserParams{
		LanguageCode: language,
		UserID:       userID,
	})
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

// Get stories for course doesn't use cache, but returns all available stories for a course
// It returns just basic information
func GetStoriesForCourse(ctx context.Context, courseID int) ([]Story, error) {
	stories, err := queries.GetCourseStoriesWithTitles(ctx, db.GetCourseStoriesWithTitlesParams{
		CourseID:     pgtype.Int4{Int32: int32(courseID), Valid: true},
		LanguageCode: "en",
	})
	if err != nil {
		return nil, err
	}

	result := make([]Story, len(stories))
	for i, story := range stories {
		result[i] = Story{
			Metadata: StoryMetadata{
				StoryID:    int(story.StoryID),
				WeekNumber: int(story.WeekNumber),
				DayLetter:  story.DayLetter,
				Title:      map[string]string{"en": story.Title},
			},
		}
	}

	return result, nil
}
