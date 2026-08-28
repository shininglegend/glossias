package models

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"glossias/src/pkg/generated/db"

	"github.com/jackc/pgx/v5"
)

// ProduceGradingPrompt is one version of the grader's system prompt. Versions
// are append-only; the newest is active. ID 0 with IsDefault set is the
// built-in prompt, used only when the table is empty or unreadable.
type ProduceGradingPrompt struct {
	ID        int       `json:"id"`
	Text      string    `json:"text"`
	Note      string    `json:"note,omitempty"`
	CreatedBy string    `json:"createdBy,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	IsDefault bool      `json:"isDefault,omitempty"`
}

// MaxGradingPromptLen bounds an edited prompt; the default is ~2 KB.
const MaxGradingPromptLen = 20000

func toGradingPrompt(row db.ProduceGradingPrompt) ProduceGradingPrompt {
	return ProduceGradingPrompt{
		ID:        int(row.ID),
		Text:      row.PromptText,
		Note:      row.Note.String,
		CreatedBy: row.CreatedBy.String,
		CreatedAt: row.CreatedAt.Time,
	}
}

// GetActiveProduceGradingPrompt returns the newest stored prompt, or
// ErrNotFound when none has been stored.
func GetActiveProduceGradingPrompt(ctx context.Context) (ProduceGradingPrompt, error) {
	if queries == nil {
		return ProduceGradingPrompt{}, errors.New("database not initialized")
	}
	row, err := queries.GetActiveProduceGradingPrompt(ctx)
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, pgx.ErrNoRows) {
		return ProduceGradingPrompt{}, ErrNotFound
	}
	if err != nil {
		return ProduceGradingPrompt{}, err
	}
	return toGradingPrompt(row), nil
}

// ListProduceGradingPrompts returns every version, newest first.
func ListProduceGradingPrompts(ctx context.Context) ([]ProduceGradingPrompt, error) {
	if queries == nil {
		return nil, errors.New("database not initialized")
	}
	rows, err := queries.ListProduceGradingPrompts(ctx)
	if err != nil {
		return nil, err
	}
	prompts := make([]ProduceGradingPrompt, 0, len(rows))
	for _, row := range rows {
		prompts = append(prompts, toGradingPrompt(row))
	}
	return prompts, nil
}

// CreateProduceGradingPrompt appends a new version, which becomes active
// for every grading run from now on.
func CreateProduceGradingPrompt(ctx context.Context, text, note, createdBy string) (ProduceGradingPrompt, error) {
	if queries == nil {
		return ProduceGradingPrompt{}, errors.New("database not initialized")
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return ProduceGradingPrompt{}, errors.New("prompt text is empty")
	}
	row, err := queries.InsertProduceGradingPrompt(ctx, db.InsertProduceGradingPromptParams{
		PromptText: text,
		Note:       optionalText(strings.TrimSpace(note)),
		CreatedBy:  optionalText(createdBy),
	})
	if err != nil {
		return ProduceGradingPrompt{}, err
	}
	return toGradingPrompt(row), nil
}

// EnsureProduceGradingPrompt seeds the table with the built-in default when
// it is empty, so the first version is on record and every log row can point
// at a real version. Called once at startup.
func EnsureProduceGradingPrompt(ctx context.Context) error {
	if queries == nil {
		return errors.New("database not initialized")
	}
	n, err := queries.CountProduceGradingPrompts(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	_, err = CreateProduceGradingPrompt(ctx, DefaultGradingSystemPrompt, "Built-in default", "")
	return err
}

// defaultGradingPrompt is what the grader falls back to when no version can
// be read from the database.
func defaultGradingPrompt() ProduceGradingPrompt {
	return ProduceGradingPrompt{Text: DefaultGradingSystemPrompt, IsDefault: true}
}
