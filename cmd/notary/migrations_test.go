package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/notary"
)

func TestCleanTrustedCertificates(t *testing.T) {
	require.NoError(t, cleanTrustedCertificates(""))

	tmpDir, err := ioutil.TempDir("", "notary-migrations-")
	require.NoError(t, err)

	// no error when the dir doesn't exist per os.RemoveAll behaviour
	require.NoError(t, cleanTrustedCertificates(tmpDir))

	cd := filepath.Join(tmpDir, notary.TrustedCertsDir)
	os.Mkdir(
		filepath.Join(tmpDir, notary.TrustedCertsDir),
		notary.PrivExecPerms,
	)
	cf := filepath.Join(cd, "cert.test")
	require.NoError(t, ioutil.WriteFile(
		cf,
		[]byte{'1'},
		notary.PrivNoExecPerms,
	))

	require.NoError(t, os.Chmod(cd, 0000))
	require.Error(t, cleanTrustedCertificates(tmpDir))
	require.NoError(t, os.Chmod(cd, notary.PrivExecPerms))

	require.NoError(t, cleanTrustedCertificates(tmpDir))
	_, err = os.Stat(cf)
	require.True(t, os.IsNotExist(err))
}
