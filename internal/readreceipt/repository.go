package readreceipt

import (
	"context"
	"time"

	"github.com/francesandprincesss/new-telegram/internal/apperror"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GORMRepository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *GORMRepository { return &GORMRepository{db: db} }

func (r *GORMRepository) MarkThrough(
	ctx context.Context,
	dialogID, readerID, upToMessageID uuid.UUID,
	readAt time.Time,
) (MarkResult, error) {
	var result MarkResult
	err := r.db.WithContext(ctx).Raw(`
		WITH inserted AS (
			INSERT INTO read_receipts (message_id, user_id, read_at)
			SELECT message.id, ?, ?
			FROM messages AS message
			JOIN messages AS cutoff ON cutoff.id = ? AND cutoff.dialog_id = message.dialog_id
			WHERE message.dialog_id = ? AND message.sender_id <> ?
			AND (message.created_at, message.id) <= (cutoff.created_at, cutoff.id)
			ORDER BY message.created_at, message.id
			ON CONFLICT (message_id, user_id) DO NOTHING
			RETURNING read_at
		)
		SELECT COUNT(*) AS marked_count, COALESCE(MAX(read_at),
			(SELECT read_at FROM read_receipts WHERE message_id = ? AND user_id = ?),
			CAST(? AS timestamptz)) AS read_at
		FROM inserted`,
		readerID, readAt, upToMessageID, dialogID, readerID, upToMessageID, readerID, readAt).Scan(&result).Error
	if err != nil {
		return MarkResult{}, apperror.Wrap(apperror.CodeInternal, "could not mark messages as read", err)
	}
	if result.MarkedCount == 0 {
		// A conflicting transaction can commit after the insert's snapshot.
		// Read the persisted cutoff in a fresh statement for idempotent replies.
		err := r.db.WithContext(ctx).Table("read_receipts").Select("read_at").
			Where("message_id = ? AND user_id = ?", upToMessageID, readerID).Take(&result).Error
		if err != nil {
			return MarkResult{}, apperror.Wrap(apperror.CodeInternal, "could not load read receipt", err)
		}
	}
	return result, nil
}
