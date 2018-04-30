// +build !pkcs11

package client

import (
	"fmt"

	"github.com/theupdateframework/notary"
	"github.com/theupdateframework/notary/trustmanager"
	"github.com/theupdateframework/notary/trustmanager/p11store"
)

func getKeyStores(baseDir string, retriever notary.PassRetriever) ([]trustmanager.KeyStore, error) {
	fileKeyStore, err := trustmanager.NewKeyFileStore(baseDir, retriever)
	if err != nil {
		return nil, fmt.Errorf("failed to create private key store in directory: %s", baseDir)
	}
	keyStores := []trustmanager.KeyStore{fileKeyStore}
	var pkcs11 *p11store.Pkcs11Store
	if pkcs11, err = p11store.NewPkcs11Store("", retriever); err == nil {
		keyStores = append(keyStores, pkcs11)
	} else if err != p11store.ErrNoProvider {
		// A PKCS#11 provider was configured but something went wrong setting it up
		return nil, err
	} // else nothing was configured
	return keyStores, nil
}
