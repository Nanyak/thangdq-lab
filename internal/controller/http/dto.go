package http

type CreateShortURLRequest struct {
	LongURL string `json:"long_url" binding:"required"`
}

type CreateShortURLResponse struct {
	ShortURL string `json:"short_url"`
}

// Storage DTOs

type UploadFileResponse struct {
	Key string `json:"key"`
	URL string `json:"url"`
}

type FileInfo struct {
	Key          string `json:"key"`
	Size         int64  `json:"size"`
	ContentType  string `json:"content_type"`
	LastModified string `json:"last_modified"`
}

type ListFilesResponse struct {
	Files []FileInfo `json:"files"`
}

type PresignedURLResponse struct {
	URL       string `json:"url"`
	ExpiresAt string `json:"expires_at"`
}

// Auth DTOs

type SignUpRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name" binding:"required"`
}

type SignUpResponse struct {
	UserID    string `json:"user_id"`
	Confirmed bool   `json:"confirmed"`
}

type ConfirmSignUpRequest struct {
	Email            string `json:"email" binding:"required,email"`
	ConfirmationCode string `json:"confirmation_code" binding:"required"`
}

type SignInRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthTokensResponse struct {
	AccessToken  string `json:"access_token"`
	IDToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int32  `json:"expires_in"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ConfirmForgotPasswordRequest struct {
	Email       string `json:"email" binding:"required,email"`
	Code        string `json:"code" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

type UserResponse struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	EmailVerified bool   `json:"email_verified"`
}
