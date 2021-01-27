// +build pkcs11

package client

import (
	"fmt"

	"github.com/theupdateframework/notary"
	"github.com/theupdateframework/notary/trustmanager"
	"github.com/theupdateframework/notary/trustmanager/pkcs11"
	"github.com/theupdateframework/notary/trustmanager/pkcs11/common"
)

func init() {
	pkcs11.Setup()
}

func getKeyStores(baseDir string, retriever notary.PassRetriever) ([]trustmanager.KeyStore, error) {
	fileKeyStore, err := trustmanager.NewKeyFileStore(baseDir, retriever)
	if err != nil {
		return nil, fmt.Errorf("failed to create private key store in directory: %s", baseDir)
	}

	keyStores := []trustmanager.KeyStore{fileKeyStore}
	hardwareKeyStore, _ := common.NewHardwareStore(fileKeyStore, retriever)
	if hardwareKeyStore != nil {
		keyStores = []trustmanager.KeyStore{hardwareKeyStore, fileKeyStore}
		return keyStores, nil
	}
	return keyStores, nil
}
