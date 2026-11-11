package user

import (
	"context"
	"regexp"
	"strings"

	"github.com/francesandprincesss/new-telegram/internal/apperror"
	"github.com/google/uuid"
)

var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9_]{3,32}$`)

type Service struct {
	repository Repository
	presence   PresenceResolver
}

func NewService(repository Repository, presence PresenceResolver) *Service {
	return &Service{repository: repository, presence: presence}
}

// UserExists allows the chat module to validate dialog participants without
// depending on this package's repository implementation.
func (s *Service) UserExists(ctx context.Context, userID uuid.UUID) (bool, error) {
	return s.repository.Exists(ctx, userID)
}

func (s *Service) Get(ctx context.Context, userID uuid.UUID) (User, error) {
	result, err := s.repository.GetByID(ctx, userID)
	if err != nil {
		return User{}, err
	}
	return s.withPresence(ctx, result)
}

func (s *Service) Find(ctx context.Context, username string) (User, error) {
	if !usernamePattern.MatchString(username) {
		return User{}, apperror.New(apperror.CodeInvalidArgument, "username must be 3-32 ASCII letters, digits, or underscores")
	}
	result, err := s.repository.FindByNormalizedUsername(ctx, strings.ToLower(username))
	if err != nil {
		return User{}, err
	}
	return s.withPresence(ctx, result)
}

func (s *Service) withPresence(ctx context.Context, result User) (User, error) {
	status, lastSeenAt, err := s.presence.Resolve(ctx, result.ID)
	if err != nil {
		return User{}, err
	}
	result.ManualStatus = status
	result.LastSeenAt = lastSeenAt
	return result, nil
}
