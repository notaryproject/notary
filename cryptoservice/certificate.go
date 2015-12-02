package cryptoservice

import (
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"fmt"

	"github.com/docker/notary/trustmanager"
	"github.com/docker/notary/tuf/data"
)

// GenerateCertificate generates an X509 Certificate from a template, given a GUN
func GenerateCertificate(rootKey data.PrivateKey, gun string) (*x509.Certificate, error) {
	signer := rootKey.CryptoSigner()
	if signer == nil {
		return nil, fmt.Errorf("key type not supported for Certificate generation: %s\n", rootKey.Algorithm())
	}

	return generateCertificate(signer, gun)
}

// GenerateTestingCertificate generates an X509 Certificate from a template, given a GUN.
func GenerateTestingCertificate(signer crypto.Signer, gun string) (*x509.Certificate, error) {
	return generateCertificate(signer, gun)
}

func generateCertificate(signer crypto.Signer, gun string) (*x509.Certificate, error) {
	template, err := trustmanager.NewCertificate(gun)
	if err != nil {
		return nil, fmt.Errorf("failed to create the certificate template for: %s (%v)", gun, err)
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, signer.Public(), signer)
	if err != nil {
		return nil, fmt.Errorf("failed to create the certificate for: %s (%v)", gun, err)
	}

	cert, err := x509.ParseCertificate(derBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the certificate for key: %s (%v)", gun, err)
	}

	return cert, nil
}
