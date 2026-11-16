package auth

import (
	"context"
	"strings"

	chatv1 "github.com/francesandprincesss/new-telegram/gen/chat/v1"
	"github.com/francesandprincesss/new-telegram/internal/session"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type Interceptor struct{ verifier TokenVerifier }

func NewInterceptor(verifier TokenVerifier) *Interceptor { return &Interceptor{verifier: verifier} }

func (i *Interceptor) Unary() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if isPublic(info.FullMethod) {
			return handler(ctx, req)
		}
		authenticated, err := i.authenticate(ctx)
		if err != nil {
			return nil, err
		}
		return handler(authenticated, req)
	}
}

func (i *Interceptor) Stream() grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if isPublic(info.FullMethod) {
			return handler(srv, stream)
		}
		ctx, err := i.authenticate(stream.Context())
		if err != nil {
			return err
		}
		return handler(srv, &authenticatedStream{ServerStream: stream, ctx: ctx})
	}
}

func (i *Interceptor) authenticate(ctx context.Context) (context.Context, error) {
	values := metadata.ValueFromIncomingContext(ctx, "authorization")
	if len(values) != 1 {
		return nil, status.Error(codes.Unauthenticated, "authorization token is required")
	}
	parts := strings.Fields(values[0])
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return nil, status.Error(codes.Unauthenticated, "invalid authorization header")
	}
	userID, err := i.verifier.Verify(parts[1])
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid authorization token")
	}
	return session.WithUserID(ctx, userID), nil
}

func isPublic(method string) bool {
	return method == chatv1.AuthService_Register_FullMethodName || method == chatv1.AuthService_Login_FullMethodName
}

type authenticatedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *authenticatedStream) Context() context.Context { return s.ctx }
