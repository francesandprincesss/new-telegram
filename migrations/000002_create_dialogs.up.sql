CREATE TABLE dialogs (
    id UUID PRIMARY KEY,
    user_low_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    user_high_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,

    CONSTRAINT dialogs_user_order_check CHECK (user_low_id < user_high_id),
    CONSTRAINT dialogs_user_pair_unique UNIQUE (user_low_id, user_high_id)
);

CREATE INDEX dialogs_user_low_updated_idx
    ON dialogs (user_low_id, updated_at DESC, id DESC);

CREATE INDEX dialogs_user_high_updated_idx
    ON dialogs (user_high_id, updated_at DESC, id DESC);
