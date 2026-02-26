-- +goose Up
CREATE TABLE IF NOT EXISTS comment_reactions (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    comment_id UUID NOT NULL REFERENCES study_comments(id) ON DELETE CASCADE,
    user_id    TEXT NOT NULL,
    emoji      TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(comment_id, user_id, emoji)
);
CREATE INDEX IF NOT EXISTS idx_comment_reactions_comment ON comment_reactions(comment_id);

-- +goose Down
DROP TABLE IF EXISTS comment_reactions;
