CREATE TABLE messages (
    id UUID PRIMARY KEY,
    dialog_id UUID NOT NULL REFERENCES dialogs(id) ON DELETE CASCADE,
    sender_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    body TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,

    CONSTRAINT messages_body_length_check
        CHECK (char_length(body) BETWEEN 1 AND 4000),
    CONSTRAINT messages_body_not_blank_check
        CHECK (btrim(body) <> '')
);

CREATE INDEX messages_dialog_created_idx
    ON messages (dialog_id, created_at DESC, id DESC);
