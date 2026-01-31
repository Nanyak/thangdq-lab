package entity

// User represents a user in the system
type User struct {
	ID            string
	Name          string
	Username      string
	Email         string
	EmailVerified bool
}

// AuthResult represents the result of a sign-up operation
type AuthResult struct {
	UserID    string
	Confirmed bool
}

// AuthTokens represents authentication tokens returned after sign-in
type AuthTokens struct {
	AccessToken  string
	IDToken      string
	RefreshToken string
	ExpiresIn    int32
}
