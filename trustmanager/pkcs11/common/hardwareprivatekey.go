//+build pkcs11

package common

import (
	"crypto"
	"crypto/x509"
	"errors"
	"fmt"
	"io"

	"github.com/theupdateframework/notary"
	"github.com/theupdateframework/notary/tuf/data"
	"github.com/theupdateframework/notary/tuf/signed"
)

const (
	SigAttempts = 5
)

// HardwarePrivateKey represents a private key inside of a Hardwarestore
type HardwarePrivateKey struct {
	data.ECDSAPublicKey
	passRetriever notary.PassRetriever
	slot          HardwareSlot
	libLoader     Pkcs11LibLoader
}

// hardwareSigner wraps a HardwarePrivateKey and implements the crypto.Signer interface
type hardwareSigner struct {
	HardwarePrivateKey
}

// NewHwrdwarePrivateKey returns a HwardwarePrivateKey, which implements the data.PrivateKey
// interface except that the private material is inaccessible
func NewHardwarePrivateKey(slot HardwareSlot, pubKey data.ECDSAPublicKey, passRetriever notary.PassRetriever) *HardwarePrivateKey {
	return &HardwarePrivateKey{
		ECDSAPublicKey: pubKey,
		passRetriever:  passRetriever,
		slot:           slot,
		libLoader:      DefaultLoader,
	}
}

// Public is a required method of the crypto.Signer interface
func (ys *hardwareSigner) Public() crypto.PublicKey {
	publicKey, err := x509.ParsePKIXPublicKey(ys.HardwarePrivateKey.Public())
	if err != nil {
		return nil
	}

	return publicKey
}

func (hpk *HardwarePrivateKey) SetLibLoader(loader Pkcs11LibLoader) {
	hpk.libLoader = loader
}

// CryptoSigner returns a crypto.Signer tha wraps the HardwarePrivateKey. Needed for
// Certificate generation only
func (hpk *HardwarePrivateKey) CryptoSigner() crypto.Signer {
	return &hardwareSigner{HardwarePrivateKey: *hpk}
}

// Private is not implemented in hardware  keys
func (hpk *HardwarePrivateKey) Private() []byte {
	return nil
}

// SignatureAlgorithm returns which algorithm this key uses to sign - currently
// hardcoded to ECDSA
func (y HardwarePrivateKey) SignatureAlgorithm() data.SigAlgorithm {
	return data.ECDSASignature
}

// Sign is a required method of the crypto.Signer interface and the data.PrivateKey
// interface
func (hpk *HardwarePrivateKey) Sign(rand io.Reader, msg []byte, opts crypto.SignerOpts) ([]byte, error) {
	ctx, session, err := hardwareKeyStore.SetupHSMEnv(hpk.libLoader)
	if err != nil {
		return nil, err
	}
	defer Cleanup(ctx, session)

	v := signed.Verifiers[data.ECDSASignature]
	for i := 0; i < SigAttempts; i++ {
		sig, err := hardwareKeyStore.Sign(ctx, session, hpk.slot, hpk.passRetriever, msg)
		if err != nil {
			return nil, fmt.Errorf("failed to sign using %s: %v", hardwareName, err)
		}
		if err := v.Verify(&hpk.ECDSAPublicKey, sig, msg); err == nil {
			return sig, nil
		}
	}
	return nil, errors.New(fmt.Sprintln("failed to generate signature on %s", hardwareName))
}
