// Package webapi exposes a small JSON/SSE adapter and an interactive API console.
// The gRPC services remain the primary transport; this package is intended for
// local development, documentation, and manual testing from a browser.
package webapi

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/francesandprincesss/new-telegram/internal/apperror"
	"github.com/francesandprincesss/new-telegram/internal/auth"
	"github.com/francesandprincesss/new-telegram/internal/event"
	"github.com/francesandprincesss/new-telegram/internal/presence"
	"github.com/francesandprincesss/new-telegram/internal/session"
	"github.com/francesandprincesss/new-telegram/internal/user"
	"github.com/google/uuid"
)

//go:embed static/*
var staticFiles embed.FS

type Server struct {
	auth     authService
	users    userService
	presence presenceService
	tokens   auth.TokenVerifier
	hub      eventSubscriber
	observer event.ConnectionObserver
	handler  http.Handler
}

type authService interface {
	Register(context.Context, string, string) (auth.Result, error)
	Login(context.Context, string, string) (auth.Result, error)
}

type userService interface {
	Get(context.Context, uuid.UUID) (user.User, error)
	Find(context.Context, string) (user.User, error)
}

type presenceService interface {
	SetStatus(context.Context, uuid.UUID, presence.Status) (presence.State, error)
}

type eventSubscriber interface {
	Subscribe(uuid.UUID) *event.Subscription
}

func New(authService authService, userService userService, presenceService presenceService, tokens auth.TokenVerifier, hub eventSubscriber, observer event.ConnectionObserver) *Server {
	s := &Server{auth: authService, users: userService, presence: presenceService, tokens: tokens, hub: hub, observer: observer}
	s.handler = s.routes()
	return s
}

func (s *Server) Handler() http.Handler { return s.handler }

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /openapi.json", s.serveOpenAPI)
	mux.HandleFunc("GET /docs", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/", http.StatusTemporaryRedirect) })
	mux.Handle("POST /api/v1/auth/register", requestTimeout(http.HandlerFunc(s.register)))
	mux.Handle("POST /api/v1/auth/login", requestTimeout(http.HandlerFunc(s.login)))
	mux.Handle("GET /api/v1/users/me", s.authenticate(requestTimeout(http.HandlerFunc(s.getMe))))
	mux.Handle("GET /api/v1/users/find", s.authenticate(requestTimeout(http.HandlerFunc(s.findUser))))
	mux.Handle("GET /api/v1/users/{user_id}", s.authenticate(requestTimeout(http.HandlerFunc(s.getUser))))
	mux.Handle("PUT /api/v1/presence/status", s.authenticate(requestTimeout(http.HandlerFunc(s.setStatus))))
	mux.Handle("GET /api/v1/events", s.authenticate(http.HandlerFunc(s.events)))
	assets, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic(err)
	}
	mux.Handle("GET /", http.FileServer(http.FS(assets)))
	return securityHeaders(mux)
}

func requestTimeout(next http.Handler) http.Handler {
	timed := http.TimeoutHandler(next, 15*time.Second, `{"error":"request timeout"}`)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		timed.ServeHTTP(w, r)
	})
}

type credentialsRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	var request credentialsRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	result, err := s.auth.Register(r.Context(), request.Username, request.Password)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, authResult(result))
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var request credentialsRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	result, err := s.auth.Login(r.Context(), request.Username, request.Password)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, authResult(result))
}

func (s *Server) getMe(w http.ResponseWriter, r *http.Request) {
	userID, _ := session.UserID(r.Context())
	result, err := s.users.Get(r.Context(), userID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, userResult(result))
}

func (s *Server) getUser(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(r.PathValue("user_id"))
	if err != nil || userID == uuid.Nil {
		writeError(w, apperror.New(apperror.CodeInvalidArgument, "user_id must be a UUID"))
		return
	}
	result, err := s.users.Get(r.Context(), userID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, userResult(result))
}

func (s *Server) findUser(w http.ResponseWriter, r *http.Request) {
	result, err := s.users.Find(r.Context(), r.URL.Query().Get("username"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, userResult(result))
}

func (s *Server) setStatus(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Status string `json:"status"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	userID, _ := session.UserID(r.Context())
	state, err := s.presence.SetStatus(r.Context(), userID, presence.Status(strings.ToUpper(request.Status)))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, presenceResult(state))
}

func (s *Server) events(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming is unavailable"})
		return
	}
	userID, _ := session.UserID(r.Context())
	subscription := s.hub.Subscribe(userID)
	if err := s.observer.Connected(r.Context(), userID); err != nil {
		subscription.Cancel()
		writeError(w, err)
		return
	}
	defer func() {
		subscription.Cancel()
		ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 5*time.Second)
		defer cancel()
		if err := s.observer.Disconnected(ctx, userID); err != nil {
			log.Printf("disconnect web presence for %s: %v", userID, err)
		}
	}()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	_, _ = io.WriteString(w, ": connected\n\n")
	flusher.Flush()
	heartbeat := time.NewTicker(20 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case value := <-subscription.Events:
			payload, err := json.Marshal(eventResult(value))
			if err != nil {
				continue
			}
			if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventName(value), payload); err != nil {
				return
			}
			flusher.Flush()
		case <-subscription.Done:
			return
		case <-heartbeat.C:
			if _, err := io.WriteString(w, ": keepalive\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (s *Server) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Fields(r.Header.Get("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authorization token is required"})
			return
		}
		userID, err := s.tokens.Verify(parts[1])
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid authorization token"})
			return
		}
		next.ServeHTTP(w, r.WithContext(session.WithUserID(r.Context(), userID)))
	})
}

func (s *Server) serveOpenAPI(w http.ResponseWriter, r *http.Request) {
	content, err := staticFiles.ReadFile("static/openapi.json")
	if err != nil {
		http.Error(w, "openapi document unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(content)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeError(w, apperror.New(apperror.CodeInvalidArgument, "Content-Type must be application/json"))
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, apperror.New(apperror.CodeInvalidArgument, "request body must be valid JSON"))
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, apperror.New(apperror.CodeInvalidArgument, "request body must contain one JSON object"))
		return false
	}
	return true
}

func writeError(w http.ResponseWriter, err error) {
	statusCode := http.StatusInternalServerError
	message := "internal error"
	var appErr *apperror.Error
	if errors.As(err, &appErr) {
		message = appErr.Message
		switch appErr.Code {
		case apperror.CodeInvalidArgument:
			statusCode = http.StatusBadRequest
		case apperror.CodeUnauthenticated:
			statusCode = http.StatusUnauthorized
		case apperror.CodeForbidden:
			statusCode = http.StatusForbidden
		case apperror.CodeNotFound:
			statusCode = http.StatusNotFound
		case apperror.CodeConflict:
			statusCode = http.StatusConflict
		}
	}
	writeJSON(w, statusCode, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, statusCode int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(value)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self'; script-src 'self'; connect-src 'self'")
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
		}
		next.ServeHTTP(w, r)
	})
}
