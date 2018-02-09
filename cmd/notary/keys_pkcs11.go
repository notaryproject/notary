// +build pkcs11

package main

import (
	"github.com/theupdateframework/notary"
	store "github.com/theupdateframework/notary/storage"
	"github.com/theupdateframework/notary/trustmanager"
	"github.com/theupdateframework/notary/trustmanager/pkcs11"
	"github.com/theupdateframework/notary/trustmanager/pkcs11/common"
)

func init() {
	pkcs11.Setup()
}

func getHardwareStore(fileKeyStore trustmanager.KeyStore, ret notary.PassRetriever) (*common.HardwareStore, error) {
	return common.NewHardwareStore(fileKeyStore, ret)
}

func getImporters(baseDir string, ret notary.PassRetriever) ([]trustmanager.Importer, error) {
	var importers []trustmanager.Importer
	if common.IsAccessible() {
		yubiStore, err := getHardwareStore(nil, ret)
		if err == nil {
			importers = append(
				importers,
				pkcs11.NewImporter(yubiStore, ret),
			)
		}
	}
	fileStore, err := store.NewPrivateKeyFileStorage(baseDir, notary.KeyExtension)
	if err == nil {
		importers = append(
			importers,
			fileStore,
		)
	} else if len(importers) == 0 {
		return nil, err // couldn't initialize any stores
	}
	return importers, nil
}
