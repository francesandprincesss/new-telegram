package auth

import (
	"context"
	"testing"

	chatv1 "github.com/francesandprincesss/new-telegram/gen/chat/v1"
	"github.com/francesandprincesss/new-telegram/internal/session"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type verifierStub struct {
	userID uuid.UUID
	err    error
}

func (v verifierStub) Verify(string) (uuid.UUID, error) { return v.userID, v.err }

func TestUnaryInterceptorAllowsOnlyPublicAuthMethods(t *testing.T) {
	interceptor := NewInterceptor(verifierStub{}).Unary()
	called := false
	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: chatv1.AuthService_Register_FullMethodName}, func(context.Context, any) (any, error) {
		called = true
		return nil, nil
	})
	if err != nil || !called {
		t.Fatalf("public method: called=%v err=%v", called, err)
	}

	_, err = interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: chatv1.UserService_GetMe_FullMethodName}, func(context.Context, any) (any, error) {
		t.Fatal("protected handler was called without a token")
		return nil, nil
	})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("protected method code = %v", status.Code(err))
	}
}

func TestUnaryInterceptorAddsVerifiedIdentity(t *testing.T) {
	userID := uuid.New()
	interceptor := NewInterceptor(verifierStub{userID: userID}).Unary()
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer valid-token"))
	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: chatv1.UserService_GetMe_FullMethodName}, func(ctx context.Context, _ any) (any, error) {
		got, err := session.UserID(ctx)
		if err != nil || got != userID {
			t.Fatalf("session.UserID() = %v, %v", got, err)
		}
		return nil, nil
	})
	if err != nil {
		t.Fatalf("interceptor error = %v", err)
	}
}
