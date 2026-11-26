package reaction

import (
	"context"

	"github.com/francesandprincesss/new-telegram/internal/apperror"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GORMRepository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *GORMRepository { return &GORMRepository{db: db} }

func (r *GORMRepository) Upsert(ctx context.Context, input Reaction) (Reaction, error) {
	var result Reaction
	err := r.db.WithContext(ctx).Raw(`
		INSERT INTO reactions (message_id, user_id, reaction, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (message_id, user_id) DO UPDATE SET
			reaction = EXCLUDED.reaction,
			updated_at = CASE WHEN reactions.reaction = EXCLUDED.reaction THEN reactions.updated_at
				ELSE GREATEST(reactions.updated_at + INTERVAL '1 microsecond', EXCLUDED.updated_at) END
		RETURNING message_id, user_id, reaction, created_at, updated_at`,
		input.MessageID, input.UserID, input.Type, input.CreatedAt, input.UpdatedAt).Scan(&result).Error
	if err != nil {
		return Reaction{}, apperror.Wrap(apperror.CodeInternal, "could not set reaction", err)
	}
	return result, nil
}

func (r *GORMRepository) Delete(ctx context.Context, messageID, userID uuid.UUID) (bool, error) {
	result := r.db.WithContext(ctx).Exec("DELETE FROM reactions WHERE message_id = ? AND user_id = ?", messageID, userID)
	if result.Error != nil {
		return false, apperror.Wrap(apperror.CodeInternal, "could not remove reaction", result.Error)
	}
	return result.RowsAffected > 0, nil
}
