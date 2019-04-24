// +build pkcs11

package client

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/theupdateframework/notary"
	"github.com/theupdateframework/notary/trustmanager"
	"github.com/theupdateframework/notary/trustmanager/grpckeystore"
  "github.com/theupdateframework/notary/trustmanager/yubikey"
)

func getKeyStores(baseDir string, retriever notary.PassRetriever,
	grpcKeyStoreConfig *grpckeystore.GRPCClientConfig) ([]trustmanager.KeyStore, error) {

	// Add the fileKeyStore as the default "always there" keystore
	fileKeyStore, err := trustmanager.NewKeyFileStore(baseDir, retriever)
	if err != nil {
		return nil, fmt.Errorf("failed to create private key store in directory: %s", baseDir)
	}
	keyStores := []trustmanager.KeyStore{fileKeyStore}

  // Add additional/optional keystores to list.  Keystores are prepended, so
	// last keystore has highest priority.

	// if there is a Yubi KeyStore in use, prepend it to the list
	yubiKeyStore, _ := yubikey.NewYubiStore(fileKeyStore, retriever)
	if yubiKeyStore != nil {
		keyStores = append([]trustmanager.KeyStore{yubiKeyStore}, keyStores...)
	}

	// if there is a GRPC Remote KeyStore configured, prepend it to the list
	if grpcKeyStoreConfig.Server != "" {
		grpcKeyStore, err := grpckeystore.NewGRPCKeyStore(grpcKeyStoreConfig)

 	  if err == nil {
			keyStores = append([]trustmanager.KeyStore{grpcKeyStore}, keyStores...)
			logrus.Debugf("GRPCKeyStore connection to %s succeeded", grpcKeyStoreConfig.Server)
		} else {
			logrus.Debugf("GRPCKeyStore connection attempt to %s failed:%s", grpcKeyStoreConfig.Server,err)
		}
	} else {
		logrus.Debug("GRPCKeyStore server not configured, disabled")
	}
	return keyStores, nil
}
