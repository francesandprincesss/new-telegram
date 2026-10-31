package apperror

import (
	"errors"
	"testing"
)

func TestErrorExposesCodeAndCause(t *testing.T) {
	cause := errors.New("database unavailable")
	err := Wrap(CodeInternal, "load user", cause)

	if got := CodeOf(err); got != CodeInternal {
		t.Fatalf("CodeOf() = %q, want %q", got, CodeInternal)
	}
	if !errors.Is(err, cause) {
		t.Fatal("wrapped error does not expose its cause")
	}
}

func TestCodeOfUnknownErrorReturnsInternal(t *testing.T) {
	if got := CodeOf(errors.New("unknown")); got != CodeInternal {
		t.Fatalf("CodeOf() = %q, want %q", got, CodeInternal)
	}
}

func TestNewCreatesSafeErrorWithoutCause(t *testing.T) {
	err := New(CodeInvalidArgument, "invalid username")

	if got := CodeOf(err); got != CodeInvalidArgument {
		t.Fatalf("CodeOf() = %q, want %q", got, CodeInvalidArgument)
	}
	if got := err.Error(); got != "invalid_argument: invalid username" {
		t.Fatalf("Error() = %q", got)
	}
	if errors.Unwrap(err) != nil {
		t.Fatal("New() unexpectedly exposes an underlying cause")
	}
}

func TestErrorWithoutMessageUsesCode(t *testing.T) {
	err := New(CodeNotFound, "")
	if got := err.Error(); got != string(CodeNotFound) {
		t.Fatalf("Error() = %q, want %q", got, CodeNotFound)
	}
}
