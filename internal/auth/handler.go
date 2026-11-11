package auth

import (
	"context"

	chatv1 "github.com/francesandprincesss/new-telegram/gen/chat/v1"
	"github.com/francesandprincesss/new-telegram/internal/grpcutil"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Handler struct {
	chatv1.UnimplementedAuthServiceServer
	service *Service
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Register(ctx context.Context, request *chatv1.RegisterRequest) (*chatv1.AuthResponse, error) {
	if request == nil {
		request = &chatv1.RegisterRequest{}
	}
	result, err := h.service.Register(ctx, request.GetUsername(), request.GetPassword())
	if err != nil {
		return nil, grpcutil.Error(err)
	}
	return authResponse(result), nil
}

func (h *Handler) Login(ctx context.Context, request *chatv1.LoginRequest) (*chatv1.AuthResponse, error) {
	if request == nil {
		request = &chatv1.LoginRequest{}
	}
	result, err := h.service.Login(ctx, request.GetUsername(), request.GetPassword())
	if err != nil {
		return nil, grpcutil.Error(err)
	}
	return authResponse(result), nil
}

func authResponse(result Result) *chatv1.AuthResponse {
	return &chatv1.AuthResponse{
		AccessToken: result.Token,
		ExpiresAt:   timestamppb.New(result.ExpiresAt),
		User: &chatv1.User{Id: result.UserID.String(), Username: result.Username,
			Status: chatv1.PresenceStatus_PRESENCE_STATUS_OFFLINE, CreatedAt: timestamppb.New(result.CreatedAt)},
	}
}
