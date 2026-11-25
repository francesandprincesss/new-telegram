DROP INDEX messages_reply_target_idx;
ALTER TABLE messages
    DROP CONSTRAINT messages_reply_same_dialog_fk,
    DROP CONSTRAINT messages_dialog_id_id_key,
    DROP CONSTRAINT messages_reply_not_self_check,
    DROP COLUMN reply_to_id;
