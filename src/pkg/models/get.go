// story_data.go
package models

import (
	"context"
	"database/sql"
	"fmt"
	"glossias/src/pkg/database"
	"strconv"
	"time"

	"glossias/src/pkg/generated/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/lib/pq"
)

// accessCacheTTL is how long a cached "allowed" stands. BigCache keeps the
// entry for its whole life window (hours), so the age is checked here; the
// access epoch (BumpAccessEpoch) cuts it short on known revocations.
const accessCacheTTL = 5 * time.Minute

// checkStoryContentAccessCached is CheckStoryContentAccess with positive
// results cached per user and story. Denials (ErrNotFound, and
// *PaymentRequiredError) and other errors are never cached, so a new
// enrollment, link or payment opens the story at once.
func checkStoryContentAccessCached(ctx context.Context, userID string, storyID int) error {
	if cacheInstance == nil || keyBuilder == nil {
		return CheckStoryContentAccess(ctx, userID, int32(storyID))
	}
	key := keyBuilder.StoryAccess(userID, storyID)
	if data, err := cacheInstance.Get(key); err == nil {
		if at, perr := strconv.ParseInt(string(data), 10, 64); perr == nil &&
			time.Since(time.Unix(at, 0)) < accessCacheTTL {
			return nil
		}
	}
	if err := CheckStoryContentAccess(ctx, userID, int32(storyID)); err != nil {
		return err
	}
	_ = cacheInstance.Set(key, []byte(strconv.FormatInt(time.Now().Unix(), 10)))
	return nil
}

// CheckStoryContentAccessCached is the cached access check for handlers that
// write on a story without loading it (ErrNotFound → 404,
// *PaymentRequiredError → 402).
func CheckStoryContentAccessCached(ctx context.Context, userID string, storyID int) error {
	return checkStoryContentAccessCached(ctx, userID, storyID)
}

func GetStoryData(ctx context.Context, id int, userID string) (*Story, error) {
	if err := checkStoryContentAccessCached(ctx, userID, id); err != nil {
		return nil, err
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

	// Main story data and its description in one query
	dbStory, err := queries.GetStoryWithDescription(ctx, int32(id))
	if err != nil {
		if err == sql.ErrNoRows || err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
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
	if dbStory.LanguageCode.Valid && dbStory.DescriptionText.Valid {
		story.Metadata.Language = dbStory.LanguageCode.String
		story.Metadata.Description.Text = dbStory.DescriptionText.String
	}

	// Get titles
	titles, err := queries.GetStoryTitles(ctx, int32(id))
	if err != nil {
		return nil, err
	}
	for _, title := range titles {
		story.Metadata.Title[title.LanguageCode] = title.Title
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

// footnoteRefs is never nil, so a footnote without references reads as [].
func footnoteRefs(refs []string) []string {
	if refs == nil {
		return []string{}
	}
	return refs
}

func getStoryLines(ctx context.Context, storyID int) ([]StoryLine, error) {
	dbLines, err := queries.GetStoryLines(ctx, int32(storyID))
	if err != nil {
		return nil, err
	}

	// Map for Vocabulary items grouped by line number
	vocabMap := make(map[int][]VocabularyItem)
	vocabRows, err := queries.GetAllVocabularyForStory(ctx, pgtype.Int4{Int32: int32(storyID), Valid: true})
	if err != nil {
		return nil, err
	}
	for _, v := range vocabRows {
		lineNum := int(v.LineNumber.Int32)
		vocabMap[lineNum] = append(vocabMap[lineNum], VocabularyItem{
			Word:        v.Word,
			LexicalForm: v.LexicalForm,
			Position:    [2]int{int(v.PositionStart), int(v.PositionEnd)},
		})
	}

	// Map for Grammar items grouped by line number
	grammarMap := make(map[int][]GrammarItem)
	grammarRows, err := queries.GetAllGrammarForStory(ctx, pgtype.Int4{Int32: int32(storyID), Valid: true})
	if err != nil {
		return nil, err
	}
	for _, g := range grammarRows {
		lineNum := int(g.LineNumber.Int32)
		grammarItem := GrammarItem{
			Text:     g.Text,
			Position: [2]int{int(g.PositionStart), int(g.PositionEnd)},
		}
		if g.GrammarPointID.Valid {
			gpID := int(g.GrammarPointID.Int32)
			grammarItem.GrammarPointID = &gpID
		}
		grammarMap[lineNum] = append(grammarMap[lineNum], grammarItem)
	}

	// Map for Audio files grouped by line number
	audioMap := make(map[int][]AudioFile)
	audioRows, err := queries.GetAllStoryAudioFiles(ctx, pgtype.Int4{Int32: int32(storyID), Valid: true})
	if err != nil {
		return nil, err
	}
	for _, a := range audioRows {
		lineNum := int(a.LineNumber.Int32)
		audioMap[lineNum] = append(audioMap[lineNum], AudioFile{
			ID:         int(a.AudioFileID),
			FilePath:   a.FilePath,
			FileBucket: a.FileBucket,
			Label:      a.Label,
		})
	}

	// Map for Footnotes grouped by line number
	footnoteMap := make(map[int][]Footnote)
	footnoteRows, err := queries.GetStoryFootnotesWithRefs(ctx, int32(storyID))
	if err != nil {
		return nil, err
	}
	for _, f := range footnoteRows {
		lineNum := int(f.LineNumber.Int32)
		footnoteMap[lineNum] = append(footnoteMap[lineNum], Footnote{
			ID:         int(f.ID),
			Text:       f.FootnoteText,
			References: footnoteRefs(f.Refs),
		})
	}

	var lines []StoryLine
	for _, dbLine := range dbLines {
		lineNum := int(dbLine.LineNumber)

		vocabItems := vocabMap[lineNum]
		if vocabItems == nil {
			vocabItems = []VocabularyItem{}
		}

		grammarItems := grammarMap[lineNum]
		if grammarItems == nil {
			grammarItems = []GrammarItem{}
		}

		audioFiles := audioMap[lineNum]
		if audioFiles == nil {
			audioFiles = []AudioFile{}
		}

		footnotes := footnoteMap[lineNum]
		if footnotes == nil {
			footnotes = []Footnote{}
		}

		line := StoryLine{
			LineNumber: lineNum,
			Text:       dbLine.Text,
			Vocabulary: vocabItems,
			Grammar:    grammarItems,
			AudioFiles: audioFiles,
			Footnotes:  footnotes,
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
	vocabItems, err := queries.GetVocabularyItems(ctx, db.GetVocabularyItemsParams{
		StoryID:    pgtype.Int4{Int32: int32(storyID), Valid: true},
		LineNumber: pgtype.Int4{Int32: int32(lineNumber), Valid: true},
	})
	if err != nil {
		return nil, err
	}
	for _, vocab := range vocabItems {
		line.Vocabulary = append(line.Vocabulary, VocabularyItem{
			Word:        vocab.Word,
			LexicalForm: vocab.LexicalForm,
			Position:    [2]int{int(vocab.PositionStart), int(vocab.PositionEnd)},
		})
	}

	// Get grammar items for this line
	grammarItems, err := queries.GetGrammarItems(ctx, db.GetGrammarItemsParams{
		StoryID:    pgtype.Int4{Int32: int32(storyID), Valid: true},
		LineNumber: pgtype.Int4{Int32: int32(lineNumber), Valid: true},
	})
	if err != nil {
		return nil, err
	}
	for _, grammar := range grammarItems {
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
	footnotes, err := queries.GetLineFootnotesWithRefs(ctx, db.GetLineFootnotesWithRefsParams{
		StoryID:    int32(storyID),
		LineNumber: int32(lineNumber),
	})
	if err != nil {
		return nil, err
	}
	for _, fn := range footnotes {
		line.Footnotes = append(line.Footnotes, Footnote{
			ID:         int(fn.ID),
			Text:       fn.FootnoteText,
			References: footnoteRefs(fn.Refs),
		})
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
	footnotes, err := queries.GetStoryFootnotesWithRefs(ctx, int32(storyID))
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
		lines[lineNumber].Footnotes = append(lines[lineNumber].Footnotes, Footnote{
			ID:         int(fn.ID),
			Text:       fn.FootnoteText,
			References: footnoteRefs(fn.Refs),
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
func withTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	var pool *pgxpool.Pool
	if p, ok := rawConn.(*pgxpool.Pool); ok {
		pool = p
	} else if r, ok := rawConn.(*database.ReconnectableDBTX); ok {
		pool = r.Pool()
	}

	if pool != nil {
		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx)

		txCtx := context.WithValue(ctx, TxContextKey{}, tx)

		err = fn(txCtx)
		if err != nil {
			return err
		}

		return tx.Commit(ctx)
	}

	// Fallback for other connection types
	fmt.Println("# Connection type not recognized. Transactions disabled.")
	return fn(ctx)
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
		if basicStory.CourseID.Valid {
			courseID := int(basicStory.CourseID.Int32)
			story.Metadata.CourseID = &courseID
		}
		if len(basicStory.CourseIds) > 0 {
			story.Metadata.LinkedCourseIDs = int32sToInts(basicStory.CourseIds)
		}
		stories = append(stories, story)
	}
	if err := AnnotateStoryLocks(ctx, userID, stories); err != nil {
		return nil, err
	}
	return stories, nil
}

// Get stories for course doesn't use cache, but returns all available stories for a course
// It returns just basic information
func GetStoriesForCourse(ctx context.Context, courseID int) ([]Story, error) {
	stories, err := queries.GetCourseStoriesWithTitles(ctx, db.GetCourseStoriesWithTitlesParams{
		CourseID:     int32(courseID),
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
