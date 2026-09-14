-- +goose Up
ALTER TABLE courses
    ADD COLUMN is_trial BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE users
    ADD COLUMN stripe_customer_id TEXT UNIQUE;

CREATE TABLE access_entitlements (
    entitlement_id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users (user_id) ON DELETE CASCADE,
    course_id INTEGER NOT NULL REFERENCES courses (course_id) ON DELETE CASCADE,
    source TEXT NOT NULL CHECK (source IN ('purchase', 'comp', 'promo')),
    starts_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMPTZ NOT NULL,
    amount_cents INTEGER,
    stripe_checkout_session_id TEXT UNIQUE,
    stripe_payment_intent_id TEXT,
    granted_by TEXT REFERENCES users (user_id) ON DELETE SET NULL
);

CREATE INDEX access_entitlements_user_course_expires_idx
    ON access_entitlements (user_id, course_id, expires_at);

CREATE INDEX access_entitlements_payment_intent_idx
    ON access_entitlements (stripe_payment_intent_id)
    WHERE stripe_payment_intent_id IS NOT NULL;

-- +goose Down
DROP TABLE IF EXISTS access_entitlements;

ALTER TABLE users
    DROP COLUMN IF EXISTS stripe_customer_id;

ALTER TABLE courses
    DROP COLUMN IF EXISTS is_trial;
