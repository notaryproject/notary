package passphrase

import (
  "strings"
  "os"
  )

const (
	envPrefix       = "NOTARY_SIGNER_"
)

// NewDefaultPasswordStore instantiates a default password store
func NewDefaultPasswordStore() PasswordStore {
	return DefaultPasswordStore{}
}

// DefaultPasswordStore implements a basic password store which just stores and
// retrieves password from an environment variable.
type DefaultPasswordStore struct {
}

//The default password store stores the clear password in the ENV variable as is.
func (pw DefaultPasswordStore) SetPassword(alias string, newPassword string) error {
	envVariable := strings.ToUpper(envPrefix + alias)
	error := os.Setenv(envVariable, newPassword)

	return error
}

//The default password store retrieves the clear password from the ENV variable.
func (pw DefaultPasswordStore) GetPassword(alias string) (string, error) {
	envVariable := strings.ToUpper(envPrefix + alias)
  password := os.Getenv(envVariable)
	return password, nil
}
