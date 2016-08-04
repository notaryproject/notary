package keydbstore

import (
	"crypto"
	"io"

	"github.com/Sirupsen/logrus"
	"github.com/docker/notary/tuf/data"
)

type activatingPrivateKey struct {
	data.PrivateKey
	activationFunc func(keyID string) error
}

func (a activatingPrivateKey) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) (signature []byte, err error) {
	keyID := a.PrivateKey.ID()
	sig, err := a.PrivateKey.Sign(rand, digest, opts)
	if err == nil {
		if activationErr := a.activationFunc(keyID); activationErr != nil {
			logrus.Errorf("Key %s was just used to sign hash %s, error when trying to mark key as active: %s",
				keyID, digest, activationErr.Error())
		}
	}
	return sig, err
}
