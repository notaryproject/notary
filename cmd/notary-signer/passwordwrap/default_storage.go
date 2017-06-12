package passwordwrap

import (
	"os"
	"strings"
	"sync"
)

const (
	envPrefix = "NOTARY_SIGNER_"
)

// NewDefaultPasswordStore instantiates a default password store
func NewDefaultPasswordStore() Storage {
	return DefaultPasswordStore{
		lock:        &sync.RWMutex{},
		passwdCache: make(map[string]string),
	}
}

// DefaultPasswordStore implements a basic password store which just stores and
// retrieves password from an environment variable.
type DefaultPasswordStore struct {
	lock        *sync.RWMutex
	passwdCache map[string]string
}

// SetPassword stores the clear password in the ENV variable as is in clear text and also caches it.
func (pw DefaultPasswordStore) SetPassword(alias string, newPassword string) error {
	envVariable := strings.ToUpper(envPrefix + alias)
	error := os.Setenv(envVariable, newPassword)

	// If we could successfully set it to the environment, then add/update the new password into the cache.
	if error == nil {
		pw.lock.Lock()
		defer pw.lock.Unlock()
		pw.passwdCache[alias] = newPassword
	}
	return error
}

// GetPassword retrieves the clear password from the cache and if not present, from the ENV variable.
func (pw DefaultPasswordStore) GetPassword(alias string) (string, error) {
	//If the password is available in the cache return it
	pw.lock.RLock()
	passwd, ok := pw.passwdCache[alias]
	pw.lock.RUnlock()
	if ok {
		return passwd, nil
	}

	//If not in the cache, get the password from the environment variable
	envVariable := strings.ToUpper(envPrefix + alias)
	passwd = os.Getenv(envVariable)

	//Cache it and return out the password
	pw.lock.Lock()
	defer pw.lock.Unlock()
	pw.passwdCache[alias] = passwd
	return passwd, nil
}
