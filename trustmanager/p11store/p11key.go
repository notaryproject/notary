package p11store

import (
	"crypto"
	"crypto/sha256"
	"github.com/miekg/pkcs11"
	"github.com/theupdateframework/notary/tuf/data"
	"io"
)

// Pkcs11PrivateKey represents a handle to a private key in a PKCS#11
// token.
type Pkcs11PrivateKey struct {
	data.TUFKey

	// The owning PKCS#11 key store
	Store *Pkcs11Store

	// An RW session onto the token containing this key
	Session pkcs11.SessionHandle

	// The PKCS#11 object handle for the private key object
	Object pkcs11.ObjectHandle

	// The public key.
	PublicKey crypto.PublicKey
}

// data.PrivateKey methods

func (k *Pkcs11PrivateKey) Sign(rand io.Reader, msg []byte, opts crypto.SignerOpts) (signature []byte, err error) {
	k.Store.lock.Lock()
	defer k.Store.lock.Unlock()
	// Despite looking superficially similar to crypto.Signer.Sign,
	// data.PrivateKey.Sign takes a whole message.
	digest := sha256.Sum256(msg)
	mechanism := []*pkcs11.Mechanism{
		pkcs11.NewMechanism(pkcs11.CKM_ECDSA, nil),
	}
	if err = k.Store.ctx.SignInit(k.Session, mechanism, k.Object); err != nil {
		return
	}
	if signature, err = k.Store.ctx.Sign(k.Session, digest[:]); err != nil {
		return
	}
	// Despite looking superficially similar to crypto.Signer.Sign,
	// data.PrivateKey.Sign returns r || s (with lengths of both normalized).
	return
}

func (k *Pkcs11PrivateKey) Private() []byte {
	panic("cannot get private half of PKCS#11 key") // No better way to return an error
}

func (k *Pkcs11PrivateKey) CryptoSigner() crypto.Signer {
	return &Pkcs11Signer{k} // Shim to work around interface mismatch
}

func (k *Pkcs11PrivateKey) SignatureAlgorithm() data.SigAlgorithm {
	return data.ECDSASignature
}
