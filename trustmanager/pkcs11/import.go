// +build pkcs11

package pkcs11

import (
	"encoding/pem"
	"errors"

	"github.com/theupdateframework/notary"
	"github.com/theupdateframework/notary/trustmanager"
	"github.com/theupdateframework/notary/trustmanager/pkcs11/opencryptoki"
	"github.com/theupdateframework/notary/trustmanager/pkcs11/common"
	"github.com/theupdateframework/notary/trustmanager/pkcs11/yubikey"
	"github.com/theupdateframework/notary/tuf/data"
	"github.com/theupdateframework/notary/tuf/utils"
)

var hardwareKeyStore common.HardwareSpecificStore

// Setup defines which hardwarestore is available and prefers yubikey over opencryptoki.
// If none is found, opencryptoki is set, as one has to be set.
func Setup() {
	if hardwareKeyStore != nil {
		return
	}
	hardwareKeyStore = yubikey.NewKeyStore()
	ctx, session, err := hardwareKeyStore.SetupHSMEnv(common.DefaultLoader)
	if err == nil {
		common.SetKeyStore(hardwareKeyStore)
		defer common.Cleanup(ctx, session)
		return
	}

	hardwareKeyStore = opencryptoki.NewKeyStore()
	common.SetKeyStore(hardwareKeyStore)
	return
}

// HardwareImport is a wrapper around the HardwareStore that allows us to import private
// keys to the yubikey
type HardwareImport struct {
	dest          *common.HardwareStore
	passRetriever notary.PassRetriever
}

// NewImporter returns a wrapper for the HardwareStore provided that enables importing
// keys via the simple Set(string, []byte) interface
func NewImporter(hs *common.HardwareStore, ret notary.PassRetriever) *HardwareImport {
	return &HardwareImport{
		dest:          hs,
		passRetriever: ret,
	}
}

// Set determines if we are allowed to set the given key on the Yubikey and
// calls through to HardwareStore.AddKey if it's valid
func (s *HardwareImport) Set(name string, bytes []byte) error {
	block, _ := pem.Decode(bytes)
	if block == nil {
		return errors.New("invalid PEM data, could not parse")
	}
	role, ok := block.Headers["role"]
	if !ok {
		return errors.New("no role found for key")
	}
	ki := trustmanager.KeyInfo{
		// GUN is ignored by HardwareStore
		Role: data.RoleName(role),
	}
	privKey, err := utils.ParsePEMPrivateKey(bytes, "")
	if err != nil {
		privKey, _, err = trustmanager.GetPasswdDecryptBytes(
			s.passRetriever,
			bytes,
			name,
			ki.Role.String(),
		)
		if err != nil {
			return err
		}
	}
	return s.dest.AddKey(ki, privKey)
}
