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
