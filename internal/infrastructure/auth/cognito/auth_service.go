package cognito

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"

	"github.com/Nanyak/thangdq-lab/internal/entity"
	"github.com/Nanyak/thangdq-lab/pkg/config"
	pkgerrors "github.com/Nanyak/thangdq-lab/pkg/errors"
	"github.com/lestrrat-go/jwx/v2/jwk"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	cip "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
)

// CognitoService implements the AuthProvider interface using AWS Cognito
type CognitoService struct {
	client       *cip.Client
	userPoolID   string
	clientID     string
	clientSecret string
	tokenUrl     string
	jwtIssuerUrl string
	jwkSet       jwk.Set
}

// NewCognitoService creates a new CognitoService instance
func NewCognitoService(cfg *config.CognitoConfig) (*CognitoService, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(cfg.Region),
	)
	if err != nil {
		return nil, err
	}
	keySet, err := jwk.Fetch(context.Background(), cfg.TokenUrl)
	if err != nil {
		return nil, err
	}
	client := cip.NewFromConfig(awsCfg)

	return &CognitoService{
		client:       client,
		clientID:     cfg.ClientID,
		clientSecret: cfg.ClientSecret,
		tokenUrl:     cfg.TokenUrl,
		jwtIssuerUrl: cfg.JWTIssuerUrl,
		jwkSet:       keySet,
	}, nil
}

// SignUp registers a new user in Cognito
func (s *CognitoService) SignUp(ctx context.Context, username, email, password, name string) (*entity.AuthResult, error) {
	secretHash := s.computeSecretHash(username)

	input := &cip.SignUpInput{
		ClientId:   &s.clientID,
		Username:   &username,
		Password:   &password,
		SecretHash: &secretHash,
		UserAttributes: []types.AttributeType{
			{Name: strPtr("email"), Value: &email},
			{Name: strPtr("name"), Value: &name},
		},
	}

	output, err := s.client.SignUp(ctx, input)
	if err != nil {
		return nil, mapCognitoError(err)
	}

	return &entity.AuthResult{
		UserID:    *output.UserSub,
		Confirmed: output.UserConfirmed,
	}, nil
}

// ConfirmSignUp confirms a user's registration with a verification code
func (s *CognitoService) ConfirmSignUp(ctx context.Context, email, confirmationCode string) error {
	secretHash := s.computeSecretHash(email)

	input := &cip.ConfirmSignUpInput{
		ClientId:         &s.clientID,
		Username:         &email,
		ConfirmationCode: &confirmationCode,
		SecretHash:       &secretHash,
	}

	_, err := s.client.ConfirmSignUp(ctx, input)
	if err != nil {
		return mapCognitoError(err)
	}
	return nil
}

// SignIn authenticates a user and returns tokens
func (s *CognitoService) SignIn(ctx context.Context, email, password string) (*entity.AuthTokens, error) {
	secretHash := s.computeSecretHash(email)

	input := &cip.InitiateAuthInput{
		AuthFlow: types.AuthFlowTypeUserPasswordAuth,
		ClientId: &s.clientID,
		AuthParameters: map[string]string{
			"USERNAME":    email,
			"PASSWORD":    password,
			"SECRET_HASH": secretHash,
		},
	}

	output, err := s.client.InitiateAuth(ctx, input)
	if err != nil {
		return nil, mapCognitoError(err)
	}

	result := output.AuthenticationResult
	return &entity.AuthTokens{
		AccessToken:  *result.AccessToken,
		IDToken:      *result.IdToken,
		RefreshToken: *result.RefreshToken,
		ExpiresIn:    result.ExpiresIn,
	}, nil
}

// ForgotPassword initiates a password reset flow
func (s *CognitoService) ForgotPassword(ctx context.Context, email string) error {
	secretHash := s.computeSecretHash(email)

	input := &cip.ForgotPasswordInput{
		ClientId:   &s.clientID,
		Username:   &email,
		SecretHash: &secretHash,
	}

	_, err := s.client.ForgotPassword(ctx, input)
	if err != nil {
		return mapCognitoError(err)
	}
	return nil
}

// ConfirmForgotPassword completes the password reset with a confirmation code
func (s *CognitoService) ConfirmForgotPassword(ctx context.Context, email, code, newPassword string) error {
	secretHash := s.computeSecretHash(email)

	input := &cip.ConfirmForgotPasswordInput{
		ClientId:         &s.clientID,
		Username:         &email,
		ConfirmationCode: &code,
		Password:         &newPassword,
		SecretHash:       &secretHash,
	}

	_, err := s.client.ConfirmForgotPassword(ctx, input)
	if err != nil {
		return mapCognitoError(err)
	}
	return nil
}

func (s *CognitoService) GetUser(ctx context.Context, accessToken string) (*entity.User, error) {
	output, err := s.client.GetUser(ctx, &cip.GetUserInput{
		AccessToken: &accessToken,
	})
	if err != nil {
		return nil, mapCognitoError(err)
	}

	user := &entity.User{Username: *output.Username}
	for _, attr := range output.UserAttributes {
		switch *attr.Name {
		case "sub":
			user.ID = *attr.Value
		case "email":
			user.Email = *attr.Value
		case "email_verified":
			user.EmailVerified = *attr.Value == "true"
		case "name":
			user.Name = *attr.Value
		}
	}
	return user, nil
}

// computeSecretHash generates the SECRET_HASH for Cognito API calls
func (s *CognitoService) computeSecretHash(username string) string {
	mac := hmac.New(sha256.New, []byte(s.clientSecret))
	mac.Write([]byte(username + s.clientID))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// mapCognitoError maps Cognito SDK errors to domain errors
func mapCognitoError(err error) error {
	var usernameExists *types.UsernameExistsException
	var notAuthorized *types.NotAuthorizedException
	var userNotConfirmed *types.UserNotConfirmedException
	var codeMismatch *types.CodeMismatchException
	var userNotFound *types.UserNotFoundException

	switch {
	case asError(err, &usernameExists):
		return pkgerrors.ErrUserAlreadyExists
	case asError(err, &notAuthorized):
		return pkgerrors.ErrInvalidCredentials
	case asError(err, &userNotConfirmed):
		return pkgerrors.ErrUserNotConfirmed
	case asError(err, &codeMismatch):
		return pkgerrors.ErrCodeMismatch
	case asError(err, &userNotFound):
		return pkgerrors.ErrUserNotFound
	default:
		return err
	}
}

func asError[T error](err error, target *T) bool {
	return errors.As(err, target)
}

func strPtr(s string) *string {
	return &s
}
