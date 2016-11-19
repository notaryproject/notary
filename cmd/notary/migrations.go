package main

import (
	"os"
	"path/filepath"

	"github.com/Sirupsen/logrus"
	"github.com/docker/notary"
)

// cleanTrustedCertificates removes the obsoleted trusted_certificates
// We're only returning the error for ease of testing. Everyone else should
// ignore the error return.
func cleanTrustedCertificates(trustDir string) error {
	if trustDir == "" {
		return nil
	}
	loc := filepath.Join(trustDir, notary.TrustedCertsDir)
	if err := os.RemoveAll(loc); err != nil {
		logrus.Errorf(
			"failed to cleanup obsolete trusted_certificates directory: \"%s\"; you may run `rm -r %s` to manually remove this directory",
			err.Error(),
			trustDir,
		)
		return err
	}
	return nil
}
