package chat

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

// TODO: Заполнять peer, last_message и unread_count.
// Пока эти поля возвращаются с нулевыми значениями.
// CreateDirectDialog может вернуть уже существующий диалог.

type Handler struct {
	chatv1.UnimplementedChatServiceServer
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) Register(server *grpc.Server) {
	chatv1.RegisterChatServiceServer(server, h)
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

func (h *Handler) CreateDirectDialog(ctx context.Context, req *chatv1.CreateDirectDialogRequest) (*chatv1.Dialog, error) {
	userID, err := session.UserID(ctx); if err != nil {
		return nil, status.Error(codes.Unauthenticated, "authentication requaired")
	}

	otherUserID, err := uuid.Parse(req.GetOtherUserId()); if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid other_user_id")
	}

	dialog, err := h.service.CreateDirectDialog(ctx, userID, otherUserID); if err != nil {
		return nil, toGRPCError(err)
	}

	response := &chatv1.Dialog{
		Id:						dialog.ID.String(),
		LastMessage: 	nil,
		UnreadCount: 	0,
		CreatedAt: 		timestamppb.New(dialog.CreatedAt),
		UpdatedAt: 		timestamppb.New(dialog.UpdatedAt),
	}

	return response, nil

}

func (h *Handler) ListDialogs(
	ctx context.Context, req *chatv1.ListDialogsRequest,
) (*chatv1.ListDialogsResponse, error) {

	userID, err := session.UserID(ctx); if err != nil {
		return nil, status.Error(codes.Unauthenticated, "authentification requaired")
	}

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is requaired")
	}

	limit := int(req.GetPageSize())

	if limit == 0 {
		limit = 15
	}

	if limit < 0 || limit > 15 {
		return nil, status.Error(codes.InvalidArgument, "invalid page size")
	}

	token := req.GetPageToken()

	cursor, err := decodePageToken(token); if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid page token")
	}

	dialogs, err := h.service.ListDialogs(ctx, userID, cursor, limit); if err != nil {
		return nil, toGRPCError(err)
	}

	hasNext := len(dialogs) > limit

	if hasNext {
		dialogs = dialogs[:limit]
	}

	pbDialogs := make([]*chatv1.Dialog, 0, len(dialogs))

	for _, dialog := range dialogs {
		pbDialogs = append(pbDialogs, &chatv1.Dialog{
			Id: dialog.ID.String(),
			CreatedAt: timestamppb.New(dialog.CreatedAt),
			UpdatedAt: timestamppb.New(dialog.UpdatedAt),
		})
	}

	if hasNext {
		lastDialog := dialogs[len(dialogs)-1]

		nextToken, err := encodePageToken(&Cursor{
			UpdatedAt:  lastDialog.UpdatedAt,
			ID: 				lastDialog.ID,
		}); if err != nil {
			return nil, status.Error(codes.Internal, "failed to encode page token")
		}

		response := &chatv1.ListDialogsResponse{
		Dialogs: pbDialogs,
		PageInfo: &chatv1.PageInfo{
			NextPageToken: nextToken,
		},
	}

	return response, nil

	}

	response := &chatv1.ListDialogsResponse{
		Dialogs: pbDialogs,
		PageInfo: &chatv1.PageInfo{
			NextPageToken: "",
		},
	}

	return response, nil
}