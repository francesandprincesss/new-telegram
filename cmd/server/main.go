package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os/signal"
	"strings"
	"syscall"
	"time"

	chatv1 "github.com/francesandprincesss/new-telegram/gen/chat/v1"
	"github.com/francesandprincesss/new-telegram/internal/auth"
	"github.com/francesandprincesss/new-telegram/internal/config"
	"github.com/francesandprincesss/new-telegram/internal/database"
	"github.com/francesandprincesss/new-telegram/internal/event"
	"github.com/francesandprincesss/new-telegram/internal/presence"
	"github.com/francesandprincesss/new-telegram/internal/user"
	"github.com/francesandprincesss/new-telegram/internal/webapi"
	"google.golang.org/grpc"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, sqlDB, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	hub := event.NewHub(64)
	presenceRepository := presence.NewRepository(db)
	presenceManager := presence.NewManager(presenceRepository, presence.NewPeerRepository(db), hub)
	presenceService := presence.NewService(presenceManager)
	userService := user.NewService(user.NewRepository(db), presenceManager)
	jwtManager := auth.NewJWTManager(cfg.JWT.Secret, cfg.JWT.Issuer, cfg.JWT.TTL)
	authService := auth.NewService(auth.NewRepository(db), auth.NewBcryptHasher(0), jwtManager)
	authInterceptor := auth.NewInterceptor(jwtManager)

	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(authInterceptor.Unary()), grpc.StreamInterceptor(authInterceptor.Stream()))
	chatv1.RegisterAuthServiceServer(grpcServer, auth.NewHandler(authService))
	chatv1.RegisterUserServiceServer(grpcServer, user.NewHandler(userService))
	chatv1.RegisterPresenceServiceServer(grpcServer, presence.NewHandler(presenceService))
	chatv1.RegisterEventServiceServer(grpcServer, event.NewHandler(hub, presenceManager))
	httpServer := &http.Server{
		Handler:           webapi.New(authService, userService, presenceService, jwtManager, hub, presenceManager).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	grpcListener, err := net.Listen("tcp", cfg.GRPCAddress)
	if err != nil {
		return err
	}
	defer grpcListener.Close()
	httpListener, err := net.Listen("tcp", cfg.HTTPAddress)
	if err != nil {
		return err
	}
	defer httpListener.Close()

	type serverError struct {
		name string
		err  error
	}
	serveErr := make(chan serverError, 2)
	go func() { serveErr <- serverError{name: "gRPC", err: grpcServer.Serve(grpcListener)} }()
	go func() { serveErr <- serverError{name: "HTTP", err: httpServer.Serve(httpListener)} }()
	log.Printf("gRPC server listening on %s", cfg.GRPCAddress)
	docsHost := cfg.HTTPAddress
	if strings.HasPrefix(docsHost, ":") {
		docsHost = "localhost" + docsHost
	}
	log.Printf("interactive API docs available at http://%s/docs", docsHost)

	var runtimeErr error
	select {
	case result := <-serveErr:
		if !errors.Is(result.err, grpc.ErrServerStopped) && !errors.Is(result.err, http.ErrServerClosed) {
			runtimeErr = fmt.Errorf("%s server: %w", result.name, result.err)
		}
	case <-ctx.Done():
	}

	hub.Close()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	httpStopped := make(chan error, 1)
	grpcStopped := make(chan struct{})
	go func() { httpStopped <- httpServer.Shutdown(shutdownCtx) }()
	go func() { grpcServer.GracefulStop(); close(grpcStopped) }()
	var httpErr error
	for httpStopped != nil || grpcStopped != nil {
		select {
		case httpErr = <-httpStopped:
			httpStopped = nil
		case <-grpcStopped:
			grpcStopped = nil
		case <-shutdownCtx.Done():
			grpcServer.Stop()
			_ = httpServer.Close()
			if runtimeErr != nil {
				return runtimeErr
			}
			return fmt.Errorf("graceful shutdown: %w", shutdownCtx.Err())
		}
	}
	if runtimeErr != nil {
		return runtimeErr
	}
	if httpErr != nil {
		return httpErr
	}
	return nil
}
