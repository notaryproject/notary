// +build pkcs11

package client

import (
	"fmt"

	"github.com/docker/notary"
	"github.com/docker/notary/passphrase"
	"github.com/docker/notary/trustmanager"
	"github.com/docker/notary/trustmanager/yubikey"
)

func getKeyStores(baseDir string, retriever notary.PassRetriever, useNative bool) ([]trustmanager.KeyStore, error) {
	fileKeyStore, err := trustmanager.NewKeyFileStore(baseDir, retriever)
	if err != nil {
		return nil, fmt.Errorf("failed to create private key store in directory: %s", baseDir)
	}

	keyStores := []trustmanager.KeyStore{fileKeyStore}
	if useNative {
		nativeKeyStore, err := trustmanager.NewKeyNativeStore(passphrase.ConstantRetriever("password"))
		if err == nil {
			// Note that the order is important, since we want to prioritize
			// the native key store
			keyStores = append([]trustmanager.KeyStore{nativeKeyStore}, keyStores...)
		}
	}
	yubiKeyStore, _ := yubikey.NewYubiStore(fileKeyStore, retriever)
	if yubiKeyStore != nil {
		// Note that the order is important, since we want to prioritize
		// the yubi key store
		keyStores = append([]trustmanager.KeyStore{yubiKeyStore}, keyStores...)
	}
	return keyStores, nil
}
