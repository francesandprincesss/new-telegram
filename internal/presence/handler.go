package presence

import (
	"context"

	chatv1 "github.com/francesandprincesss/new-telegram/gen/chat/v1"
	"github.com/francesandprincesss/new-telegram/internal/apperror"
	"github.com/francesandprincesss/new-telegram/internal/grpcutil"
	"github.com/francesandprincesss/new-telegram/internal/session"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Handler struct {
	chatv1.UnimplementedPresenceServiceServer
	service *Service
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) SetStatus(ctx context.Context, request *chatv1.SetStatusRequest) (*chatv1.PresenceState, error) {
	userID, err := session.UserID(ctx)
	if err != nil {
		return nil, grpcutil.Error(apperror.New(apperror.CodeUnauthenticated, "authentication is required"))
	}
	if request == nil {
		request = &chatv1.SetStatusRequest{}
	}
	state, err := h.service.SetStatus(ctx, userID, fromProto(request.GetStatus()))
	if err != nil {
		return nil, grpcutil.Error(err)
	}
	response := &chatv1.PresenceState{UserId: state.UserID.String(), Status: toProto(state.Status)}
	if state.LastSeenAt != nil {
		response.LastSeenAt = timestamppb.New(*state.LastSeenAt)
	}
	return response, nil
}

func fromProto(status chatv1.PresenceStatus) Status {
	switch status {
	case chatv1.PresenceStatus_PRESENCE_STATUS_ONLINE:
		return Online
	case chatv1.PresenceStatus_PRESENCE_STATUS_IDLE:
		return Idle
	case chatv1.PresenceStatus_PRESENCE_STATUS_DND:
		return DND
	default:
		return Offline
	}
}

func toProto(status Status) chatv1.PresenceStatus {
	switch status {
	case Online:
		return chatv1.PresenceStatus_PRESENCE_STATUS_ONLINE
	case Idle:
		return chatv1.PresenceStatus_PRESENCE_STATUS_IDLE
	case DND:
		return chatv1.PresenceStatus_PRESENCE_STATUS_DND
	default:
		return chatv1.PresenceStatus_PRESENCE_STATUS_OFFLINE
	}
}
