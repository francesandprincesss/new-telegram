ALTER TABLE messages
    ADD COLUMN reply_to_id UUID,
    ADD CONSTRAINT messages_dialog_id_id_key UNIQUE (dialog_id, id),
    ADD CONSTRAINT messages_reply_not_self_check CHECK (reply_to_id <> id),
    ADD CONSTRAINT messages_reply_same_dialog_fk
        FOREIGN KEY (dialog_id, reply_to_id) REFERENCES messages (dialog_id, id);

CREATE INDEX messages_reply_target_idx
    ON messages (dialog_id, reply_to_id) WHERE reply_to_id IS NOT NULL;
