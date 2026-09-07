package models

import (
	"context"
	"testing"

	"glossias/src/pkg/database"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestGetStoriesContentReadiness_QueryBudget(t *testing.T) {
	mockDB := database.NewMockDBTX()
	mockDB.StubQuery("name: GetStoriesVideoURLs", [][]any{
		{int32(1), pgtype.Text{}},
		{int32(2), pgtype.Text{}},
		{int32(3), pgtype.Text{}},
	}, nil)
	mockDB.StubQuery("name: GetStoriesTargetVocabulary", nil, nil)
	mockDB.StubQuery("name: GetStoriesLexicalFormCounts", nil, nil)
	mockDB.StubQuery("name: GetStoriesProduceSegments", nil, nil)
	mockDB.StubQuery("name: GetStoriesProduceExplanations", nil, nil)
	mockDB.StubQuery("name: GetStoriesRecallSentences", nil, nil)
	mockDB.StubQuery("name: GetStoriesLines", nil, nil)
	mockDB.StubQuery("name: GetStoriesAudioFilesByLabel", nil, nil)
	SetDB(mockDB)
	t.Cleanup(func() { SetDB(struct{}{}) })

	ctx, _ := database.WithQueryCounter(context.Background())
	reports, err := GetStoriesContentReadiness(ctx, []int{1, 2, 3})
	if err != nil {
		t.Fatal(err)
	}
	if got := database.QueryCount(ctx); got > 8 {
		t.Errorf("readiness for 3 stories made %d queries, want <= 8 (batched ANY($1))", got)
	}
	if len(reports) != 3 {
		t.Fatalf("got %d reports, want 3", len(reports))
	}
}

func TestGetStoryContentReadiness_MissingStory(t *testing.T) {
	mockDB := database.NewMockDBTX()
	mockDB.StubQuery("name: GetStoriesVideoURLs", nil, nil)
	SetDB(mockDB)
	t.Cleanup(func() { SetDB(struct{}{}) })

	_, err := GetStoryContentReadiness(context.Background(), 99)
	if err != ErrNotFound {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}
