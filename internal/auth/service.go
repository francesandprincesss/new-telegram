package auth

// Service coordinates registration and login.
type Service struct {
	repository Repository
	passwords  PasswordHasher
	tokens     TokenIssuer
}

func NewService(repository Repository, passwords PasswordHasher, tokens TokenIssuer) *Service {
	return &Service{repository: repository, passwords: passwords, tokens: tokens}
}

// TODO: Implement Register and Login with normalized usernames and uniform
// invalid-credential errors that do not reveal whether an account exists.
