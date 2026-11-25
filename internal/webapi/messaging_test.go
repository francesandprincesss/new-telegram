package webapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/francesandprincesss/new-telegram/internal/apperror"
	"github.com/francesandprincesss/new-telegram/internal/chat"
	"github.com/francesandprincesss/new-telegram/internal/message"
	"github.com/francesandprincesss/new-telegram/internal/reaction"
	"github.com/francesandprincesss/new-telegram/internal/readreceipt"
	"github.com/francesandprincesss/new-telegram/internal/user"
	"github.com/google/uuid"
)

type messagingStub struct {
	actorID uuid.UUID
	peerID  uuid.UUID
	dialog  chat.Dialog
	message message.Message
}

func (s messagingStub) CreateDirectDialog(_ context.Context, actor, peer uuid.UUID) (chat.Dialog, error) {
	if actor != s.actorID || peer != s.peerID {
		return chat.Dialog{}, apperror.New(apperror.CodeForbidden, "invalid participants")
	}
	return s.dialog, nil
}

func (s messagingStub) ListDialogs(_ context.Context, actor uuid.UUID, size int, token string) ([]chat.Dialog, string, error) {
	if actor != s.actorID || size != 5 || token != "dial" {
		return nil, "", apperror.New(apperror.CodeInvalidArgument, "invalid dialog page")
	}
	return []chat.Dialog{s.dialog}, "dial-next", nil
}

func (s messagingStub) SendMessage(_ context.Context, actor, dialog uuid.UUID, body string, _ *uuid.UUID) (message.Message, error) {
	if actor != s.actorID || dialog != s.dialog.ID {
		return message.Message{}, apperror.New(apperror.CodeForbidden, "not a participant")
	}
	return message.Message{ID: s.message.ID, SenderID: actor, DialogID: dialog, Body: body, CreatedAt: s.message.CreatedAt}, nil
}

func (s messagingStub) ListMessages(_ context.Context, actor, dialog uuid.UUID, size int, token string) ([]message.Message, string, error) {
	if actor != s.actorID || dialog != s.dialog.ID {
		return nil, "", apperror.New(apperror.CodeForbidden, "not a participant")
	}
	if size != 3 || token != "hist" {
		return nil, "", apperror.New(apperror.CodeInvalidArgument, "invalid history page")
	}
	return []message.Message{s.message}, "hist-next", nil
}

func (s messagingStub) DialogSummary(_ context.Context, actor, dialog uuid.UUID) (*message.Message, int64, error) {
	if actor != s.actorID || dialog != s.dialog.ID {
		return nil, 0, apperror.New(apperror.CodeForbidden, "not a participant")
	}
	return &s.message, 2, nil
}

func (s messagingStub) MarkRead(_ context.Context, actor, dialog, target uuid.UUID) (readreceipt.MarkResult, error) {
	if actor != s.actorID || dialog != s.dialog.ID || target != s.message.ID {
		return readreceipt.MarkResult{}, apperror.New(apperror.CodeForbidden, "not a participant")
	}
	return readreceipt.MarkResult{MarkedCount: 2, ReadAt: *s.message.ReadAt}, nil
}

func (s messagingStub) SetReaction(_ context.Context, actor, target uuid.UUID, kind reaction.Type) (reaction.Reaction, error) {
	if actor != s.actorID || target != s.message.ID {
		return reaction.Reaction{}, apperror.New(apperror.CodeForbidden, "not a participant")
	}
	return reaction.Reaction{MessageID: target, UserID: actor, Type: kind, CreatedAt: s.message.CreatedAt, UpdatedAt: s.message.CreatedAt}, nil
}

func (s messagingStub) RemoveReaction(_ context.Context, actor, target uuid.UUID) (bool, error) {
	return actor == s.actorID && target == s.message.ID, nil
}

func TestMessagingRoutes(t *testing.T) {
	server, actor, _ := newTestServer()
	peer := uuid.New()
	dialogID := uuid.New()
	messageID := uuid.New()
	created := time.Unix(1_700_000_000, 0).UTC()
	read := created.Add(time.Second)
	store := messagingStub{
		actorID: actor, peerID: peer,
		dialog: chat.Dialog{ID: dialogID, UserLowID: peer, UserHighID: actor, CreatedAt: created, UpdatedAt: created},
		message: message.Message{
			ID: messageID, SenderID: peer, DialogID: dialogID, Body: "Привет", CreatedAt: created, ReadAt: &read,
			Reactions: []reaction.Reaction{{MessageID: messageID, UserID: actor, Type: reaction.Like, CreatedAt: created, UpdatedAt: created}},
		},
	}
	server = New(server.auth, userStub{result: user.User{ID: peer, Username: "bob", ManualStatus: "ONLINE"}}, server.presence,
		server.tokens, server.hub, server.observer, Messaging{Dialogs: store, Messages: store, Receipts: store, Reactions: store})
	dialogPath := "/api/v1/dialogs/" + dialogID.String()
	reactionPath := "/api/v1/messages/" + messageID.String() + "/reaction"

	tests := []struct {
		name, method, path, body string
		want                     int
		fields                   []string
	}{
		{"create", http.MethodPost, "/api/v1/dialogs", `{"other_user_id":"` + peer.String() + `"}`, http.StatusOK,
			[]string{`"id":"` + dialogID.String() + `"`, `"username":"bob"`, `"unread_count":2`, `"last_message":{`}},
		{"dialogs", http.MethodGet, "/api/v1/dialogs?page_size=5&page_token=dial", "", http.StatusOK,
			[]string{`"dialogs":[`, `"next_page_token":"dial-next"`}},
		{"send", http.MethodPost, dialogPath + "/messages", `{"body":"Новый текст"}`, http.StatusCreated,
			[]string{`"sender_id":"` + actor.String() + `"`, `"body":"Новый текст"`, `"read_at":null`, `"reactions":[]`}},
		{"history", http.MethodGet, dialogPath + "/messages?page_size=3&page_token=hist", "", http.StatusOK,
			[]string{`"messages":[`, `"reaction":"LIKE"`, `"read_at":"` + read.Format(time.RFC3339) + `"`, `"next_page_token":"hist-next"`}},
		{"mark read", http.MethodPost, dialogPath + "/read", `{"up_to_message_id":"` + messageID.String() + `"}`, http.StatusOK,
			[]string{`"marked_count":2`, `"read_at":"` + read.Format(time.RFC3339) + `"`}},
		{"reaction", http.MethodPut, reactionPath, `{"reaction":"WOW"}`, http.StatusOK,
			[]string{`"message_id":"` + messageID.String() + `"`, `"reaction":"WOW"`}},
		{"remove reaction", http.MethodDelete, reactionPath, "", http.StatusOK, []string{`"removed":true`}},
		{"invalid UUID", http.MethodPost, "/api/v1/dialogs", `{"other_user_id":"invalid"}`, http.StatusBadRequest,
			[]string{`"error":"other_user_id must be a UUID"`}},
		{"invalid page size", http.MethodGet, "/api/v1/dialogs?page_size=-5", "", http.StatusBadRequest,
			[]string{`"error":"page_size must be a positive integer"`}},
		{"forbidden history", http.MethodGet, "/api/v1/dialogs/" + uuid.NewString() + "/messages", "", http.StatusForbidden,
			[]string{`"error":"not a participant"`}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Authorization", "Bearer demo")
			if tc.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			response := httptest.NewRecorder()
			server.Handler().ServeHTTP(response, req)
			body := response.Body.String()
			if response.Code != tc.want || !json.Valid(response.Body.Bytes()) {
				t.Fatalf("response = %d %s, want %d with JSON", response.Code, body, tc.want)
			}
			for _, field := range tc.fields {
				if !strings.Contains(body, field) {
					t.Errorf("response %s does not contain %s", body, field)
				}
			}
		})
	}
}

func TestMessagingRoutesRequireBearerToken(t *testing.T) {
	server, _, _ := newTestServer()
	for _, endpoint := range []struct {
		method, path string
	}{
		{http.MethodGet, "/api/v1/dialogs"},
		{http.MethodPost, "/api/v1/dialogs"},
		{http.MethodGet, "/api/v1/dialogs/invalid/messages"},
		{http.MethodPost, "/api/v1/dialogs/invalid/messages"},
		{http.MethodPost, "/api/v1/dialogs/invalid/read"},
		{http.MethodPut, "/api/v1/messages/invalid/reaction"},
		{http.MethodDelete, "/api/v1/messages/invalid/reaction"},
	} {
		request := httptest.NewRequest(endpoint.method, endpoint.path, nil)
		response := httptest.NewRecorder()
		server.Handler().ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized {
			t.Errorf("%s %s status = %d, want 401", endpoint.method, endpoint.path, response.Code)
		}
	}
}
