// +build pkcs11, yubikey

package externalstore

import (
	"strings"
)

const (
	userpin = "123456"
	sopin   = "010203040506070801020304050607080102030405060708"
)

// Overwrite Test Parameters
func init() {
	testNumSlots = 4

	ret = func(k, a string, c bool, n int) (string, bool, error) {
		if strings.Contains(k, "SO") {
			return sopin, false, nil
		} else if strings.Contains(k, "User") {
			return userpin, false, nil
		} else {
			return "passphrase", true, nil
		}
	}
}
