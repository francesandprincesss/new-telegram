package chat

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repo struct {
	db	*gorm.DB
}

func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

// наличие пользователя в диалоге
func (r *Repo) EnsureParticipant(ctx context.Context, chatID, userID uuid.UUID) (error) {
	var newDialog Dialog

	if err := r.db.WithContext(ctx).Model(&Dialog{}).Where("id = ?", chatID).Take(&newDialog).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("record not found")
		}
		return fmt.Errorf("find dialog: %w", err)
	}
	
	if (newDialog.UserHighID == userID) || (newDialog.UserLowID == userID) {
		return nil
	}

	return fmt.Errorf("user not in dialog")
}

func (r *Repo) CreateOrGetDirect(ctx context.Context, firstUserID, secondUserID uuid.UUID) (Dialog, error) {
	var chat Dialog
	var chat2 Dialog
	lowID, highID := firstUserID, secondUserID
	if bytes.Compare(lowID[:], highID[:]) > 0 {
		lowID, highID = highID, lowID
	}

	if err := r.db.WithContext(ctx).Model(&Dialog{}).Where("user_low_id = ? AND user_high_id = ?", lowID, highID).Take(&chat).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			newDialog := &Dialog{
			ID: uuid.New(),
			UserLowID: lowID,
			UserHighID: highID,
			CreatedAt: time.Now(),
			}

			if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
				Columns: []clause.Column{
					{Name: "user_low_id"},
					{Name: "user_high_id"},
				},
				DoNothing: true,
			}).Create(&newDialog).Error; err != nil {
				return Dialog{}, fmt.Errorf("invalid create database row: %w", err)
			}

			if err := r.db.WithContext(ctx).Where("user_low_id = ? AND user_high_id = ?", lowID, highID).Take(&chat2).Error; err != nil {
				return Dialog{}, fmt.Errorf("invalid db request, have: %w", err)
			}

			return chat2, nil
			
		} else {
			return Dialog{}, fmt.Errorf("ivalid find rows on database")
		}
	}
	return chat, nil
}

// диалоги пользователя с пагинацией
func (r *Repo) ListForUser(ctx context.Context, userID uuid.UUID, cursor *Cursor, limit int) ([]Dialog, error) {
	var dialogs []Dialog

	query := r.db.WithContext(ctx).Model(&Dialog{}).Where("user_low_id = ? OR user_high_id = ?", userID, userID)

	if cursor != nil {
		query = query.Where("(updated_at, id) < (?, ?)", cursor.UpdatedAt, cursor.ID)
	}

	if err := query.Order("updated_at DESC, id DESC").Limit(limit).Find(&dialogs).Error; err != nil {
		return nil, fmt.Errorf("invalid find chats users: %w", err)
	}

	return dialogs, nil
}

// возвращает всех пользователей с кем есть диалог
func (r *Repo) PeerIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	var dataID []uuid.UUID
	var newDataID []uuid.UUID

	if err := r.db.WithContext(ctx).Model(&Dialog{}).Where("user_high_id = ?", userID).Pluck("user_low_id" ,&dataID).Error; err != nil {
		return nil, fmt.Errorf("internal server error: %w", err)
	}

	if err := r.db.WithContext(ctx).Model(&Dialog{}).Where("user_low_id = ?", userID).Pluck("user_high_id", &newDataID).Error; err != nil {
		return nil, fmt.Errorf("internal server error: %w", err)
	}

	dataID = append(dataID, newDataID...)

	return dataID, nil
}

// возвращает участников диалога
func (r *Repo) Participants(ctx context.Context, dialogID uuid.UUID) ([2]uuid.UUID, error) {
	var dialogRepo [2]uuid.UUID
	var chat Dialog

	if err := r.db.WithContext(ctx).Model(&Dialog{}).Select("user_high_id, user_low_id").Where("id = ?", dialogID).Take(&chat).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return [2]uuid.UUID{}, fmt.Errorf("dialog is not found")
		}
		return [2]uuid.UUID{}, fmt.Errorf("found: %w", err)
	}

	dialogRepo[0] = chat.UserHighID
	dialogRepo[1] = chat.UserLowID

	return dialogRepo, nil
}