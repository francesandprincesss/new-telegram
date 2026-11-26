package message

import (
	"context"
	"errors"
	"time"

	"github.com/francesandprincesss/new-telegram/internal/apperror"
	"github.com/francesandprincesss/new-telegram/internal/reaction"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GORMRepository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *GORMRepository { return &GORMRepository{db: db} }

func (r *GORMRepository) CreateAndTouchDialog(ctx context.Context, input NewMessage) (Message, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Lock the dialog before inserting so concurrent sends share a stable
		// chronology for pagination and read cutoffs, even within one microsecond.
		stamp := struct{ UpdatedAt time.Time }{}
		result := tx.Raw(`
			UPDATE dialogs
			SET updated_at = GREATEST(?, updated_at + INTERVAL '1 microsecond')
			WHERE id = ?
			RETURNING updated_at`, input.CreatedAt, input.DialogID).Scan(&stamp)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		input.CreatedAt = stamp.UpdatedAt
		return tx.Table("messages").Create(&input).Error
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Message{}, apperror.New(apperror.CodeNotFound, "dialog not found")
	}
	if err != nil {
		return Message{}, apperror.Wrap(apperror.CodeInternal, "could not send message", err)
	}
	return Message{
		ID: input.ID, DialogID: input.DialogID, SenderID: input.SenderID,
		Body: input.Body, CreatedAt: input.CreatedAt,
	}, nil
}

func (r *GORMRepository) ListByDialog(ctx context.Context, dialogID uuid.UUID, cursor *Cursor, limit int) ([]Message, error) {
	var messages []Message
	query := r.db.WithContext(ctx).Table("messages").Select(`messages.*,
		(SELECT read_at FROM read_receipts
		 WHERE message_id = messages.id AND user_id <> messages.sender_id
		 ORDER BY read_at LIMIT 1) AS read_at`).Where("dialog_id = ?", dialogID)
	if cursor != nil {
		query = query.Where("(created_at, id) < (?, ?)", cursor.CreatedAt, cursor.ID)
	}
	if err := query.Order("created_at DESC, id DESC").Limit(limit).Find(&messages).Error; err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, "could not load messages", err)
	}
	if err := r.loadReactions(ctx, messages); err != nil {
		return nil, err
	}
	return messages, nil
}

func (r *GORMRepository) loadReactions(ctx context.Context, messages []Message) error {
	if len(messages) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, 0, len(messages))
	positions := make(map[uuid.UUID]int, len(messages))
	for i, message := range messages {
		ids = append(ids, message.ID)
		positions[message.ID] = i
	}
	var reactions []reaction.Reaction
	err := r.db.WithContext(ctx).Table("reactions").Where("message_id IN ?", ids).Order("user_id").Find(&reactions).Error
	if err != nil {
		return apperror.Wrap(apperror.CodeInternal, "could not load reactions", err)
	}
	for _, item := range reactions {
		index := positions[item.MessageID]
		messages[index].Reactions = append(messages[index].Reactions, item)
	}
	return nil
}

func (r *GORMRepository) GetByID(ctx context.Context, messageID uuid.UUID) (Message, error) {
	var message Message
	err := r.db.WithContext(ctx).Table("messages").Where("id = ?", messageID).Take(&message).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Message{}, apperror.New(apperror.CodeNotFound, "message not found")
	}
	if err != nil {
		return Message{}, apperror.Wrap(apperror.CodeInternal, "could not load message", err)
	}
	return message, nil
}

func (r *GORMRepository) CountUnread(ctx context.Context, dialogID, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("messages").
		Where(`dialog_id = ? AND sender_id <> ? AND NOT EXISTS (
			SELECT 1 FROM read_receipts WHERE message_id = messages.id AND user_id = ?
		)`, dialogID, userID, userID).Count(&count).Error
	if err != nil {
		return 0, apperror.Wrap(apperror.CodeInternal, "could not count unread messages", err)
	}
	return count, nil
}
