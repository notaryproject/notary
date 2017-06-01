package passphrase

import (
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestDefaultStorageGetPassword(t *testing.T) {
	os.Setenv("NOTARY_SIGNER_TIMESTAMP", "password")
	defer os.Unsetenv("NOTARY_SIGNER_TIMESTAMP")

	passphraseStore := NewDefaultPassphraseStore()
	passphrase, _ := passphraseStore.GetPassphrase("timestamp")
	require.Equal(t, "password", passphrase)
}

func TestDefaultStorageSetPassword(t *testing.T) {
	passphraseStore := NewDefaultPassphraseStore()
	passphraseStore.SetPassphrase("timestamp", "password")
	defer os.Unsetenv("NOTARY_SIGNER_TIMESTAMP")

	require.Equal(t, "password", os.Getenv("NOTARY_SIGNER_TIMESTAMP"))
}

func TestDefaultProtectorEncrypt(t *testing.T) {
	passphraseProtector := NewDefaultPassphraseProtector()
	cipherText, _ := passphraseProtector.Encrypt("password")

	require.Equal(t, "password", cipherText)
}

func TestDefaultProtectorDecrypt(t *testing.T) {
	passphraseProtector := NewDefaultPassphraseProtector()
	clearText, _ := passphraseProtector.Decrypt("password")

	require.Equal(t, "password", clearText)
}
