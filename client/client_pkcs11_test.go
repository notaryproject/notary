// +build pkcs11

package client

import (
	"github.com/theupdateframework/notary/trustmanager/pkcs11/common"
	"github.com/theupdateframework/notary/trustmanager/pkcs11/externalstore"
)

// clear out all keys
func init() {
	ks := externalstore.NewKeyStore()
	common.SetKeyStore(ks)
	if !common.IsAccessible() {
		return
	}
	store, err := common.NewHardwareStore(nil, nil)
	if err == nil {
		for k := range store.ListKeys() {
			store.RemoveKey(k)
		}
	}
}
