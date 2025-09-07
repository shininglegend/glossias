package repository

import (
	"context"
	"database/sql"
	"glossias/src/pkg/generated/db"
	"glossias/src/pkg/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{
		pool:    pool,
		queries: db.New(pool),
	}
}

// GetStoryData retrieves a complete story with all its components
func (r *Repository) GetStoryData(ctx context.Context, storyID int) (*models.Story, error) {
	story, err := r.queries.GetStory(ctx, int32(storyID))
	if err != nil {
		if err == sql.ErrNoRows || err == pgx.ErrNoRows {
			return nil, models.ErrNotFound
		}
		return nil, err
	}

	// Get titles
	titles, err := r.queries.GetStoryTitles(ctx, int32(storyID))
	if err != nil {
		return nil, err
	}

	titleMap := make(map[string]string)
	for _, title := range titles {
		titleMap[title.LanguageCode] = title.Title
	}

	// Get lines
	lines, err := r.queries.GetStoryLines(ctx, int32(storyID))
	if err != nil {
		return nil, err
	}

	storyLines := make([]models.StoryLine, len(lines))
	for i, line := range lines {
		storyLine := models.StoryLine{
			LineNumber: int(line.LineNumber),
			Text:       line.Text,
		}

		if line.AudioFile.Valid {
			storyLine.AudioFile = &line.AudioFile.String
		}

		// Get vocabulary for this line
		vocab, err := r.queries.GetVocabularyItems(ctx, db.GetVocabularyItemsParams{
			StoryID:    pgtype.Int4{Int32: int32(storyID), Valid: true},
			LineNumber: pgtype.Int4{Int32: line.LineNumber, Valid: true},
		})
		if err != nil {
			return nil, err
		}

		vocabItems := make([]models.VocabularyItem, len(vocab))
		for j, v := range vocab {
			vocabItems[j] = models.VocabularyItem{
				Word:        v.Word,
				LexicalForm: v.LexicalForm,
				Position:    [2]int{int(v.PositionStart), int(v.PositionEnd)},
			}
		}
		storyLine.Vocabulary = vocabItems

		// Get grammar for this line
		grammar, err := r.queries.GetGrammarItems(ctx, db.GetGrammarItemsParams{
			StoryID:    pgtype.Int4{Int32: int32(storyID), Valid: true},
			LineNumber: pgtype.Int4{Int32: line.LineNumber, Valid: true},
		})
		if err != nil {
			return nil, err
		}

		grammarItems := make([]models.GrammarItem, len(grammar))
		for j, g := range grammar {
			grammarItems[j] = models.GrammarItem{
				Text:     g.Text,
				Position: [2]int{int(g.PositionStart), int(g.PositionEnd)},
			}
		}
		storyLine.Grammar = grammarItems

		// Get footnotes for this line
		footnotes, err := r.queries.GetFootnotes(ctx, db.GetFootnotesParams{
			StoryID:    pgtype.Int4{Int32: int32(storyID), Valid: true},
			LineNumber: pgtype.Int4{Int32: line.LineNumber, Valid: true},
		})
		if err != nil {
			return nil, err
		}

		footnoteItems := make([]models.Footnote, len(footnotes))
		for j, f := range footnotes {
			refs, err := r.queries.GetFootnoteReferences(ctx, f.ID)
			if err != nil {
				return nil, err
			}

			footnoteItems[j] = models.Footnote{
				ID:         int(f.ID),
				Text:       f.FootnoteText,
				References: refs,
			}
		}
		storyLine.Footnotes = footnoteItems

		storyLines[i] = storyLine
	}

	result := &models.Story{
		Metadata: models.StoryMetadata{
			StoryID:    int(story.StoryID),
			WeekNumber: int(story.WeekNumber),
			DayLetter:  story.DayLetter,
			Title:      titleMap,
			Author: models.Author{
				ID:   story.AuthorID,
				Name: story.AuthorName,
			},
			LastRevision: &story.LastRevision.Time,
		},
		Content: models.StoryContent{
			Lines: storyLines,
		},
	}

	if story.GrammarPoint.Valid {
		result.Metadata.GrammarPoint = story.GrammarPoint.String
	}

	return result, nil
}

// GetAllStories retrieves basic story list
func (r *Repository) GetAllStories(ctx context.Context) ([]models.Story, error) {
	stories, err := r.queries.GetAllStories(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]models.Story, len(stories))
	for i, story := range stories {
		// Get titles for each story
		titles, err := r.queries.GetStoryTitles(ctx, story.StoryID)
		if err != nil {
			return nil, err
		}

		titleMap := make(map[string]string)
		for _, title := range titles {
			titleMap[title.LanguageCode] = title.Title
		}

		result[i] = models.Story{
			Metadata: models.StoryMetadata{
				StoryID:    int(story.StoryID),
				WeekNumber: int(story.WeekNumber),
				DayLetter:  story.DayLetter,
				Title:      titleMap,
				Author: models.Author{
					ID:   story.AuthorID,
					Name: story.AuthorName,
				},
				LastRevision: &story.LastRevision.Time,
			},
		}

		if story.GrammarPoint.Valid {
			result[i].Metadata.GrammarPoint = story.GrammarPoint.String
		}
	}

	return result, nil
}

// SaveNewStory creates a new story
func (r *Repository) SaveNewStory(ctx context.Context, story *models.Story) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	// Create story
	grammarPoint := pgtype.Text{}
	if story.Metadata.GrammarPoint != "" {
		grammarPoint = pgtype.Text{String: story.Metadata.GrammarPoint, Valid: true}
	}

	result, err := qtx.CreateStory(ctx, db.CreateStoryParams{
		WeekNumber:   int32(story.Metadata.WeekNumber),
		DayLetter:    story.Metadata.DayLetter,
		GrammarPoint: grammarPoint,
		AuthorID:     story.Metadata.Author.ID,
		AuthorName:   story.Metadata.Author.Name,
	})
	if err != nil {
		return err
	}

	story.Metadata.StoryID = int(result.StoryID)
	story.Metadata.LastRevision = &result.LastRevision.Time

	// Save titles
	for lang, title := range story.Metadata.Title {
		err = qtx.UpsertStoryTitle(ctx, db.UpsertStoryTitleParams{
			StoryID:      result.StoryID,
			LanguageCode: lang,
			Title:        title,
		})
		if err != nil {
			return err
		}
	}

	// Save description if present
	if story.Metadata.Description.Text != "" {
		err = qtx.UpsertStoryDescription(ctx, db.UpsertStoryDescriptionParams{
			StoryID:         result.StoryID,
			LanguageCode:    story.Metadata.Description.Language,
			DescriptionText: story.Metadata.Description.Text,
		})
		if err != nil {
			return err
		}
	}

	// Save lines and annotations
	for _, line := range story.Content.Lines {
		audioFile := pgtype.Text{}
		if line.AudioFile != nil {
			audioFile = pgtype.Text{String: *line.AudioFile, Valid: true}
		}

		err = qtx.UpsertStoryLine(ctx, db.UpsertStoryLineParams{
			StoryID:    result.StoryID,
			LineNumber: int32(line.LineNumber),
			Text:       line.Text,
			AudioFile:  audioFile,
		})
		if err != nil {
			return err
		}

		// Save vocabulary items
		for _, vocab := range line.Vocabulary {
			_, err = qtx.CreateVocabularyItem(ctx, db.CreateVocabularyItemParams{
				StoryID:       pgtype.Int4{Int32: result.StoryID, Valid: true},
				LineNumber:    pgtype.Int4{Int32: int32(line.LineNumber), Valid: true},
				Word:          vocab.Word,
				LexicalForm:   vocab.LexicalForm,
				PositionStart: int32(vocab.Position[0]),
				PositionEnd:   int32(vocab.Position[1]),
			})
			if err != nil {
				return err
			}
		}

		// Save grammar items
		for _, grammar := range line.Grammar {
			_, err = qtx.CreateGrammarItem(ctx, db.CreateGrammarItemParams{
				StoryID:       pgtype.Int4{Int32: result.StoryID, Valid: true},
				LineNumber:    pgtype.Int4{Int32: int32(line.LineNumber), Valid: true},
				Text:          grammar.Text,
				PositionStart: int32(grammar.Position[0]),
				PositionEnd:   int32(grammar.Position[1]),
			})
			if err != nil {
				return err
			}
		}

		// Save footnotes
		for _, footnote := range line.Footnotes {
			footnoteID, err := qtx.CreateFootnote(ctx, db.CreateFootnoteParams{
				StoryID:      pgtype.Int4{Int32: result.StoryID, Valid: true},
				LineNumber:   pgtype.Int4{Int32: int32(line.LineNumber), Valid: true},
				FootnoteText: footnote.Text,
			})
			if err != nil {
				return err
			}

			// Save footnote references
			for _, ref := range footnote.References {
				err = qtx.CreateFootnoteReference(ctx, db.CreateFootnoteReferenceParams{
					FootnoteID: footnoteID,
					Reference:  ref,
				})
				if err != nil {
					return err
				}
			}
		}
	}

	return tx.Commit(ctx)
}

// EditStoryMetadata updates story metadata
func (r *Repository) EditStoryMetadata(ctx context.Context, storyID int, metadata models.StoryMetadata) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)
	grammarPoint := pgtype.Text{}
	if metadata.GrammarPoint != "" {
		grammarPoint = pgtype.Text{String: metadata.GrammarPoint, Valid: true}
	}

	err = qtx.UpdateStory(ctx, db.UpdateStoryParams{
		StoryID:      int32(storyID),
		WeekNumber:   int32(metadata.WeekNumber),
		DayLetter:    metadata.DayLetter,
		GrammarPoint: grammarPoint,
	})
	if err != nil {
		return err
	}

	// Update titles
	for lang, title := range metadata.Title {
		err = qtx.UpsertStoryTitle(ctx, db.UpsertStoryTitleParams{
			StoryID:      int32(storyID),
			LanguageCode: lang,
			Title:        title,
		})
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// Delete removes a story and all associated data
func (r *Repository) Delete(ctx context.Context, storyID int) error {
	return r.queries.DeleteStory(ctx, int32(storyID))
}

// GetLineText retrieves just the text for a specific line
func (r *Repository) GetLineText(ctx context.Context, storyID, lineNumber int) (string, error) {
	line, err := r.queries.GetStoryLine(ctx, db.GetStoryLineParams{
		StoryID:    int32(storyID),
		LineNumber: int32(lineNumber),
	})
	if err != nil {
		if err == sql.ErrNoRows || err == pgx.ErrNoRows {
			return "", models.ErrNotFound
		}
		return "", err
	}
	return line.Text, nil
}
