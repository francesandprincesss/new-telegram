package auth

import (
	"context"
	"errors"

	"github.com/francesandprincesss/new-telegram/internal/apperror"
	"gorm.io/gorm"
)

type GORMRepository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *GORMRepository { return &GORMRepository{db: db} }

func (r *GORMRepository) CreateAccount(ctx context.Context, account NewAccount) (Account, error) {
	row := Account{ID: account.ID, Username: account.Username, UsernameNormalized: account.UsernameNormalized, PasswordHash: account.PasswordHash, CreatedAt: account.CreatedAt, UpdatedAt: account.CreatedAt}
	if err := r.db.WithContext(ctx).Table("users").Create(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return Account{}, apperror.Wrap(apperror.CodeConflict, "username is already taken", err)
		}
		return Account{}, apperror.Wrap(apperror.CodeInternal, "could not create user", err)
	}
	return row, nil
}

func (r *GORMRepository) FindCredentialsByNormalizedUsername(ctx context.Context, username string) (Credentials, error) {
	var credentials Credentials
	err := r.db.WithContext(ctx).Table("users").Select("id AS user_id, username, password_hash, created_at").Where("username_normalized = ?", username).Take(&credentials).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Credentials{}, apperror.New(apperror.CodeNotFound, "user not found")
	}
	if err != nil {
		return Credentials{}, apperror.Wrap(apperror.CodeInternal, "could not load credentials", err)
	}
	return credentials, nil
}
