package osxkeychain

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Security -framework Foundation

#include "osxkeychain_darwin.h"
#include <stdlib.h>
*/
import "C"
import (
	"errors"
	"net/url"
	"strconv"
	"strings"
	"unsafe"

	"github.com/docker/docker-credential-helpers/credentials"
)

// errCredentialsNotFound is the specific error message returned by OS X
// when the credentials are not in the keychain.
const errCredentialsNotFound = "The specified item could not be found in the keychain."

// Osxkeychain handles secrets using the OS X Keychain as store.
type Osxkeychain struct{}

// Add adds new credentials to the keychain.
func (h Osxkeychain) Add(creds *credentials.Credentials) error {
	s, err := splitServer(creds.ServerURL)
	if err != nil {
		return err
	}
	defer freeServer(s)

	username := C.CString(creds.Username)
	defer C.free(unsafe.Pointer(username))
	secret := C.CString(creds.Secret)
	defer C.free(unsafe.Pointer(secret))

	errMsg := C.keychain_add(s, username, secret)
	if errMsg != nil {
		defer C.free(unsafe.Pointer(errMsg))
		return errors.New(C.GoString(errMsg))
	}

	return nil
}

// Delete removes credentials from the keychain.
func (h Osxkeychain) Delete(serverURL string) error {
	s, err := splitServer(serverURL)
	if err != nil {
		return err
	}
	defer freeServer(s)

	errMsg := C.keychain_delete(s)
	if errMsg != nil {
		defer C.free(unsafe.Pointer(errMsg))
		return errors.New(C.GoString(errMsg))
	}

	return nil
}

// Get returns the username and secret to use for a given registry server URL.
func (h Osxkeychain) Get(serverURL string) (string, string, error) {
	s, err := splitServer(serverURL)
	if err != nil {
		return "", "", err
	}
	defer freeServer(s)

	var usernameLen C.uint
	var username *C.char
	var secretLen C.uint
	var secret *C.char
	defer C.free(unsafe.Pointer(username))
	defer C.free(unsafe.Pointer(secret))

	errMsg := C.keychain_get(s, &usernameLen, &username, &secretLen, &secret)
	if errMsg != nil {
		defer C.free(unsafe.Pointer(errMsg))
		goMsg := C.GoString(errMsg)

		if goMsg == errCredentialsNotFound {
			return "", "", credentials.NewErrCredentialsNotFound()
		}

		return "", "", errors.New(goMsg)
	}

	user := C.GoStringN(username, C.int(usernameLen))
	pass := C.GoStringN(secret, C.int(secretLen))
	return user, pass, nil
}

func splitServer(serverURL string) (*C.struct_Server, error) {
	u, err := url.Parse(serverURL)
	if err != nil {
		return nil, err
	}

	hostAndPort := strings.Split(u.Host, ":")
	host := hostAndPort[0]
	var port int
	if len(hostAndPort) == 2 {
		p, err := strconv.Atoi(hostAndPort[1])
		if err != nil {
			return nil, err
		}
		port = p
	}

	proto := C.kSecProtocolTypeHTTPS
	if u.Scheme != "https" {
		proto = C.kSecProtocolTypeHTTP
	}

	return &C.struct_Server{
		proto: C.SecProtocolType(proto),
		host:  C.CString(host),
		port:  C.uint(port),
		path:  C.CString(u.Path),
	}, nil
}

func freeServer(s *C.struct_Server) {
	C.free(unsafe.Pointer(s.host))
	C.free(unsafe.Pointer(s.path))
}
