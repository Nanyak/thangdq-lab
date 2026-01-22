package errors

import "errors"

// Domain errors
var (
	ErrLinkNotFound       = errors.New("link not found")
	ErrDuplicateShortCode = errors.New("short code already exists")
	ErrInvalidURL         = errors.New("invalid URL format")
)

// IsNotFound checks if error is a not found error
func IsNotFound(err error) bool {
	return errors.Is(err, ErrLinkNotFound)
}

// IsDuplicate checks if error is a duplicate error
func IsDuplicate(err error) bool {
	return errors.Is(err, ErrDuplicateShortCode)
}
