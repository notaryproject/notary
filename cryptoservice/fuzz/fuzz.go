// +build gofuzz

package fuzz

import (
	"github.com/theupdateframework/notary/cryptoservice"
	"github.com/theupdateframework/notary/passphrase"
	"github.com/theupdateframework/notary/trustmanager"
)

// Fuzz implements the fuzzer that targets GetPrivateKey
func Fuzz(data []byte) int {
	cryptos := cryptoservice.NewCryptoService(trustmanager.NewKeyMemoryStore(passphrase.ConstantRetriever("pass")))
	_, _, err := cryptos.GetPrivateKey(string(data))
	if err != nil {
		return 0
	}
	return 1
}
