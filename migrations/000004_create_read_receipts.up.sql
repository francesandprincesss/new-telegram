CREATE TABLE read_receipts (
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    read_at TIMESTAMPTZ NOT NULL,

    PRIMARY KEY (message_id, user_id)
);

CREATE INDEX read_receipts_user_read_idx
    ON read_receipts (user_id, read_at DESC);
