package message

import (
	"context"

	chatv1 "github.com/francesandprincesss/new-telegram/gen/chat/v1"
	"github.com/francesandprincesss/new-telegram/internal/apperror"
	"github.com/francesandprincesss/new-telegram/internal/session"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Handler struct {
	chatv1.UnimplementedMessageServiceServer
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) Register(server *grpc.Server) {
	chatv1.RegisterMessageServiceServer(server, h)
}

func toGRPCError(err error) error {
	if err == nil {
		return nil
	}

	code := apperror.CodeOf(err)
	
	switch code {
    case apperror.CodeInvalidArgument:
        return status.Error(codes.InvalidArgument, err.Error())

    case apperror.CodeNotFound:
        return status.Error(codes.NotFound, err.Error())

    case apperror.CodeForbidden:
        return status.Error(codes.PermissionDenied, err.Error())

    case apperror.CodeUnauthenticated:
        return status.Error(codes.Unauthenticated, err.Error())

    case apperror.CodeConflict:
        return status.Error(codes.AlreadyExists, err.Error())

    default:
        return status.Error(codes.Internal, "internal server error")
    }
}

func (h *Handler) SendMessage(ctx context.Context, req *chatv1.SendMessageRequest) (*chatv1.Message, error) {
	userID, err := session.UserID(ctx); if err != nil {
		return nil, status.Error(codes.Unauthenticated, "unautenticated requaired")
	}

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "incorrect request")
	}

	dialogID, err := uuid.Parse(req.GetDialogId()); if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid have dialog id")
	}

	body := req.GetBody()

	message, err := h.service.SendMessage(ctx, dialogID, userID, body); if err != nil {
		return nil, toGRPCError(err)
	}

	response := &chatv1.Message{
		Id: message.ID.String(),
		DialogId: message.DialogID.String(),
		SenderId: message.SenderID.String(),
		Body: message.Body,
		CreatedAt: timestamppb.New(message.CreatedAt),
	}

	return response, nil
}

func (h *Handler) ListMessages(ctx context.Context, req *chatv1.ListMessagesRequest) (*chatv1.ListMessagesResponse, error) {
	userID, err := session.UserID(ctx); if err != nil {
		return nil, status.Error(codes.Unauthenticated, "unauthenticated requaired")
	}

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	limit := int(req.GetPageSize())

	if limit == 0 {
		limit = 15
	}

	if limit < 0 || limit > 15 {
		return nil, status.Error(codes.InvalidArgument, "invalid limit")
	}

	dialogID, err := uuid.Parse(req.GetDialogId()); if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid dialogID")
	}

	token := req.GetPageToken()

	cursor, err := decodePageToken(token); if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid token")
	}

	messages, err := h.service.ListMessages(ctx, userID, dialogID, cursor, limit); if err != nil {
		return nil, toGRPCError(err)
	}

	hasNext := len(messages) > limit
	if hasNext {
		messages = messages[:limit]
	}

	pbMessages := make([]*chatv1.Message, 0, len(messages))

	for _, message := range messages {
		pbMessages = append(pbMessages, &chatv1.Message{
			Id: message.ID.String(),
			DialogId: message.DialogID.String(),
			SenderId: message.SenderID.String(),
			Body: message.Body,
			CreatedAt: timestamppb.New(message.CreatedAt),
		})
	}

	if hasNext {
		lastMessage := messages[len(messages)-1]

		nextToken, err := encodePageToken(&Cursor{
			CreatedAt: lastMessage.CreatedAt,
			ID: lastMessage.ID,
		}); if err != nil {
			return nil, status.Error(codes.Internal, "failed to encode page token")
		}

		response := &chatv1.ListMessagesResponse{
			Messages: pbMessages,
			PageInfo: &chatv1.PageInfo{
				NextPageToken: nextToken,
			},
		}

		return response, nil
	}

	response := &chatv1.ListMessagesResponse{
		Messages: pbMessages,
		PageInfo: &chatv1.PageInfo{
			NextPageToken: "",
		},
	}

	return response, nil
}