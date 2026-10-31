package user

import (
	"context"

	"github.com/google/uuid"
)

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service { return &Service{repository: repository} }

// UserExists allows the chat module to validate dialog participants without
// depending on this package's repository implementation.
func (s *Service) UserExists(ctx context.Context, userID uuid.UUID) (bool, error) {
	return s.repository.Exists(ctx, userID)
}

// TODO: Implement GetMe and exact normalized-username lookup.
