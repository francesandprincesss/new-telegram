package chat

import (
	"context"

	"github.com/francesandprincesss/new-telegram/internal/apperror"
	"github.com/francesandprincesss/new-telegram/internal/pagination"
	"github.com/google/uuid"
)

type Service struct {
	repository Repository
	users      UserLookup
}

func NewService(repository Repository, users UserLookup) *Service {
	return &Service{repository: repository, users: users}
}

func (s *Service) CreateDirectDialog(ctx context.Context, actorID, otherID uuid.UUID) (Dialog, error) {
	if otherID == uuid.Nil || actorID == otherID {
		return Dialog{}, apperror.New(apperror.CodeInvalidArgument, "other_user_id must identify another user")
	}
	exists, err := s.users.UserExists(ctx, otherID)
	if err != nil {
		return Dialog{}, err
	}
	if !exists {
		return Dialog{}, apperror.New(apperror.CodeNotFound, "user not found")
	}
	return s.repository.CreateOrGetDirect(ctx, actorID, otherID)
}

func (s *Service) ListDialogs(ctx context.Context, actorID uuid.UUID, pageSize int, pageToken string) ([]Dialog, string, error) {
	size, err := pagination.Size(pageSize)
	if err != nil {
		return nil, "", err
	}
	var cursor *Cursor
	if pageToken != "" {
		timestamp, id, err := pagination.Decode(pageToken)
		if err != nil {
			return nil, "", err
		}
		cursor = &Cursor{UpdatedAt: timestamp, ID: id}
	}
	dialogs, err := s.repository.ListForUser(ctx, actorID, cursor, size+1)
	if err != nil {
		return nil, "", err
	}
	if len(dialogs) <= size {
		return dialogs, "", nil
	}
	last := dialogs[size-1]
	return dialogs[:size], pagination.Encode(last.UpdatedAt, last.ID), nil
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
