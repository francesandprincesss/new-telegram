package event

import (
	"context"
	"log"
	"time"

	chatv1 "github.com/francesandprincesss/new-telegram/gen/chat/v1"
	"github.com/francesandprincesss/new-telegram/internal/session"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Handler struct {
	chatv1.UnimplementedEventServiceServer
	hub      *Hub
	observer ConnectionObserver
}

func NewHandler(hub *Hub, observer ConnectionObserver) *Handler {
	return &Handler{hub: hub, observer: observer}
}

func (h *Handler) SubscribeEvents(_ *chatv1.SubscribeEventsRequest, stream grpc.ServerStreamingServer[chatv1.EventEnvelope]) error {
	userID, err := session.UserID(stream.Context())
	if err != nil {
		return status.Error(codes.Unauthenticated, "authentication is required")
	}
	subscription := h.hub.Subscribe(userID)
	if err := h.observer.Connected(stream.Context(), userID); err != nil {
		subscription.Cancel()
		return status.Error(codes.Internal, "could not establish presence")
	}
	defer func() {
		subscription.Cancel()
		disconnectCtx, cancel := context.WithTimeout(context.WithoutCancel(stream.Context()), 5*time.Second)
		defer cancel()
		if err := h.observer.Disconnected(disconnectCtx, userID); err != nil {
			log.Printf("disconnect presence for %s: %v", userID, err)
		}
	}()
	for {
		select {
		case event := <-subscription.Events:
			if err := stream.Send(toEnvelope(event)); err != nil {
				return err
			}
		case <-subscription.Done:
			return status.Error(codes.ResourceExhausted, "event stream was closed")
		case <-stream.Context().Done():
			return stream.Context().Err()
		}
	}
}

func toEnvelope(value Event) *chatv1.EventEnvelope {
	envelope := &chatv1.EventEnvelope{EventId: value.ID.String(), OccurredAt: timestamppb.New(value.OccurredAt)}
	switch payload := value.Payload.(type) {
	case PresenceChanged:
		envelope.Payload = &chatv1.EventEnvelope_PresenceChanged{PresenceChanged: &chatv1.PresenceChanged{User: &chatv1.UserSummary{Id: payload.UserID.String(), Status: eventPresenceStatus(payload.Status), LastSeenAt: eventTimestamp(payload.LastSeenAt)}}}
	case MessageCreated:
		envelope.Payload = &chatv1.EventEnvelope_MessageCreated{MessageCreated: &chatv1.MessageCreated{Message: &chatv1.Message{Id: payload.MessageID.String(), DialogId: payload.DialogID.String(), SenderId: payload.SenderID.String(), Body: payload.Body, CreatedAt: timestamppb.New(payload.CreatedAt)}}}
	case MessagesRead:
		envelope.Payload = &chatv1.EventEnvelope_MessagesRead{MessagesRead: &chatv1.MessagesRead{DialogId: payload.DialogID.String(), ReaderId: payload.ReaderID.String(), UpToMessageId: payload.UpToMessageID.String(), ReadAt: timestamppb.New(payload.ReadAt)}}
	case ReactionChanged:
		envelope.Payload = &chatv1.EventEnvelope_ReactionChanged{ReactionChanged: &chatv1.ReactionChanged{MessageId: payload.MessageID.String(), UserId: payload.UserID.String(), Reaction: eventReaction(payload.Reaction), Removed: payload.Removed, UpdatedAt: timestamppb.New(payload.UpdatedAt)}}
	}
	return envelope
}

func eventTimestamp(value *time.Time) *timestamppb.Timestamp {
	if value == nil {
		return nil
	}
	return timestamppb.New(*value)
}

func eventPresenceStatus(value string) chatv1.PresenceStatus {
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

func eventReaction(value string) chatv1.ReactionType {
	switch value {
	case "LIKE":
		return chatv1.ReactionType_REACTION_TYPE_LIKE
	case "LOVE":
		return chatv1.ReactionType_REACTION_TYPE_LOVE
	case "LAUGH":
		return chatv1.ReactionType_REACTION_TYPE_LAUGH
	case "WOW":
		return chatv1.ReactionType_REACTION_TYPE_WOW
	default:
		return chatv1.ReactionType_REACTION_TYPE_UNSPECIFIED
	}
}
