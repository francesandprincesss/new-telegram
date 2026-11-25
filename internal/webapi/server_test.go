package webapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/francesandprincesss/new-telegram/internal/apperror"
	"github.com/francesandprincesss/new-telegram/internal/auth"
	"github.com/francesandprincesss/new-telegram/internal/event"
	"github.com/francesandprincesss/new-telegram/internal/presence"
	"github.com/francesandprincesss/new-telegram/internal/user"
	"github.com/google/uuid"
)

type authStub struct{ result auth.Result }

func (s authStub) Register(context.Context, string, string) (auth.Result, error) {
	return s.result, nil
}
func (s authStub) Login(context.Context, string, string) (auth.Result, error) { return s.result, nil }

type userStub struct{ result user.User }

func (s userStub) Get(context.Context, uuid.UUID) (user.User, error) { return s.result, nil }
func (s userStub) Find(context.Context, string) (user.User, error)   { return s.result, nil }

type presenceStub struct{}

func (presenceStub) SetStatus(_ context.Context, userID uuid.UUID, status presence.Status) (presence.State, error) {
	return presence.State{UserID: userID, Status: status}, nil
}

type tokenStub struct{ userID uuid.UUID }

func (s tokenStub) Verify(token string) (uuid.UUID, error) { return s.userID, nil }

type invalidTokenStub struct{}

func (invalidTokenStub) Verify(string) (uuid.UUID, error) { return uuid.Nil, errors.New("invalid") }

type expiringTokenStub struct {
	tokenStub
	expiry time.Time
}

func (s expiringTokenStub) VerifyWithExpiry(string) (uuid.UUID, time.Time, error) {
	return s.userID, s.expiry, nil
}

type observerStub struct {
	mu           sync.Mutex
	connected    int
	disconnected int
}

func (o *observerStub) Connected(context.Context, uuid.UUID) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.connected++
	return nil
}
func (o *observerStub) Disconnected(context.Context, uuid.UUID) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.disconnected++
	return nil
}

func newTestServer() (*Server, uuid.UUID, *observerStub) {
	userID := uuid.New()
	now := time.Unix(1_700_000_000, 0).UTC()
	observer := &observerStub{}
	server := New(
		authStub{result: auth.Result{UserID: userID, Username: "alice", CreatedAt: now, Token: "token", ExpiresAt: now.Add(time.Hour)}},
		userStub{result: user.User{ID: userID, Username: "alice", ManualStatus: string(presence.Online), CreatedAt: now}},
		presenceStub{}, tokenStub{userID: userID}, event.NewHub(4), observer, Messaging{},
	)
	return server, userID, observer
}

func TestDocumentationAndOpenAPIAreEmbedded(t *testing.T) {
	server, _, _ := newTestServer()
	for _, path := range []string{"/", "/styles.css", "/app.js", "/openapi.json"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		response := httptest.NewRecorder()
		server.Handler().ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d", path, response.Code)
		}
		if response.Body.Len() == 0 {
			t.Fatalf("GET %s returned an empty body", path)
		}
	}
	var document map[string]any
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/openapi.json", nil))
	if err := json.Unmarshal(response.Body.Bytes(), &document); err != nil {
		t.Fatalf("OpenAPI is invalid JSON: %v", err)
	}
	if document["openapi"] != "3.1.0" {
		t.Fatalf("OpenAPI version = %v", document["openapi"])
	}
}

func TestProtectedRouteRequiresBearerToken(t *testing.T) {
	server, _, _ := newTestServer()
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", response.Code)
	}
}

func TestProtectedRouteRejectsInvalidToken(t *testing.T) {
	server, _, observer := newTestServer()
	server = New(server.auth, server.users, server.presence, invalidTokenStub{}, server.hub, observer, Messaging{})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	request.Header.Set("Authorization", "Bearer invalid")
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", response.Code)
	}
}

func TestApplicationErrorHTTPMapping(t *testing.T) {
	tests := []struct {
		code apperror.Code
		want int
	}{
		{apperror.CodeInvalidArgument, http.StatusBadRequest},
		{apperror.CodeUnauthenticated, http.StatusUnauthorized},
		{apperror.CodeForbidden, http.StatusForbidden},
		{apperror.CodeNotFound, http.StatusNotFound},
		{apperror.CodeConflict, http.StatusConflict},
		{apperror.CodeInternal, http.StatusInternalServerError},
	}
	for _, tc := range tests {
		response := httptest.NewRecorder()
		writeError(response, apperror.New(tc.code, "safe message"))
		if response.Code != tc.want || !strings.Contains(response.Body.String(), "safe message") {
			t.Fatalf("code %s: response = %d %s", tc.code, response.Code, response.Body.String())
		}
	}
}

func TestRegisterAndGetMe(t *testing.T) {
	server, userID, _ := newTestServer()
	register := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(`{"username":"alice","password":"password123"}`))
	register.Header.Set("Content-Type", "application/json")
	registerResponse := httptest.NewRecorder()
	server.Handler().ServeHTTP(registerResponse, register)
	if registerResponse.Code != http.StatusCreated || !strings.Contains(registerResponse.Body.String(), `"access_token":"token"`) {
		t.Fatalf("register response = %d %s", registerResponse.Code, registerResponse.Body.String())
	}

	getMe := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	getMe.Header.Set("Authorization", "Bearer token")
	getMeResponse := httptest.NewRecorder()
	server.Handler().ServeHTTP(getMeResponse, getMe)
	if getMeResponse.Code != http.StatusOK || !strings.Contains(getMeResponse.Body.String(), userID.String()) {
		t.Fatalf("get me response = %d %s", getMeResponse.Code, getMeResponse.Body.String())
	}
}

func TestJSONBodyValidation(t *testing.T) {
	server, _, _ := newTestServer()
	tests := []struct {
		name        string
		contentType string
		body        string
	}{
		{name: "wrong content type", contentType: "text/plain", body: `{}`},
		{name: "unknown field", contentType: "application/json", body: `{"username":"alice","password":"password123","admin":true}`},
		{name: "two objects", contentType: "application/json", body: `{} {}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(tc.body))
			request.Header.Set("Content-Type", tc.contentType)
			response := httptest.NewRecorder()
			server.Handler().ServeHTTP(response, request)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", response.Code)
			}
		})
	}
}

func TestEventStreamTracksConnectionLifecycle(t *testing.T) {
	server, _, observer := newTestServer()
	ctx, cancel := context.WithCancel(context.Background())
	request := httptest.NewRequest(http.MethodGet, "/api/v1/events", nil).WithContext(ctx)
	request.Header.Set("Authorization", "Bearer token")
	response := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		server.Handler().ServeHTTP(response, request)
		close(done)
	}()
	waitFor := time.Now().Add(time.Second)
	for time.Now().Before(waitFor) {
		observer.mu.Lock()
		connected := observer.connected
		observer.mu.Unlock()
		if connected == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("event stream did not stop after request cancellation")
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		observer.mu.Lock()
		connected, disconnected := observer.connected, observer.disconnected
		observer.mu.Unlock()
		if connected == 1 && disconnected == 1 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("event stream did not call Connected and Disconnected exactly once")
}

func TestEventStreamClosesAtTokenExpiry(t *testing.T) {
	server, userID, observer := newTestServer()
	server.tokens = expiringTokenStub{tokenStub{userID}, time.Now().Add(50 * time.Millisecond)}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/events", nil)
	request.Header.Set("Authorization", "Bearer token")
	response := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		server.Handler().ServeHTTP(response, request)
		close(done)
	}()
	select {
	case <-done:
		if response.Code != http.StatusOK || observer.connected != 1 || observer.disconnected != 1 {
			t.Fatalf("stream response %d, connected %d, disconnected %d", response.Code, observer.connected, observer.disconnected)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("event stream did not stop at the token deadline")
	}
}
