package user

import (
	"context"
	"errors"

	"github.com/francesandprincesss/new-telegram/internal/apperror"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GORMRepository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *GORMRepository { return &GORMRepository{db: db} }

const publicUserColumns = "id, username, manual_status, last_seen_at, created_at, updated_at"

func (r *GORMRepository) GetByID(ctx context.Context, userID uuid.UUID) (User, error) {
	var result User
	err := r.db.WithContext(ctx).Table("users").Select(publicUserColumns).Where("id = ?", userID).Take(&result).Error
	return mapResult(result, err)
}

func (r *GORMRepository) FindByNormalizedUsername(ctx context.Context, username string) (User, error) {
	var result User
	err := r.db.WithContext(ctx).Table("users").Select(publicUserColumns).Where("username_normalized = ?", username).Take(&result).Error
	return mapResult(result, err)
}

func (r *GORMRepository) Exists(ctx context.Context, userID uuid.UUID) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Table("users").Where("id = ?", userID).Count(&count).Error; err != nil {
		return false, apperror.Wrap(apperror.CodeInternal, "could not check user", err)
	}
	return count > 0, nil
}

func mapResult(result User, err error) (User, error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return User{}, apperror.New(apperror.CodeNotFound, "user not found")
	}
	if err != nil {
		return User{}, apperror.Wrap(apperror.CodeInternal, "could not load user", err)
	}
	return result, nil
}
