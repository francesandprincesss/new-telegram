CREATE TABLE reactions (
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    reaction VARCHAR(16) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,

    PRIMARY KEY (message_id, user_id),
    CONSTRAINT reactions_type_check
        CHECK (reaction IN ('LIKE', 'LOVE', 'LAUGH', 'WOW'))
);
