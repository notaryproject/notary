package passwordwrap

import (
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestDefaultStorageGetPassword(t *testing.T) {
	os.Setenv("NOTARY_SIGNER_TIMESTAMP", "password")
	defer os.Unsetenv("NOTARY_SIGNER_TIMESTAMP")

	passwordStore := NewDefaultPasswordStore()
	password, _ := passwordStore.GetPassword("timestamp")
	require.Equal(t, "password", password)
}

func TestDefaultStorageSetPassword(t *testing.T) {
	passwordStore := NewDefaultPasswordStore()
	passwordStore.SetPassword("timestamp", "password")
	defer os.Unsetenv("NOTARY_SIGNER_TIMESTAMP")

	require.Equal(t, "password", os.Getenv("NOTARY_SIGNER_TIMESTAMP"))
}

func TestDefaultProtectorEncrypt(t *testing.T) {
	passwordProtector := NewDefaultPasswordProtector()
	cipherText, _ := passwordProtector.Encrypt("password")

	require.Equal(t, "password", cipherText)
}

func TestDefaultProtectorDecrypt(t *testing.T) {
	passwordProtector := NewDefaultPasswordProtector()
	clearText, _ := passwordProtector.Decrypt("password")

	require.Equal(t, "password", clearText)
}
