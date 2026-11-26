package webapi

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/francesandprincesss/new-telegram/internal/apperror"
	"github.com/francesandprincesss/new-telegram/internal/chat"
	"github.com/francesandprincesss/new-telegram/internal/event"
	"github.com/francesandprincesss/new-telegram/internal/message"
	"github.com/francesandprincesss/new-telegram/internal/reaction"
	"github.com/francesandprincesss/new-telegram/internal/readreceipt"
	"github.com/francesandprincesss/new-telegram/internal/session"
	"github.com/google/uuid"
)

type dialogService interface {
	CreateDirectDialog(context.Context, uuid.UUID, uuid.UUID) (chat.Dialog, error)
	ListDialogs(context.Context, uuid.UUID, int, string) ([]chat.Dialog, string, error)
	PeerInDialog(context.Context, uuid.UUID, uuid.UUID) (uuid.UUID, error)
}

type messageService interface {
	SendMessage(context.Context, uuid.UUID, uuid.UUID, string, *uuid.UUID) (message.Message, error)
	ListMessages(context.Context, uuid.UUID, uuid.UUID, int, string) ([]message.Message, string, error)
	DialogSummary(context.Context, uuid.UUID, uuid.UUID) (*message.Message, int64, error)
}

type receiptService interface {
	MarkRead(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (readreceipt.MarkResult, error)
}

type reactionService interface {
	SetReaction(context.Context, uuid.UUID, uuid.UUID, reaction.Type) (reaction.Reaction, error)
	RemoveReaction(context.Context, uuid.UUID, uuid.UUID) (bool, error)
}

// Messaging supplies the same application services for browser requests.
type Messaging struct {
	Dialogs   dialogService
	Messages  messageService
	Receipts  receiptService
	Reactions reactionService
}

func (s *Server) registerMessagingRoutes(mux *http.ServeMux) {
	protected := func(pattern string, handler http.HandlerFunc) {
		mux.Handle(pattern, s.authenticate(requestTimeout(handler)))
	}
	protected("GET /api/v1/dialogs", s.listDialogs)
	protected("POST /api/v1/dialogs", s.createDialog)
	protected("GET /api/v1/dialogs/{dialog_id}/messages", s.listMessages)
	protected("POST /api/v1/dialogs/{dialog_id}/messages", s.sendMessage)
	protected("POST /api/v1/dialogs/{dialog_id}/read", s.markRead)
	protected("POST /api/v1/dialogs/{dialog_id}/typing", s.typing)
	protected("PUT /api/v1/messages/{message_id}/reaction", s.setReaction)
	protected("DELETE /api/v1/messages/{message_id}/reaction", s.removeReaction)
}

type pageInfo struct {
	NextPageToken string `json:"next_page_token"`
}

type dialogResponse struct {
	ID          uuid.UUID        `json:"id"`
	Peer        peerResponse     `json:"peer"`
	LastMessage *messageResponse `json:"last_message"`
	UnreadCount int64            `json:"unread_count"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

type peerResponse struct {
	ID         uuid.UUID  `json:"id"`
	Username   string     `json:"username"`
	Status     string     `json:"status"`
	LastSeenAt *time.Time `json:"last_seen_at"`
}

type messageResponse struct {
	ID        uuid.UUID          `json:"id"`
	DialogID  uuid.UUID          `json:"dialog_id"`
	SenderID  uuid.UUID          `json:"sender_id"`
	Body      string             `json:"body"`
	CreatedAt time.Time          `json:"created_at"`
	ReadAt    *time.Time         `json:"read_at"`
	Reactions []reactionResponse `json:"reactions"`
	ReplyTo   *replyResponse     `json:"reply_to,omitempty"`
}

type replyResponse struct {
	ID       uuid.UUID `json:"id"`
	SenderID uuid.UUID `json:"sender_id"`
	Excerpt  string    `json:"excerpt"`
}

type reactionResponse struct {
	MessageID uuid.UUID     `json:"message_id"`
	UserID    uuid.UUID     `json:"user_id"`
	Reaction  reaction.Type `json:"reaction"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

func (s *Server) dialogResult(ctx context.Context, actorID uuid.UUID, dialog chat.Dialog) (dialogResponse, error) {
	peerID := dialog.UserHighID
	if peerID == actorID {
		peerID = dialog.UserLowID
	}
	peer, err := s.users.Get(ctx, peerID)
	if err != nil {
		return dialogResponse{}, err
	}
	last, unread, err := s.messaging.Messages.DialogSummary(ctx, actorID, dialog.ID)
	if err != nil {
		return dialogResponse{}, err
	}
	result := dialogResponse{
		ID: dialog.ID, CreatedAt: dialog.CreatedAt, UpdatedAt: dialog.UpdatedAt, UnreadCount: unread,
		Peer: peerResponse{ID: peer.ID, Username: peer.Username, Status: peer.ManualStatus, LastSeenAt: peer.LastSeenAt},
	}
	if last != nil {
		value := messageResult(*last)
		result.LastMessage = &value
	}
	return result, nil
}

func messageResult(value message.Message) messageResponse {
	reactions := make([]reactionResponse, 0, len(value.Reactions))
	for _, item := range value.Reactions {
		reactions = append(reactions, reactionResult(item))
	}
	result := messageResponse{
		ID: value.ID, DialogID: value.DialogID, SenderID: value.SenderID, Body: value.Body,
		CreatedAt: value.CreatedAt, ReadAt: value.ReadAt, Reactions: reactions,
	}
	if value.ReplyTo != nil {
		result.ReplyTo = &replyResponse{ID: value.ReplyTo.ID, SenderID: value.ReplyTo.SenderID, Excerpt: value.ReplyTo.Excerpt}
	}
	return result
}

func reactionResult(value reaction.Reaction) reactionResponse {
	return reactionResponse{
		MessageID: value.MessageID, UserID: value.UserID, Reaction: value.Type,
		CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt,
	}
}

func requestID(w http.ResponseWriter, name, value string) (uuid.UUID, bool) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		writeError(w, apperror.New(apperror.CodeInvalidArgument, name+" must be a UUID"))
		return uuid.Nil, false
	}
	return id, true
}

func requestPage(w http.ResponseWriter, r *http.Request) (size int, token string, ok bool) {
	query := r.URL.Query()
	if value := query.Get("page_size"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 {
			writeError(w, apperror.New(apperror.CodeInvalidArgument, "page_size must be a positive integer"))
			return 0, "", false
		}
		size = parsed
	}
	return size, query.Get("page_token"), true
}

func (s *Server) createDialog(w http.ResponseWriter, r *http.Request) {
	var request struct {
		OtherUserID string `json:"other_user_id"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	peerID, ok := requestID(w, "other_user_id", request.OtherUserID)
	if !ok {
		return
	}
	actorID, _ := session.UserID(r.Context())
	dialog, err := s.messaging.Dialogs.CreateDirectDialog(r.Context(), actorID, peerID)
	if err != nil {
		writeError(w, err)
		return
	}
	result, err := s.dialogResult(r.Context(), actorID, dialog)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) listDialogs(w http.ResponseWriter, r *http.Request) {
	size, token, ok := requestPage(w, r)
	if !ok {
		return
	}
	actorID, _ := session.UserID(r.Context())
	dialogs, next, err := s.messaging.Dialogs.ListDialogs(r.Context(), actorID, size, token)
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]dialogResponse, 0, len(dialogs))
	for _, dialog := range dialogs {
		item, err := s.dialogResult(r.Context(), actorID, dialog)
		if err != nil {
			writeError(w, err)
			return
		}
		items = append(items, item)
	}
	writeJSON(w, http.StatusOK, struct {
		Dialogs  []dialogResponse `json:"dialogs"`
		PageInfo pageInfo         `json:"page_info"`
	}{Dialogs: items, PageInfo: pageInfo{NextPageToken: next}})
}

func (s *Server) sendMessage(w http.ResponseWriter, r *http.Request) {
	dialogID, ok := requestID(w, "dialog_id", r.PathValue("dialog_id"))
	if !ok {
		return
	}
	var request struct {
		Body    string  `json:"body"`
		ReplyTo *string `json:"reply_to_message_id"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	actorID, _ := session.UserID(r.Context())
	var replyToID *uuid.UUID
	if request.ReplyTo != nil {
		replyID, ok := requestID(w, "reply_to_message_id", *request.ReplyTo)
		if !ok {
			return
		}
		replyToID = &replyID
	}
	result, err := s.messaging.Messages.SendMessage(r.Context(), actorID, dialogID, request.Body, replyToID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, messageResult(result))
}

func (s *Server) listMessages(w http.ResponseWriter, r *http.Request) {
	dialogID, ok := requestID(w, "dialog_id", r.PathValue("dialog_id"))
	if !ok {
		return
	}
	size, token, ok := requestPage(w, r)
	if !ok {
		return
	}
	actorID, _ := session.UserID(r.Context())
	values, next, err := s.messaging.Messages.ListMessages(r.Context(), actorID, dialogID, size, token)
	if err != nil {
		writeError(w, err)
		return
	}
	messages := make([]messageResponse, 0, len(values))
	for _, value := range values {
		messages = append(messages, messageResult(value))
	}
	writeJSON(w, http.StatusOK, struct {
		Messages []messageResponse `json:"messages"`
		PageInfo pageInfo          `json:"page_info"`
	}{Messages: messages, PageInfo: pageInfo{NextPageToken: next}})
}

func (s *Server) typing(w http.ResponseWriter, r *http.Request) {
	dialogID, ok := requestID(w, "dialog_id", r.PathValue("dialog_id"))
	if !ok {
		return
	}
	actorID, _ := session.UserID(r.Context())
	peer, err := s.messaging.Dialogs.PeerInDialog(r.Context(), dialogID, actorID)
	if err != nil {
		writeError(w, err)
		return
	}
	if r.Context().Err() != nil {
		return
	}
	s.hub.PublishTo([]uuid.UUID{peer}, event.Event{
		ID: uuid.New(), OccurredAt: time.Now().UTC(),
		Payload: event.DialogTyping{DialogID: dialogID, UserID: actorID},
	})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) markRead(w http.ResponseWriter, r *http.Request) {
	dialogID, ok := requestID(w, "dialog_id", r.PathValue("dialog_id"))
	if !ok {
		return
	}
	var request struct {
		UpToMessageID string `json:"up_to_message_id"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	messageID, ok := requestID(w, "up_to_message_id", request.UpToMessageID)
	if !ok {
		return
	}
	actorID, _ := session.UserID(r.Context())
	result, err := s.messaging.Receipts.MarkRead(r.Context(), actorID, dialogID, messageID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, struct {
		MarkedCount int64     `json:"marked_count"`
		ReadAt      time.Time `json:"read_at"`
	}{MarkedCount: result.MarkedCount, ReadAt: result.ReadAt})
}

func (s *Server) setReaction(w http.ResponseWriter, r *http.Request) {
	messageID, ok := requestID(w, "message_id", r.PathValue("message_id"))
	if !ok {
		return
	}
	var request struct {
		Reaction reaction.Type `json:"reaction"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	actorID, _ := session.UserID(r.Context())
	result, err := s.messaging.Reactions.SetReaction(r.Context(), actorID, messageID, request.Reaction)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, reactionResult(result))
}

func (s *Server) removeReaction(w http.ResponseWriter, r *http.Request) {
	messageID, ok := requestID(w, "message_id", r.PathValue("message_id"))
	if !ok {
		return
	}
	actorID, _ := session.UserID(r.Context())
	removed, err := s.messaging.Reactions.RemoveReaction(r.Context(), actorID, messageID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"removed": removed})
}
