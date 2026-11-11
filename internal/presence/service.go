package presence

import (
	"context"

	"github.com/francesandprincesss/new-telegram/internal/apperror"
	"github.com/google/uuid"
)

type Service struct {
	manager *Manager
}

func NewService(manager *Manager) *Service {
	return &Service{manager: manager}
}

func (s *Service) SetStatus(ctx context.Context, userID uuid.UUID, status Status) (State, error) {
	if status != Online && status != Idle && status != DND {
		return State{}, apperror.New(apperror.CodeInvalidArgument, "status must be ONLINE, IDLE, or DND")
	}
	return s.manager.SetStatus(ctx, userID, status)
}
