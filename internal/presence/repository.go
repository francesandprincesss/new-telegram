package presence

import (
	"context"
	"errors"
	"time"

	"github.com/francesandprincesss/new-telegram/internal/apperror"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GORMRepository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *GORMRepository { return &GORMRepository{db: db} }

func (r *GORMRepository) Get(ctx context.Context, userID uuid.UUID) (State, error) {
	var row struct {
		ID           uuid.UUID  `gorm:"column:id"`
		ManualStatus Status     `gorm:"column:manual_status"`
		LastSeenAt   *time.Time `gorm:"column:last_seen_at"`
	}
	err := r.db.WithContext(ctx).Table("users").Select("id, manual_status, last_seen_at").Where("id = ?", userID).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return State{}, apperror.New(apperror.CodeNotFound, "user not found")
	}
	if err != nil {
		return State{}, apperror.Wrap(apperror.CodeInternal, "could not load status", err)
	}
	return State{UserID: row.ID, Status: row.ManualStatus, LastSeenAt: row.LastSeenAt}, nil
}

func (r *GORMRepository) SetManualStatus(ctx context.Context, userID uuid.UUID, status Status) error {
	result := r.db.WithContext(ctx).Table("users").Where("id = ?", userID).Update("manual_status", status)
	if result.Error != nil {
		return apperror.Wrap(apperror.CodeInternal, "could not update status", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperror.New(apperror.CodeNotFound, "user not found")
	}
	return nil
}

func (r *GORMRepository) SetLastSeen(ctx context.Context, userID uuid.UUID, seenAt time.Time) error {
	result := r.db.WithContext(ctx).Table("users").Where("id = ?", userID).Update("last_seen_at", seenAt)
	if result.Error != nil {
		return apperror.Wrap(apperror.CodeInternal, "could not update last seen", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperror.New(apperror.CodeNotFound, "user not found")
	}
	return nil
}

type PeerRepository struct{ db *gorm.DB }

func NewPeerRepository(db *gorm.DB) *PeerRepository { return &PeerRepository{db: db} }

func (r *PeerRepository) PeerIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := r.db.WithContext(ctx).Raw(`
		SELECT CASE WHEN user_low_id = ? THEN user_high_id ELSE user_low_id END
		FROM dialogs WHERE user_low_id = ? OR user_high_id = ?`, userID, userID, userID).Scan(&ids).Error
	if err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, "could not load presence recipients", err)
	}
	return ids, nil
}
