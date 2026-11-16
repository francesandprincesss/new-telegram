package chat

import (
	"context"

	"github.com/francesandprincesss/new-telegram/internal/apperror"
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

func (s *Service) CreateDirectDialog(ctx context.Context, currentUserID uuid.UUID, otherUserID uuid.UUID) (Dialog, error) {
	if currentUserID == uuid.Nil || otherUserID == uuid.Nil {
		return Dialog{}, apperror.New(apperror.CodeInvalidArgument, "cannot create dialog with nothing")
	}

	if currentUserID == otherUserID {
		return Dialog{}, apperror.New(apperror.CodeInvalidArgument, "cannot create dialog with yourself")
	}

	ok, err := s.users.UserExists(ctx, otherUserID); if err != nil {
		return Dialog{}, apperror.Wrap(apperror.CodeInternal, "failed to check user in storage", err)
	}
	if !ok {
		return Dialog{}, apperror.New(apperror.CodeNotFound, "cannot create dialog with not real user")
	}

	dialog, err := s.repository.CreateOrGetDirect(ctx, currentUserID, otherUserID); if err != nil {
		return Dialog{}, apperror.Wrap(apperror.CodeInternal, "failed create or get dialog", err)
	}

	return dialog, nil
}

func (s *Service) ListDialogs(ctx context.Context, userID uuid.UUID, cursor *Cursor, limit int) ([]Dialog, error) {
	if userID == uuid.Nil {
		return nil, apperror.New(apperror.CodeInvalidArgument, "invalid user id")
	}

	if limit == 0 {
		limit = 15
	}

	if limit < 1 || limit > 15 {
		return nil, apperror.New(apperror.CodeInvalidArgument, "incorrect limit")
	}

	dialogs, err := s.repository.ListForUser(ctx, userID, cursor, limit+1); if err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, "failed get dialog", err)
	}

	return dialogs, nil
}