package p11store

import (
	"github.com/theupdateframework/notary/tuf/data"
	"errors"
	"github.com/theupdateframework/notary/trustmanager"
)

// Pkcs11CryptoService is a signed.CryptoService which uses a chosen
// token for PKCS#11 operations.
type Pkcs11CryptoService struct {
	// Underyling key store to use.
	Store *Pkcs11Store

	// Token to use for key generation.
	Token string
}

func (ps *Pkcs11CryptoService) Create(role data.RoleName, gun data.GUN, algorithm string) (pubKey data.PublicKey, err error) {
	_, pubKey, err = ps.Store.Generate(trustmanager.KeyInfo{gun, role}, ps.Token, algorithm)
	return
}

func (ps *Pkcs11CryptoService) AddKey(role data.RoleName, gun data.GUN, key data.PrivateKey) (err error) {
	err = errors.New("Pkcs11CryptoService.AddKey not implemented")
	return
}

func (ps *Pkcs11CryptoService) GetKey(keyID string) (pubKey data.PublicKey) {
	privKey, _, err := ps.Store.GetKey(keyID)
	if err != nil {
		return
	}
	pubKey = privKey
	return
}

func (ps *Pkcs11CryptoService) GetPrivateKey(keyID string) (privKey data.PrivateKey, role data.RoleName, err error) {
	privKey, role, err = ps.Store.GetKey(keyID)
	return
}

func (ps *Pkcs11CryptoService) RemoveKey(keyID string) error {
	return ps.Store.RemoveKey(keyID)
}

func (ps *Pkcs11CryptoService) ListKeys(role data.RoleName) (keyIDs []string) {
	for keyID, _ := range ps.Store.ListKeys() {
		keyIDs = append(keyIDs, keyID)
	}
	return
}

func (ps *Pkcs11CryptoService) ListAllKeys() (keyRoles map[string]data.RoleName) {
	keyRoles = map[string]data.RoleName{}
	for keyID, keyInfo := range ps.Store.ListKeys() {
		keyRoles[keyID] = keyInfo.Role
	}
	return
}
