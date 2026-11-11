package grpcutil

import (
	"errors"

	"github.com/francesandprincesss/new-telegram/internal/apperror"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Error(err error) error {
	if err == nil {
		return nil
	}
	var appErr *apperror.Error
	if !errors.As(err, &appErr) {
		return status.Error(codes.Internal, "internal error")
	}
	code := codes.Internal
	switch appErr.Code {
	case apperror.CodeInvalidArgument:
		code = codes.InvalidArgument
	case apperror.CodeUnauthenticated:
		code = codes.Unauthenticated
	case apperror.CodeNotFound:
		code = codes.NotFound
	case apperror.CodeForbidden:
		code = codes.PermissionDenied
	case apperror.CodeConflict:
		code = codes.AlreadyExists
	}
	return status.Error(code, appErr.Message)
}
