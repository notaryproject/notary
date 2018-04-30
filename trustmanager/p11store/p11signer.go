package p11store

import (
	"crypto"
	"encoding/asn1"
	"github.com/miekg/pkcs11"
	"io"
	"math/big"
)

// crypto.Signer support

// dsaSignature represents a DSA or ECDSA signature
type dsaSignature struct {
	R, S *big.Int
}

// Pkcs11Signer is a reference to a Pksc11PrivateKey that implements crypto.Signer.
type Pkcs11Signer struct {
	Key *Pkcs11PrivateKey
}

func (s *Pkcs11Signer) Public() crypto.PublicKey {
	return s.Key.PublicKey
}

func (s *Pkcs11Signer) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) (signature []byte, err error) {
	k := s.Key
	k.Store.lock.Lock()
	defer k.Store.lock.Unlock()
	mechanism := []*pkcs11.Mechanism{
		pkcs11.NewMechanism(pkcs11.CKM_ECDSA, nil),
	}
	if err = k.Store.ctx.SignInit(k.Session, mechanism, k.Object); err != nil {
		return
	}
	var rs []byte
	if rs, err = k.Store.ctx.Sign(k.Session, digest[:]); err != nil {
		return
	}
	// PKCS#11 returns r || s but crypto.Signer.Sign returns an ASN.1 representation.
	n := len(rs) / 2
	var sig dsaSignature
	sig.R, sig.S = new(big.Int), new(big.Int)
	sig.R.SetBytes(rs[:n])
	sig.S.SetBytes(rs[n:])
	if signature, err = asn1.Marshal(sig); err != nil {
		return
	}
	return
}
