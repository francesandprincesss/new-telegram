package user

import (
	"context"
	"time"

	chatv1 "github.com/francesandprincesss/new-telegram/gen/chat/v1"
	"github.com/francesandprincesss/new-telegram/internal/apperror"
	"github.com/francesandprincesss/new-telegram/internal/grpcutil"
	"github.com/francesandprincesss/new-telegram/internal/session"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Handler struct {
	chatv1.UnimplementedUserServiceServer
	service *Service
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) GetMe(ctx context.Context, _ *chatv1.GetMeRequest) (*chatv1.User, error) {
	userID, err := session.UserID(ctx)
	if err != nil {
		return nil, grpcutil.Error(apperror.New(apperror.CodeUnauthenticated, "authentication is required"))
	}
	result, err := h.service.Get(ctx, userID)
	if err != nil {
		return nil, grpcutil.Error(err)
	}
	return toUser(result), nil
}

func (h *Handler) GetUser(ctx context.Context, request *chatv1.GetUserRequest) (*chatv1.UserSummary, error) {
	if request == nil {
		request = &chatv1.GetUserRequest{}
	}
	userID, err := uuid.Parse(request.GetUserId())
	if err != nil || userID == uuid.Nil {
		return nil, grpcutil.Error(apperror.New(apperror.CodeInvalidArgument, "user_id must be a UUID"))
	}
	result, err := h.service.Get(ctx, userID)
	if err != nil {
		return nil, grpcutil.Error(err)
	}
	return toSummary(result), nil
}

func (h *Handler) FindUser(ctx context.Context, request *chatv1.FindUserRequest) (*chatv1.UserSummary, error) {
	if request == nil {
		request = &chatv1.FindUserRequest{}
	}
	result, err := h.service.Find(ctx, request.GetUsername())
	if err != nil {
		return nil, grpcutil.Error(err)
	}
	return toSummary(result), nil
}

func (h *Handler) FindByUsername(ctx context.Context, request *chatv1.FindByUsernameRequest) (*chatv1.UserSummary, error) {
	if request == nil {
		request = &chatv1.FindByUsernameRequest{}
	}
	return h.FindUser(ctx, &chatv1.FindUserRequest{Username: request.GetUsername()})
}

func toUser(result User) *chatv1.User {
	return &chatv1.User{Id: result.ID.String(), Username: result.Username, Status: protoStatus(result.ManualStatus), LastSeenAt: timestamp(result.LastSeenAt), CreatedAt: timestamppb.New(result.CreatedAt)}
}

func toSummary(result User) *chatv1.UserSummary {
	return &chatv1.UserSummary{Id: result.ID.String(), Username: result.Username, Status: protoStatus(result.ManualStatus), LastSeenAt: timestamp(result.LastSeenAt)}
}

func protoStatus(value string) chatv1.PresenceStatus {
	switch value {
	case "ONLINE":
		return chatv1.PresenceStatus_PRESENCE_STATUS_ONLINE
	case "IDLE":
		return chatv1.PresenceStatus_PRESENCE_STATUS_IDLE
	case "DND":
		return chatv1.PresenceStatus_PRESENCE_STATUS_DND
	default:
		return chatv1.PresenceStatus_PRESENCE_STATUS_OFFLINE
	}
}

func timestamp(value *time.Time) *timestamppb.Timestamp {
	if value == nil {
		return nil
	}
	return timestamppb.New(*value)
}
