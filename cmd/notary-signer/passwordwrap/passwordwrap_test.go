package passwordwrap

import (
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestDefaultStorageGet(t *testing.T) {
	os.Setenv("NOTARY_SIGNER_TIMESTAMP", "password")
	defer os.Unsetenv("NOTARY_SIGNER_TIMESTAMP")

	passwordStore := NewPassStore()
	password, _ := passwordStore.Get("timestamp")
	require.Equal(t, "password", password)
}

func TestDefaultStorageGetPasswordFromCache(t *testing.T) {
	os.Setenv("NOTARY_SIGNER_TIMESTAMP", "password")
	defer os.Unsetenv("NOTARY_SIGNER_TIMESTAMP")

	passwordStore := NewPassStore()

	//First call to populate the cache and second call to fetch from it
	passwordStore.Get("timestamp")
	password, _ := passwordStore.Get("timestamp")
	require.Equal(t, "password", password)
}

func TestDefaultStorageSetPassword(t *testing.T) {
	passwordStore := NewPassStore()
	passwordStore.Set("timestamp", "password")
	defer os.Unsetenv("NOTARY_SIGNER_TIMESTAMP")

	require.Equal(t, "password", os.Getenv("NOTARY_SIGNER_TIMESTAMP"))
}
