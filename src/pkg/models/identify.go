package models

import (
	"context"
	"errors"
	"time"

	"glossias/src/pkg/generated/db"
)

// IdentifyAnswer is one correct pick recorded during the Identify phase.
type IdentifyAnswer struct {
	LineNumber    int       `json:"lineNumber"`
	TargetVocabID int       `json:"targetVocabId"`
	LexicalForm   string    `json:"lexicalForm"`
	AttemptedAt   time.Time `json:"attemptedAt"`
}

// SaveIdentifyAnswer records a picture pick. selectedTargetVocabID is the target
// word whose image the student clicked; the pick is correct when it matches
// targetVocabID. Returns whether the pick was correct.
func SaveIdentifyAnswer(ctx context.Context, userID string, storyID, lineNumber, targetVocabID, selectedTargetVocabID int) (bool, error) {
	if queries == nil {
		return false, errors.New("database not initialized")
	}

	correct := targetVocabID == selectedTargetVocabID
	if correct {
		err := queries.SaveIdentifyCorrectAnswer(ctx, db.SaveIdentifyCorrectAnswerParams{
			UserID:        userID,
			StoryID:       int32(storyID),
			LineNumber:    int32(lineNumber),
			TargetVocabID: int32(targetVocabID),
		})
		return true, err
	}

	err := queries.SaveIdentifyIncorrectAnswer(ctx, db.SaveIdentifyIncorrectAnswerParams{
		UserID:                userID,
		StoryID:               int32(storyID),
		LineNumber:            int32(lineNumber),
		TargetVocabID:         int32(targetVocabID),
		SelectedTargetVocabID: int32(selectedTargetVocabID),
	})
	return false, err
}

// GetUserStoryIdentifySummary returns the user's correct/incorrect counts, in
// the shape CalculateScoreWithRetriesAllowed expects.
func GetUserStoryIdentifySummary(ctx context.Context, userID string, storyID int) (AnswerSummary, error) {
	if queries == nil {
		return AnswerSummary{}, errors.New("database not initialized")
	}

	row, err := queries.GetUserStoryIdentifySummary(ctx, db.GetUserStoryIdentifySummaryParams{
		UserID:  userID,
		StoryID: int32(storyID),
	})
	if err != nil {
		return AnswerSummary{}, err
	}

	return AnswerSummary{
		CorrectCount:   row.CorrectCount,
		IncorrectCount: row.IncorrectCount,
	}, nil
}

// GetUserIdentifyCorrectAnswers returns the user's correct picks for a story.
func GetUserIdentifyCorrectAnswers(ctx context.Context, userID string, storyID int) ([]IdentifyAnswer, error) {
	if queries == nil {
		return nil, errors.New("database not initialized")
	}

	rows, err := queries.GetUserIdentifyCorrectAnswers(ctx, db.GetUserIdentifyCorrectAnswersParams{
		UserID:  userID,
		StoryID: int32(storyID),
	})
	if err != nil {
		return nil, err
	}

	answers := make([]IdentifyAnswer, 0, len(rows))
	for _, row := range rows {
		answers = append(answers, IdentifyAnswer{
			LineNumber:    int(row.LineNumber),
			TargetVocabID: int(row.TargetVocabID),
			LexicalForm:   row.LexicalForm,
			AttemptedAt:   row.AttemptedAt.Time,
		})
	}

	return answers, nil
}

// GetUserIncompleteIdentifyTargets returns the target words the user has not
// yet identified correctly, so a partially finished phase can resume.
func GetUserIncompleteIdentifyTargets(ctx context.Context, userID string, storyID int) ([]TargetVocabulary, error) {
	if queries == nil {
		return nil, errors.New("database not initialized")
	}

	rows, err := queries.GetUserIncompleteIdentifyTargets(ctx, db.GetUserIncompleteIdentifyTargetsParams{
		StoryID: int32(storyID),
		UserID:  userID,
	})
	if err != nil {
		return nil, err
	}

	targets := make([]TargetVocabulary, 0, len(rows))
	for _, row := range rows {
		targets = append(targets, TargetVocabulary{
			ID:          int(row.ID),
			StoryID:     storyID,
			LexicalForm: row.LexicalForm,
		})
	}

	return targets, nil
}
