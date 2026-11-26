package chat

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/francesandprincesss/new-telegram/internal/apperror"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GORMRepository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *GORMRepository { return &GORMRepository{db: db} }

func (r *GORMRepository) CreateOrGetDirect(ctx context.Context, firstUserID, secondUserID uuid.UUID) (Dialog, error) {
	if bytes.Compare(firstUserID[:], secondUserID[:]) > 0 {
		firstUserID, secondUserID = secondUserID, firstUserID
	}
	now := time.Now().UTC()
	dialog := Dialog{ID: uuid.New(), UserLowID: firstUserID, UserHighID: secondUserID, CreatedAt: now, UpdatedAt: now}
	result := r.db.WithContext(ctx).Table("dialogs").Clauses(
		clause.OnConflict{Columns: []clause.Column{{Name: "user_low_id"}, {Name: "user_high_id"}}, DoNothing: true},
		clause.Returning{},
	).Create(&dialog)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrForeignKeyViolated) {
			return Dialog{}, apperror.New(apperror.CodeNotFound, "user not found")
		}
		return Dialog{}, apperror.Wrap(apperror.CodeInternal, "could not create dialog", result.Error)
	}
	if result.RowsAffected == 0 {
		// A concurrent insert won. Use a new statement so its committed row is visible.
		// A populated primary key on the destination would also become a filter.
		dialog = Dialog{}
		err := r.db.WithContext(ctx).Table("dialogs").
			Where("user_low_id = ? AND user_high_id = ?", firstUserID, secondUserID).Take(&dialog).Error
		if err != nil {
			return Dialog{}, apperror.Wrap(apperror.CodeInternal, "could not load dialog", err)
		}
	}
	return dialog, nil
}

func (r *GORMRepository) ListForUser(ctx context.Context, userID uuid.UUID, cursor *Cursor, limit int) ([]Dialog, error) {
	var dialogs []Dialog
	query := r.db.WithContext(ctx).Table("dialogs").Where("(user_low_id = ? OR user_high_id = ?)", userID, userID)
	if cursor != nil {
		query = query.Where("(updated_at, id) < (?, ?)", cursor.UpdatedAt, cursor.ID)
	}
	if err := query.Order("updated_at DESC, id DESC").Limit(limit).Find(&dialogs).Error; err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, "could not load dialogs", err)
	}
	return dialogs, nil
}

func (r *GORMRepository) EnsureParticipant(ctx context.Context, dialogID, userID uuid.UUID) error {
	participants, err := r.Participants(ctx, dialogID)
	if err != nil {
		return err
	}
	if userID != participants[0] && userID != participants[1] {
		return apperror.New(apperror.CodeForbidden, "dialog access denied")
	}
	return nil
}

func (r *GORMRepository) Participants(ctx context.Context, dialogID uuid.UUID) ([2]uuid.UUID, error) {
	var dialog Dialog
	err := r.db.WithContext(ctx).Table("dialogs").Select("user_low_id, user_high_id").Where("id = ?", dialogID).Take(&dialog).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return [2]uuid.UUID{}, apperror.New(apperror.CodeNotFound, "dialog not found")
	}
	if err != nil {
		return [2]uuid.UUID{}, apperror.Wrap(apperror.CodeInternal, "could not load dialog", err)
	}
	return [2]uuid.UUID{dialog.UserLowID, dialog.UserHighID}, nil
}

func (r *GORMRepository) PeerIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := r.db.WithContext(ctx).Raw(`
		SELECT CASE WHEN user_low_id = ? THEN user_high_id ELSE user_low_id END
		FROM dialogs WHERE user_low_id = ? OR user_high_id = ?`, userID, userID, userID).Scan(&ids).Error
	if err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, "could not load dialog participants", err)
	}
	return ids, nil
}
