package auth

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/francesandprincesss/new-telegram/internal/apperror"
	"github.com/google/uuid"
)

var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9_]{3,32}$`)

// Service coordinates registration and login.
type Service struct {
	repository Repository
	passwords  PasswordHasher
	tokens     TokenIssuer
	dummyHash  string
}

func NewService(repository Repository, passwords PasswordHasher, tokens TokenIssuer) *Service {
	dummyHash, _ := passwords.Hash("invalid-password-placeholder")
	return &Service{repository: repository, passwords: passwords, tokens: tokens, dummyHash: dummyHash}
}

func (s *Service) Register(ctx context.Context, username, password string) (Result, error) {
	if !usernamePattern.MatchString(username) {
		return Result{}, apperror.New(apperror.CodeInvalidArgument, "username must be 3-32 ASCII letters, digits, or underscores")
	}
	if err := validatePassword(password); err != nil {
		return Result{}, apperror.New(apperror.CodeInvalidArgument, err.Error())
	}
	hash, err := s.passwords.Hash(password)
	if err != nil {
		return Result{}, apperror.Wrap(apperror.CodeInternal, "could not hash password", err)
	}
	now := time.Now().UTC()
	account, err := s.repository.CreateAccount(ctx, NewAccount{ID: uuid.New(), Username: username, UsernameNormalized: strings.ToLower(username), PasswordHash: hash, CreatedAt: now})
	if err != nil {
		return Result{}, err
	}
	return s.issue(account.ID, account.Username, account.CreatedAt)
}

func (s *Service) Login(ctx context.Context, username, password string) (Result, error) {
	invalid := apperror.New(apperror.CodeUnauthenticated, "invalid username or password")
	if !usernamePattern.MatchString(username) || validatePassword(password) != nil {
		return Result{}, invalid
	}
	credentials, err := s.repository.FindCredentialsByNormalizedUsername(ctx, strings.ToLower(username))
	if err != nil {
		if apperror.CodeOf(err) == apperror.CodeNotFound {
			_ = s.passwords.Compare(s.dummyHash, password)
			return Result{}, invalid
		}
		return Result{}, err
	}
	if err := s.passwords.Compare(credentials.PasswordHash, password); err != nil {
		return Result{}, invalid
	}
	return s.issue(credentials.UserID, credentials.Username, credentials.CreatedAt)
}

func (s *Service) issue(userID uuid.UUID, username string, createdAt time.Time) (Result, error) {
	token, expiresAt, err := s.tokens.Issue(userID)
	if err != nil {
		return Result{}, apperror.Wrap(apperror.CodeInternal, "could not issue token", err)
	}
	return Result{UserID: userID, Username: username, CreatedAt: createdAt, Token: token, ExpiresAt: expiresAt}, nil
}
