package chat

import (
	"context"

	"github.com/google/uuid"
)

type Service struct {
	repository Repository
	users      UserLookup
}

func NewService(repository Repository, users UserLookup) *Service {
	return &Service{repository: repository, users: users}
}

func (s *Service) EnsureParticipant(ctx context.Context, dialogID, userID uuid.UUID) error {
	return s.repository.EnsureParticipant(ctx, dialogID, userID)
}

func (s *Service) Participants(ctx context.Context, dialogID uuid.UUID) ([2]uuid.UUID, error) {
	return s.repository.Participants(ctx, dialogID)
}

func (s *Service) PeerIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	return s.repository.PeerIDs(ctx, userID)
}

// TODO: Implement idempotent direct-dialog creation and keyset-paginated listing.
