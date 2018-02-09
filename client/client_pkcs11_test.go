// +build pkcs11

package client

import (
	"github.com/theupdateframework/notary/trustmanager/pkcs11/common"
	"github.com/theupdateframework/notary/trustmanager/pkcs11/yubikey"
)

// clear out all keys
func init() {
	yubikey.SetYubikeyKeyMode(0)
	ks := yubikey.NewKeyStore()
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
