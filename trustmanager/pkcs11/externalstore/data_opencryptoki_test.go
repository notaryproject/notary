// +build pkcs11, opencryptoki

package externalstore

import (
	"github.com/theupdateframework/notary/passphrase"
)

const userpin = "12345670"

// Overwrite Test Parameters
func init() {
	testNumSlots = 10
	ret = passphrase.ConstantRetriever(userpin)
}
