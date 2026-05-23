-- +goose Up
CREATE TABLE IF NOT EXISTS comment_edit_history (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    comment_id UUID NOT NULL REFERENCES study_comments(id) ON DELETE CASCADE,
    old_body   TEXT NOT NULL,
    new_body   TEXT NOT NULL,
    edited_by  TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_comment_edits_comment ON comment_edit_history(comment_id);

-- +goose Down
DROP TABLE IF EXISTS comment_edit_history;
