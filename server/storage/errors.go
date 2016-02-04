package storage

import (
	"fmt"
)

// ErrOldVersion is returned when a newer version of TUF metadada is already available
type ErrOldVersion struct{}

// ErrOldVersion is returned when a newer version of TUF metadada is already available
func (err ErrOldVersion) Error() string {
	return fmt.Sprintf("Error updating metadata. A newer version is already available")
}

// ErrNotFound is returned when TUF metadata isn't found for a specific record
type ErrNotFound struct{}

// Error implements error
func (err ErrNotFound) Error() string {
	return fmt.Sprintf("No record found")
}

// ErrKeyExists is returned when a key already exists
type ErrKeyExists struct {
	Gun   string
	Role  string
	KeyID string
}

// ErrKeyExists is returned when a key already exists
func (err ErrKeyExists) Error() string {
	return fmt.Sprintf("Error, %s key %s already exists for %s", err.Role, err.KeyID, err.Gun)
}

// ErrNoKey is returned when no key is found
type ErrNoKey struct {
	Gun  string
	Role string
}

// ErrNoKey is returned when no timestamp key is found
func (err ErrNoKey) Error() string {
	return fmt.Sprintf("Error, no %s key found for %s", err.Role, err.Gun)
}
