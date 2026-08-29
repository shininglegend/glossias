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
// are append-only; produce_grading_active_prompt marks the one in use, so an
// earlier version can be made active again. ID 0 with IsDefault set is the
// built-in prompt, used only when nothing is stored or the table is unreadable.
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

// GetActiveProduceGradingPrompt returns the prompt version currently marked
// active, or ErrNotFound when none has been stored.
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

// SaveProduceGradingPrompt makes text the active prompt. If an earlier
// version has exactly this text it is re-activated rather than duplicated;
// otherwise a new version is appended and activated. The returned prompt is
// the one now active.
func SaveProduceGradingPrompt(ctx context.Context, text, note, userID string) (ProduceGradingPrompt, error) {
	if queries == nil {
		return ProduceGradingPrompt{}, errors.New("database not initialized")
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return ProduceGradingPrompt{}, errors.New("prompt text is empty")
	}

	existing, err := queries.GetProduceGradingPromptByText(ctx, text)
	switch {
	case err == nil:
		return ActivateProduceGradingPrompt(ctx, int(existing.ID), userID)
	case errors.Is(err, sql.ErrNoRows), errors.Is(err, pgx.ErrNoRows):
		// New text: fall through and append it.
	default:
		return ProduceGradingPrompt{}, err
	}

	row, err := queries.InsertProduceGradingPrompt(ctx, db.InsertProduceGradingPromptParams{
		PromptText: text,
		Note:       optionalText(strings.TrimSpace(note)),
		CreatedBy:  optionalText(userID),
	})
	if err != nil {
		return ProduceGradingPrompt{}, err
	}
	return ActivateProduceGradingPrompt(ctx, int(row.ID), userID)
}

// ActivateProduceGradingPrompt points the active marker at an existing
// version. Returns ErrNotFound for an unknown ID.
func ActivateProduceGradingPrompt(ctx context.Context, id int, userID string) (ProduceGradingPrompt, error) {
	if queries == nil {
		return ProduceGradingPrompt{}, errors.New("database not initialized")
	}
	row, err := queries.GetProduceGradingPrompt(ctx, int32(id))
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, pgx.ErrNoRows) {
		return ProduceGradingPrompt{}, ErrNotFound
	}
	if err != nil {
		return ProduceGradingPrompt{}, err
	}
	if err := queries.SetActiveProduceGradingPrompt(ctx, db.SetActiveProduceGradingPromptParams{
		PromptID:    row.ID,
		ActivatedBy: optionalText(userID),
	}); err != nil {
		return ProduceGradingPrompt{}, err
	}
	return toGradingPrompt(row), nil
}

// EnsureProduceGradingPrompt seeds the built-in default as the first, active
// version when no prompt has been stored yet, so every log row can point at a
// real version. Called once at startup.
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
	_, err = SaveProduceGradingPrompt(ctx, DefaultGradingSystemPrompt, "Built-in default", "")
	return err
}

// defaultGradingPrompt is what the grader falls back to when no version can
// be read from the database.
func defaultGradingPrompt() ProduceGradingPrompt {
	return ProduceGradingPrompt{Text: DefaultGradingSystemPrompt, IsDefault: true}
}
