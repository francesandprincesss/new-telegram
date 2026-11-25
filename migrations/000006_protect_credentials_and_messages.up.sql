BEGIN;

ALTER TABLE users ALTER COLUMN password_hash TYPE TEXT;

ALTER TABLE messages
    ALTER COLUMN body DROP NOT NULL,
    ADD COLUMN body_ciphertext BYTEA,
    ADD CONSTRAINT messages_encrypted_body_check
        CHECK (body IS NULL AND body_ciphertext IS NOT NULL) NOT VALID;

CREATE INDEX messages_legacy_body_idx ON messages (id) WHERE body IS NOT NULL;

CREATE TABLE message_encryption_key (
    singleton BOOLEAN PRIMARY KEY CHECK (singleton),
    verifier BYTEA NOT NULL CHECK (octet_length(verifier) = 32)
);

COMMIT;
