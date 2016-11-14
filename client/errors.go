package client

import (
	"fmt"
)

// ErrRepoNotInitialized is returned when trying to publish an uninitialized
// notary repository
type ErrRepoNotInitialized struct{}

func (err ErrRepoNotInitialized) Error() string {
	return "repository has not been initialized"
}

// ErrInvalidRemoteRole is returned when the server is requested to manage
// a key type that is not permitted
type ErrInvalidRemoteRole struct {
	Role string
}

func (err ErrInvalidRemoteRole) Error() string {
	return fmt.Sprintf(
		"notary does not permit the server managing the %s key", err.Role)
}

// ErrInvalidLocalRole is returned when the client wants to manage
// a key type that is not permitted
type ErrInvalidLocalRole struct {
	Role string
}

func (err ErrInvalidLocalRole) Error() string {
	return fmt.Sprintf(
		"notary does not permit the client managing the %s key", err.Role)
}

// ErrRepositoryNotExist is returned when an action is taken on a remote
// repository that doesn't exist
type ErrRepositoryNotExist struct {
	remote string
	gun    string
}

func (err ErrRepositoryNotExist) Error() string {
	return fmt.Sprintf("%s does not have trust data for %s", err.remote, err.gun)
}
