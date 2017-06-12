package passwordwrap

import (
	"os"
	"strings"
)

const (
	envPrefix = "NOTARY_SIGNER_"
)

// NewDefaultPasswordStore instantiates a default password store
func NewDefaultPasswordStore() Storage {
	return DefaultPasswordStore{}
}

// DefaultPasswordStore implements a basic password store which just stores and
// retrieves password from an environment variable.
type DefaultPasswordStore struct {
}

// SetPassword stores the clear password in the ENV variable as is in clear text.
func (pw DefaultPasswordStore) SetPassword(alias string, newPassword string) error {
	envVariable := strings.ToUpper(envPrefix + alias)
	error := os.Setenv(envVariable, newPassword)
	return error
}

// GetPassword retrieves the clear password from the ENV variable.
func (pw DefaultPasswordStore) GetPassword(alias string) (string, error) {
	envVariable := strings.ToUpper(envPrefix + alias)
	pass := os.Getenv(envVariable)
	return pass, nil
}
