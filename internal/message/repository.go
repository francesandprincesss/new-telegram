package message

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repo struct {
	db *gorm.DB
}

func NewRepo(db *gorm.DB) *Repo {
	return &Repo{
		db: db,
	}
}

func (r *Repo) GetByID(ctx context.Context, messageID uuid.UUID) (Message, error) {
	var message Message

	if err := r.db.WithContext(ctx).Model(&Message{}).Where("ID = ?", messageID).Take(&message).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Message{}, fmt.Errorf("record not found: %w", err)
		}
		return Message{}, fmt.Errorf("error with find record: %w", err)
	}

	return message, nil
}

func (r *Repo) ListByDialog(ctx context.Context, dialogID uuid.UUID, cursor *Cursor, limit int) ([]Message, error) {
	var messages []Message

	query := r.db.WithContext(ctx).Model(&Message{}).Where("dialog_id = ?", dialogID)

	if cursor != nil {
		query = query.Where("(created_at, id) < (?, ?)", cursor.CreatedAt, cursor.ID)
	}

	if err := query.Find(&messages).Limit(limit).Order("created_at DESC, id DESC").Error; err != nil {
		return nil, fmt.Errorf("incorrect find messages: %w", err)
	}

	return messages, nil
}

func (r *Repo) CreateAndTouchDialog(ctx context.Context, message NewMessage) (Message, error) {
	msg := Message(message)

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error{
		if err := tx.Model(&Message{}).Create(&msg).Error; err != nil {
			return err
		}

		query := tx.Table("dialogs").Where("dialog_id = ?", message.DialogID).Update("updated_at = ?", message.CreatedAt)

		if query.RowsAffected == 1 {
			if err := query.Error; err != nil {
				return err
			}
		} else {
			return fmt.Errorf("rowsAffected > 1")
		}

		return nil
	}); if err != nil {
		return Message{}, fmt.Errorf("invalid transaction: %w", err)
	}

	return 
}