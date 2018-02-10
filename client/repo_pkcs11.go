// +build pkcs11

package client

import (
	"fmt"

	"github.com/theupdateframework/notary"
	"github.com/theupdateframework/notary/trustmanager"
	"github.com/theupdateframework/notary/trustmanager/yubikey"
)

//GetKeyStores creates a new FileStore on the harddrive to store or load keys
func GetKeyStores(baseDir string, retriever notary.PassRetriever, hardwareBackup bool) ([]trustmanager.KeyStore, error) {
	fileKeyStore, err := trustmanager.NewKeyFileStore(baseDir, retriever)
	if err != nil {
		return nil, fmt.Errorf("failed to create private key store in directory: %s", baseDir)
	}

	keyStores := []trustmanager.KeyStore{fileKeyStore}

	var yubiKeyStore trustmanager.KeyStore

	if hardwareBackup {
		yubiKeyStore, err = yubikey.NewYubiStore(fileKeyStore, retriever)
	} else {
		yubiKeyStore, err = yubikey.NewYubiStore(nil, retriever)
	}

	if yubiKeyStore != nil {
		keyStores = []trustmanager.KeyStore{yubiKeyStore, fileKeyStore}
	}
	return keyStores, nil
}
