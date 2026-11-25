package webapi

import (
	"github.com/francesandprincesss/new-telegram/internal/auth"
	"github.com/francesandprincesss/new-telegram/internal/event"
	"github.com/francesandprincesss/new-telegram/internal/presence"
	"github.com/francesandprincesss/new-telegram/internal/user"
)

func authResult(result auth.Result) map[string]any {
	return map[string]any{
		"access_token": result.Token,
		"token_type":   "Bearer",
		"expires_at":   result.ExpiresAt,
		"user": map[string]any{
			"id": result.UserID, "username": result.Username,
			"created_at": result.CreatedAt,
		},
	}
}

func userResult(result user.User) map[string]any {
	return map[string]any{
		"id": result.ID, "username": result.Username, "status": result.ManualStatus,
		"last_seen_at": result.LastSeenAt, "created_at": result.CreatedAt,
	}
}

func presenceResult(result presence.State) map[string]any {
	return map[string]any{"user_id": result.UserID, "status": result.Status, "last_seen_at": result.LastSeenAt}
}

func eventName(value event.Event) string {
	switch value.Payload.(type) {
	case event.PresenceChanged:
		return "presence.changed"
	case event.MessageCreated:
		return "message.created"
	case event.MessagesRead:
		return "messages.read"
	case event.ReactionChanged:
		return "reaction.changed"
	default:
		return "event"
	}
}

func eventResult(value event.Event) map[string]any {
	result := map[string]any{"event_id": value.ID, "occurred_at": value.OccurredAt, "type": eventName(value)}
	switch payload := value.Payload.(type) {
	case event.PresenceChanged:
		result["payload"] = map[string]any{"user_id": payload.UserID, "status": payload.Status, "last_seen_at": payload.LastSeenAt}
	case event.MessageCreated:
		messagePayload := map[string]any{"message_id": payload.MessageID, "dialog_id": payload.DialogID, "sender_id": payload.SenderID, "body": payload.Body, "created_at": payload.CreatedAt}
		if payload.ReplyToID != nil {
			messagePayload["reply_to_message_id"] = *payload.ReplyToID
		}
		result["payload"] = messagePayload
	case event.MessagesRead:
		result["payload"] = map[string]any{"dialog_id": payload.DialogID, "reader_id": payload.ReaderID, "up_to_message_id": payload.UpToMessageID, "read_at": payload.ReadAt}
	case event.ReactionChanged:
		result["payload"] = map[string]any{"message_id": payload.MessageID, "user_id": payload.UserID, "reaction": payload.Reaction, "removed": payload.Removed, "updated_at": payload.UpdatedAt}
	default:
		result["payload"] = nil
	}
	return result
}
