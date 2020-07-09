package vault

import "fmt"

// WrongVersionError represents an error raised when the KV version is not correct
type WrongVersionError struct {
	Message string
}

// Error
func (e *WrongVersionError) Error() string {
	return e.Message
}

// PathNotFound represents an error when a path is not found in vault
type PathNotFound struct {
	Path string
}

// Error
func (e *PathNotFound) Error() string {
	return fmt.Sprintf("Path %s not found", e.Path)
}
