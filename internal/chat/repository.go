package chat

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type Repo struct {
	db	*gorm.DB
}

func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) EnsureParticipant(ctx context.Context, chatID, userID uuid.UUID) (error) {
	var newDialog Dialog

	if err := r.db.WithContext(ctx).Model(&Dialog{}).First(&newDialog, chatID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return status.Error(codes.NotFound, "record not found")
		}
		return status.Error(codes.Internal, "internal server error")
	}
	
	if (newDialog.UserHighID == userID) || (newDialog.UserLowID == userID) {
		return nil
	}

	return status.Error(codes.NotFound, "user not found") 
}

func (r *Repo) CreateOrGetDirect(ctx context.Context, firstUserID, secondUserID uuid.UUID) (Dialog, error) {
	var chat Dialog

	if err := r.db.WithContext(ctx).Model(&Dialog{}).Where("((user_high_id, user_low_id) = (?, ?) OR (user_high_id, user_low_id) = (?, ?))", firstUserID, secondUserID, secondUserID, firstUserID).First(&chat).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			newDialog := &Dialog{
				ID: uuid.New(),
				UserLowID: firstUserID,
				UserHighID: secondUserID,
				CreatedAt: time.Now(),
			}
		
			if err := r.db.WithContext(ctx).Create(&newDialog).Error; err != nil {
				return Dialog{}, status.Error(codes.Unimplemented, "invalid create database row")
			}
			return *newDialog, nil
		} else {
			return Dialog{}, status.Error(codes.Unimplemented, "ivalid find rows on database")
		}
	}
	return chat, nil
}

func (r *Repo) ListForUser(ctx context.Context, userID uuid.UUID, cursor *Cursor, limit int) ([]Dialog, error) {
	var dialogs []Dialog

	query := r.db.WithContext(ctx).Model(&Dialog{}).Where("user_low_id = ? OR user_high_id = ?", userID, userID)

	if cursor != nil {
		query = query.Where("(apdate_at, id) < (?, ?)", cursor.UpdatedAt, cursor.ID)
	}

	if err := query.Order("updated_at DESC, id DESC").Limit(limit).Find(&dialogs).Error; err != nil {
		return nil, status.Error(codes.Unimplemented, "invalid find chats users")
	}

	return dialogs, nil
}

func (r *Repo) PeerIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	var dataID []uuid.UUID

	if err := r.db.WithContext(ctx).Model(&Dialog{}).Where("user_high_id = ?", userID).Pluck("user_low_id" ,&dataID).Error; err != nil {
		return nil, status.Error(codes.Internal, "internal server error")
	}

	if err := r.db.WithContext(ctx).Model(&Dialog{}).Where("user_low_id = ?", userID).Pluck("user_high_id", &dataID).Error; err != nil {
		return nil, status.Error(codes.Internal, "internal server error")
	}

	return dataID, nil
}

func (r *Repo) Participants(ctx context.Context, dialogID uuid.UUID) ([2]uuid.UUID, error) {
	var dialogRepo [2]uuid.UUID
	var chat *Dialog

	if err := r.db.WithContext(ctx).Model(&Dialog{}).Where("id = ?", dialogID).Select("user_high_id, user_low_id").First(&chat).Error; err != nil {
		return [2]uuid.UUID{}, status.Error(codes.Internal, "internal server error")
	}

	dialogRepo[0] = chat.UserHighID
	dialogRepo[1] = chat.UserLowID

	return dialogRepo, nil
}