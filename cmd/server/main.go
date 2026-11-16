package main

import (
	"context"
	"errors"
	"log"
	"net"
	"os/signal"
	"syscall"
	"time"

	chatv1 "github.com/francesandprincesss/new-telegram/gen/chat/v1"
	"github.com/francesandprincesss/new-telegram/internal/auth"
	"github.com/francesandprincesss/new-telegram/internal/config"
	"github.com/francesandprincesss/new-telegram/internal/database"
	"github.com/francesandprincesss/new-telegram/internal/event"
	"github.com/francesandprincesss/new-telegram/internal/presence"
	"github.com/francesandprincesss/new-telegram/internal/user"
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

	server := grpc.NewServer(grpc.UnaryInterceptor(authInterceptor.Unary()), grpc.StreamInterceptor(authInterceptor.Stream()))
	chatv1.RegisterAuthServiceServer(server, auth.NewHandler(authService))
	chatv1.RegisterUserServiceServer(server, user.NewHandler(userService))
	chatv1.RegisterPresenceServiceServer(server, presence.NewHandler(presenceService))
	chatv1.RegisterEventServiceServer(server, event.NewHandler(hub, presenceManager))

	listener, err := net.Listen("tcp", cfg.GRPCAddress)
	if err != nil {
		return err
	}
	defer listener.Close()
	serveErr := make(chan error, 1)
	go func() { serveErr <- server.Serve(listener) }()
	log.Printf("gRPC server listening on %s", cfg.GRPCAddress)

	select {
	case err := <-serveErr:
		if errors.Is(err, grpc.ErrServerStopped) {
			return nil
		}
		return err
	case <-ctx.Done():
	}

	hub.Close()
	stopped := make(chan struct{})
	go func() { server.GracefulStop(); close(stopped) }()
	select {
	case <-stopped:
	case <-time.After(10 * time.Second):
		server.Stop()
	}
	return nil
}
