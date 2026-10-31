// Package apperror defines transport-independent application errors.
package apperror

import (
	"errors"
	"fmt"
)

// Code classifies an application error without coupling domain services to gRPC.
type Code string

const (
	CodeInvalidArgument Code = "invalid_argument"
	CodeUnauthenticated Code = "unauthenticated"
	CodeNotFound        Code = "not_found"
	CodeForbidden       Code = "forbidden"
	CodeConflict        Code = "conflict"
	CodeInternal        Code = "internal"
)

// Error is safe for service boundaries. Message must not contain secrets or
// database details; Cause is intended for internal logging only.
type Error struct {
	Code    Code
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e.Message == "" {
		return string(e.Code)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error { return e.Cause }

// New creates an application error without an underlying cause.
func New(code Code, message string) error {
	return &Error{Code: code, Message: message}
}

// Wrap creates an application error that retains an internal cause.
func Wrap(code Code, message string, cause error) error {
	return &Error{Code: code, Message: message, Cause: cause}
}

// CodeOf returns CodeInternal for errors that are not application errors.
func CodeOf(err error) Code {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr.Code
	}
	return CodeInternal
}
