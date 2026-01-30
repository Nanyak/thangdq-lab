package usecase

import (
	"context"

	"github.com/Nanyak/thangdq-lab/internal/entity"
)

// AuthProvider defines the interface for authentication operations
type AuthProvider interface {
	SignUp(ctx context.Context, email, password, name string) (*entity.AuthResult, error)
	ConfirmSignUp(ctx context.Context, email, confirmationCode string) error
	SignIn(ctx context.Context, email, password string) (*entity.AuthTokens, error)
	ForgotPassword(ctx context.Context, email string) error
	ConfirmForgotPassword(ctx context.Context, email, code, newPassword string) error
}

// Auth wraps the AuthProvider interface for business logic
type Auth struct {
	provider AuthProvider
}

// NewAuth creates a new Auth instance
func NewAuth(provider AuthProvider) *Auth {
	return &Auth{
		provider: provider,
	}
}

// SignUp registers a new user
func (a *Auth) SignUp(ctx context.Context, email, password, name string) (*entity.AuthResult, error) {
	return a.provider.SignUp(ctx, email, password, name)
}

// ConfirmSignUp confirms a user's registration with a verification code
func (a *Auth) ConfirmSignUp(ctx context.Context, email, confirmationCode string) error {
	return a.provider.ConfirmSignUp(ctx, email, confirmationCode)
}

// SignIn authenticates a user and returns tokens
func (a *Auth) SignIn(ctx context.Context, email, password string) (*entity.AuthTokens, error) {
	return a.provider.SignIn(ctx, email, password)
}

// ForgotPassword initiates a password reset flow
func (a *Auth) ForgotPassword(ctx context.Context, email string) error {
	return a.provider.ForgotPassword(ctx, email)
}

// ConfirmForgotPassword completes the password reset with a confirmation code
func (a *Auth) ConfirmForgotPassword(ctx context.Context, email, code, newPassword string) error {
	return a.provider.ConfirmForgotPassword(ctx, email, code, newPassword)
}
