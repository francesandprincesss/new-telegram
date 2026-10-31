CREATE TABLE users (
    id UUID PRIMARY KEY,
    username VARCHAR(32) NOT NULL,
    username_normalized VARCHAR(32) NOT NULL,
    password_hash VARCHAR(60) NOT NULL,
    manual_status VARCHAR(16) NOT NULL DEFAULT 'ONLINE',
    last_seen_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,

    CONSTRAINT users_username_format_check
        CHECK (username ~ '^[A-Za-z0-9_]{3,32}$'),
    CONSTRAINT users_username_normalized_check
        CHECK (username_normalized = lower(username)),
    CONSTRAINT users_username_normalized_unique
        UNIQUE (username_normalized),
    CONSTRAINT users_manual_status_check
        CHECK (manual_status IN ('ONLINE', 'IDLE', 'DND'))
);
