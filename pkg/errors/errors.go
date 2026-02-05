package errors

import "errors"

// Domain errors
var (
	ErrLinkNotFound       = errors.New("link not found")
	ErrDuplicateShortCode = errors.New("short code already exists")
	ErrInvalidURL         = errors.New("invalid URL format")
)

// Storage errors
var (
	ErrFileNotFound    = errors.New("file not found")
	ErrUploadFailed    = errors.New("upload failed")
	ErrDownloadFailed  = errors.New("download failed")
	ErrDeleteFailed    = errors.New("delete failed")
)

// IsNotFound checks if error is a not found error
func IsNotFound(err error) bool {
	return errors.Is(err, ErrLinkNotFound)
}

// IsDuplicate checks if error is a duplicate error
func IsDuplicate(err error) bool {
	return errors.Is(err, ErrDuplicateShortCode)
}

// IsFileNotFound checks if error is a file not found error
func IsFileNotFound(err error) bool {
	return errors.Is(err, ErrFileNotFound)
}

// IsUploadFailed checks if error is an upload failed error
func IsUploadFailed(err error) bool {
	return errors.Is(err, ErrUploadFailed)
}

// IsDownloadFailed checks if error is a download failed error
func IsDownloadFailed(err error) bool {
	return errors.Is(err, ErrDownloadFailed)
}

// IsDeleteFailed checks if error is a delete failed error
func IsDeleteFailed(err error) bool {
	return errors.Is(err, ErrDeleteFailed)
}

// Auth errors
var (
	ErrInvalidCredentials    = errors.New("invalid credentials")
	ErrUserAlreadyExists     = errors.New("user already exists")
	ErrUserNotConfirmed      = errors.New("user is not confirmed")
	ErrCodeMismatch          = errors.New("confirmation code mismatch")
	ErrUserNotFound          = errors.New("user not found")
	ErrAuthChallengeRequired = errors.New("authentication challenge required")
)

// IsInvalidCredentials checks if error is an invalid credentials error
func IsInvalidCredentials(err error) bool {
	return errors.Is(err, ErrInvalidCredentials)
}

// IsUserAlreadyExists checks if error is a user already exists error
func IsUserAlreadyExists(err error) bool {
	return errors.Is(err, ErrUserAlreadyExists)
}

// IsUserNotConfirmed checks if error is a user not confirmed error
func IsUserNotConfirmed(err error) bool {
	return errors.Is(err, ErrUserNotConfirmed)
}

// IsCodeMismatch checks if error is a code mismatch error
func IsCodeMismatch(err error) bool {
	return errors.Is(err, ErrCodeMismatch)
}

// IsUserNotFound checks if error is a user not found error
func IsUserNotFound(err error) bool {
	return errors.Is(err, ErrUserNotFound)
}
