package errors

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestIsNotFound tests the IsNotFound helper function
func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "matching error - ErrLinkNotFound",
			err:      ErrLinkNotFound,
			expected: true,
		},
		{
			name:     "wrapped matching error",
			err:      fmt.Errorf("failed to get link: %w", ErrLinkNotFound),
			expected: true,
		},
		{
			name:     "non-matching error - ErrDuplicateShortCode",
			err:      ErrDuplicateShortCode,
			expected: false,
		},
		{
			name:     "non-matching error - ErrInvalidURL",
			err:      ErrInvalidURL,
			expected: false,
		},
		{
			name:     "non-matching error - ErrFileNotFound",
			err:      ErrFileNotFound,
			expected: false,
		},
		{
			name:     "non-matching error - generic error",
			err:      errors.New("some other error"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "deeply wrapped matching error",
			err:      fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", ErrLinkNotFound)),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNotFound(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsDuplicate tests the IsDuplicate helper function
func TestIsDuplicate(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "matching error - ErrDuplicateShortCode",
			err:      ErrDuplicateShortCode,
			expected: true,
		},
		{
			name:     "wrapped matching error",
			err:      fmt.Errorf("failed to save link: %w", ErrDuplicateShortCode),
			expected: true,
		},
		{
			name:     "non-matching error - ErrLinkNotFound",
			err:      ErrLinkNotFound,
			expected: false,
		},
		{
			name:     "non-matching error - ErrInvalidURL",
			err:      ErrInvalidURL,
			expected: false,
		},
		{
			name:     "non-matching error - ErrUploadFailed",
			err:      ErrUploadFailed,
			expected: false,
		},
		{
			name:     "non-matching error - generic error",
			err:      errors.New("duplicate key error"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "deeply wrapped matching error",
			err:      fmt.Errorf("validation: %w", fmt.Errorf("conflict: %w", ErrDuplicateShortCode)),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsDuplicate(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsFileNotFound tests the IsFileNotFound helper function
func TestIsFileNotFound(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "matching error - ErrFileNotFound",
			err:      ErrFileNotFound,
			expected: true,
		},
		{
			name:     "wrapped matching error",
			err:      fmt.Errorf("failed to retrieve file: %w", ErrFileNotFound),
			expected: true,
		},
		{
			name:     "non-matching error - ErrLinkNotFound",
			err:      ErrLinkNotFound,
			expected: false,
		},
		{
			name:     "non-matching error - ErrUploadFailed",
			err:      ErrUploadFailed,
			expected: false,
		},
		{
			name:     "non-matching error - ErrDownloadFailed",
			err:      ErrDownloadFailed,
			expected: false,
		},
		{
			name:     "non-matching error - generic error",
			err:      errors.New("file not found"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "deeply wrapped matching error",
			err:      fmt.Errorf("storage: %w", fmt.Errorf("s3: %w", ErrFileNotFound)),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsFileNotFound(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsUploadFailed tests the IsUploadFailed helper function
func TestIsUploadFailed(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "matching error - ErrUploadFailed",
			err:      ErrUploadFailed,
			expected: true,
		},
		{
			name:     "wrapped matching error",
			err:      fmt.Errorf("file upload error: %w", ErrUploadFailed),
			expected: true,
		},
		{
			name:     "non-matching error - ErrDownloadFailed",
			err:      ErrDownloadFailed,
			expected: false,
		},
		{
			name:     "non-matching error - ErrFileNotFound",
			err:      ErrFileNotFound,
			expected: false,
		},
		{
			name:     "non-matching error - ErrDeleteFailed",
			err:      ErrDeleteFailed,
			expected: false,
		},
		{
			name:     "non-matching error - generic error",
			err:      errors.New("upload failed"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "deeply wrapped matching error",
			err:      fmt.Errorf("s3: %w", fmt.Errorf("network: %w", ErrUploadFailed)),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsUploadFailed(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsDownloadFailed tests the IsDownloadFailed helper function
func TestIsDownloadFailed(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "matching error - ErrDownloadFailed",
			err:      ErrDownloadFailed,
			expected: true,
		},
		{
			name:     "wrapped matching error",
			err:      fmt.Errorf("file download error: %w", ErrDownloadFailed),
			expected: true,
		},
		{
			name:     "non-matching error - ErrUploadFailed",
			err:      ErrUploadFailed,
			expected: false,
		},
		{
			name:     "non-matching error - ErrFileNotFound",
			err:      ErrFileNotFound,
			expected: false,
		},
		{
			name:     "non-matching error - ErrDeleteFailed",
			err:      ErrDeleteFailed,
			expected: false,
		},
		{
			name:     "non-matching error - generic error",
			err:      errors.New("download failed"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "deeply wrapped matching error",
			err:      fmt.Errorf("s3: %w", fmt.Errorf("timeout: %w", ErrDownloadFailed)),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsDownloadFailed(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsDeleteFailed tests the IsDeleteFailed helper function
func TestIsDeleteFailed(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "matching error - ErrDeleteFailed",
			err:      ErrDeleteFailed,
			expected: true,
		},
		{
			name:     "wrapped matching error",
			err:      fmt.Errorf("file deletion error: %w", ErrDeleteFailed),
			expected: true,
		},
		{
			name:     "non-matching error - ErrUploadFailed",
			err:      ErrUploadFailed,
			expected: false,
		},
		{
			name:     "non-matching error - ErrDownloadFailed",
			err:      ErrDownloadFailed,
			expected: false,
		},
		{
			name:     "non-matching error - ErrFileNotFound",
			err:      ErrFileNotFound,
			expected: false,
		},
		{
			name:     "non-matching error - generic error",
			err:      errors.New("delete failed"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "deeply wrapped matching error",
			err:      fmt.Errorf("s3: %w", fmt.Errorf("permissions: %w", ErrDeleteFailed)),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsDeleteFailed(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsInvalidURL tests implicit behavior with ErrInvalidURL (no Is helper defined)
func TestIsInvalidURL(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "can use standard errors.Is with ErrInvalidURL",
			err:      ErrInvalidURL,
			expected: true,
		},
		{
			name:     "wrapped ErrInvalidURL",
			err:      fmt.Errorf("validation failed: %w", ErrInvalidURL),
			expected: true,
		},
		{
			name:     "non-matching error",
			err:      errors.New("invalid url"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Using standard errors.Is since there's no IsInvalidURL helper
			result := errors.Is(tt.err, ErrInvalidURL)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestMultipleErrorMatching tests that helpers correctly differentiate between multiple wrapped errors
func TestMultipleErrorMatching(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		isNotFound    bool
		isDuplicate   bool
		isFileNotFound bool
		isUploadFailed bool
		isDownloadFailed bool
		isDeleteFailed bool
	}{
		{
			name:             "only ErrLinkNotFound matches",
			err:              fmt.Errorf("operation failed: %w", ErrLinkNotFound),
			isNotFound:       true,
			isDuplicate:      false,
			isFileNotFound:   false,
			isUploadFailed:   false,
			isDownloadFailed: false,
			isDeleteFailed:   false,
		},
		{
			name:             "only ErrDuplicateShortCode matches",
			err:              fmt.Errorf("constraint violation: %w", ErrDuplicateShortCode),
			isNotFound:       false,
			isDuplicate:      true,
			isFileNotFound:   false,
			isUploadFailed:   false,
			isDownloadFailed: false,
			isDeleteFailed:   false,
		},
		{
			name:             "only ErrFileNotFound matches",
			err:              fmt.Errorf("s3 error: %w", ErrFileNotFound),
			isNotFound:       false,
			isDuplicate:      false,
			isFileNotFound:   true,
			isUploadFailed:   false,
			isDownloadFailed: false,
			isDeleteFailed:   false,
		},
		{
			name:             "no errors match",
			err:              errors.New("unknown error"),
			isNotFound:       false,
			isDuplicate:      false,
			isFileNotFound:   false,
			isUploadFailed:   false,
			isDownloadFailed: false,
			isDeleteFailed:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.isNotFound, IsNotFound(tt.err))
			assert.Equal(t, tt.isDuplicate, IsDuplicate(tt.err))
			assert.Equal(t, tt.isFileNotFound, IsFileNotFound(tt.err))
			assert.Equal(t, tt.isUploadFailed, IsUploadFailed(tt.err))
			assert.Equal(t, tt.isDownloadFailed, IsDownloadFailed(tt.err))
			assert.Equal(t, tt.isDeleteFailed, IsDeleteFailed(tt.err))
		})
	}
}

// TestErrorVariables tests that error variables are properly defined
func TestErrorVariables(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{
			name: "ErrLinkNotFound is defined",
			err:  ErrLinkNotFound,
		},
		{
			name: "ErrDuplicateShortCode is defined",
			err:  ErrDuplicateShortCode,
		},
		{
			name: "ErrInvalidURL is defined",
			err:  ErrInvalidURL,
		},
		{
			name: "ErrFileNotFound is defined",
			err:  ErrFileNotFound,
		},
		{
			name: "ErrUploadFailed is defined",
			err:  ErrUploadFailed,
		},
		{
			name: "ErrDownloadFailed is defined",
			err:  ErrDownloadFailed,
		},
		{
			name: "ErrDeleteFailed is defined",
			err:  ErrDeleteFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.err)
			assert.NotEmpty(t, tt.err.Error())
		})
	}
}
