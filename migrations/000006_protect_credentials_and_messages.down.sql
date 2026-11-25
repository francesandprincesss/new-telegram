BEGIN;

LOCK TABLE users, messages, message_encryption_key IN ACCESS EXCLUSIVE MODE;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM messages WHERE body_ciphertext IS NOT NULL)
       OR EXISTS (SELECT 1 FROM users WHERE char_length(password_hash) > 60) THEN
        RAISE EXCEPTION 'cannot downgrade while encrypted messages or new password hashes exist';
    END IF;
END
$$;

DROP TABLE message_encryption_key;
DROP INDEX messages_legacy_body_idx;
ALTER TABLE messages
    DROP CONSTRAINT messages_encrypted_body_check,
    DROP COLUMN body_ciphertext,
    ALTER COLUMN body SET NOT NULL;
ALTER TABLE users ALTER COLUMN password_hash TYPE VARCHAR(60);

COMMIT;
