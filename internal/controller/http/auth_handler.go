package http

import (
	"context"
	"net/http"
	"strings"

	"github.com/Nanyak/thangdq-lab/internal/entity"
	"github.com/Nanyak/thangdq-lab/pkg/errors"
	"github.com/gin-gonic/gin"
)

// AuthService defines the auth use case interface
type AuthService interface {
	SignUp(ctx context.Context, username, email, password, name string) (*entity.AuthResult, error)
	ConfirmSignUp(ctx context.Context, email, confirmationCode string) error
	SignIn(ctx context.Context, email, password string) (*entity.AuthTokens, error)
	ForgotPassword(ctx context.Context, email string) error
	ConfirmForgotPassword(ctx context.Context, email, code, newPassword string) error
	GetUser(ctx context.Context, accessToken string) (*entity.User, error)
	RefreshToken(ctx context.Context, refreshToken string) (*entity.AuthTokens, error)
	SignOut(ctx context.Context, accessToken string) error
	ValidateToken(tokenString string) (map[string]interface{}, error)
}

// AuthHandler handles authentication HTTP requests
type AuthHandler struct {
	auth AuthService
}

// NewAuthHandler creates a new AuthHandler instance
func NewAuthHandler(auth AuthService) *AuthHandler {
	return &AuthHandler{
		auth: auth,
	}
}

// SignUp handles user registration
func (h *AuthHandler) SignUp(c *gin.Context) {
	var req SignUpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.auth.SignUp(c.Request.Context(), req.Username, req.Email, req.Password, req.Name)
	if err != nil {
		if errors.IsUserAlreadyExists(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "User already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sign up"})
		return
	}

	c.JSON(http.StatusCreated, SignUpResponse{
		UserID:    result.UserID,
		Confirmed: result.Confirmed,
	})
}

// ConfirmSignUp handles sign-up confirmation
func (h *AuthHandler) ConfirmSignUp(c *gin.Context) {
	var req ConfirmSignUpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.auth.ConfirmSignUp(c.Request.Context(), req.Email, req.ConfirmationCode); err != nil {
		if errors.IsCodeMismatch(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid confirmation code"})
			return
		}
		if errors.IsUserAlreadyExists(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "User already confirmed"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to confirm sign up"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Account confirmed successfully"})
}

// SignIn handles user authentication
func (h *AuthHandler) SignIn(c *gin.Context) {
	var req SignInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tokens, err := h.auth.SignIn(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if errors.IsInvalidCredentials(err) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}
		if errors.IsUserNotConfirmed(err) {
			c.JSON(http.StatusForbidden, gin.H{"error": "User is not confirmed"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sign in"})
		return
	}

	c.JSON(http.StatusOK, AuthTokensResponse{
		AccessToken:  tokens.AccessToken,
		IDToken:      tokens.IDToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
	})
}

// ForgotPassword initiates a password reset
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.auth.ForgotPassword(c.Request.Context(), req.Email); err != nil {
		if errors.IsUserNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initiate password reset"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password reset code sent"})
}

// ConfirmForgotPassword completes the password reset
func (h *AuthHandler) ConfirmForgotPassword(c *gin.Context) {
	var req ConfirmForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.auth.ConfirmForgotPassword(c.Request.Context(), req.Email, req.Code, req.NewPassword); err != nil {
		if errors.IsCodeMismatch(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid confirmation code"})
			return
		}
		if errors.IsUserNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully"})
}

func (h *AuthHandler) GetUser(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing authorization header"})
		return
	}

	accessToken := strings.TrimPrefix(authHeader, "Bearer ")
	if accessToken == authHeader {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format"})
		return
	}

	user, err := h.auth.GetUser(c.Request.Context(), accessToken)
	if err != nil {
		if errors.IsUserNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		if errors.IsInvalidCredentials(err) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user"})
		return
	}

	c.JSON(http.StatusOK, UserResponse{
		ID:            user.ID,
		Username:      user.Username,
		Email:         user.Email,
		Name:          user.Name,
		EmailVerified: user.EmailVerified,
	})
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tokens, err := h.auth.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		if errors.IsInvalidCredentials(err) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to refresh token"})
		return
	}

	c.JSON(http.StatusOK, AuthTokensResponse{
		AccessToken:  tokens.AccessToken,
		IDToken:      tokens.IDToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
	})
}

func (h *AuthHandler) SignOut(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing authorization header"})
		return
	}

	accessToken := strings.TrimPrefix(authHeader, "Bearer ")
	if accessToken == authHeader {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format"})
		return
	}

	if err := h.auth.SignOut(c.Request.Context(), accessToken); err != nil {
		if errors.IsInvalidCredentials(err) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sign out"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Signed out successfully"})
}
