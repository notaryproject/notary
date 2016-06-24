package trustmanager

import (
	"crypto/rand"
	"testing"

	"github.com/docker/notary/tuf/utils"
	"github.com/stretchr/testify/require"
)

//will only test for your OS- could need to be more generic
//Testing that add is working while simultaneously testing that getkey and getkeyinfo are working for a keychain we know exists
func TestAddGetKeyInNativeStore(t *testing.T) {
	myKeyNativeStore, err := NewKeyNativeStore(defaultCredentialsStore)
	privKey, err := utils.GenerateECDSAKey(rand.Reader)
	genStore := []KeyStore{myKeyNativeStore}[0]
	myKeyInfo := KeyInfo{
		Gun:  "http://example.com/collections",
		Role: "Snapshot",
	}
	err = genStore.AddKey(myKeyInfo, privKey)
	require.NoError(t, err)
	//defer genStore.RemoveKey(privKey.ID())
	gotCreds, role, err := genStore.GetKey(privKey.ID())
	require.NoError(t, err)
	require.Equal(t, myKeyInfo.Role, role, "unexpected role")
	require.Equal(t, privKey.Private(), gotCreds.Private(), "unexpected content in the file")
	newKeyInfo, err := genStore.GetKeyInfo(privKey.ID())
	require.NoError(t, err)
	require.Equal(t, myKeyInfo.Role, newKeyInfo.Role, "Key info is incorrect")
	require.Equal(t, myKeyInfo.Gun, newKeyInfo.Gun, "Key info is incorrect")
}

//Testing that remove is working for a key that we know exists, also tests that getkey and getkeyinfo behave gracefully if a key doesn't exist
func TestRemoveWorksInNativeStore(t *testing.T) {
	myKeyNativeStore, err := NewKeyNativeStore(defaultCredentialsStore)
	privKey, err := utils.GenerateECDSAKey(rand.Reader)
	genStore := []KeyStore{myKeyNativeStore}[0]
	myKeyInfo := KeyInfo{
		Gun:  "http://example.com/collections",
		Role: "Snapshot",
	}
	err = genStore.AddKey(myKeyInfo, privKey)
	require.NoError(t, err)
	gotCreds, _, err := genStore.GetKey(privKey.ID())
	require.NoError(t, err)
	require.Equal(t, privKey.Private(), gotCreds.Private(), "issue in the add function")
	err = genStore.RemoveKey(privKey.ID())
	require.NoError(t, err)
	gotCreds, role, err := genStore.GetKey(privKey.ID())
	require.Error(t, err)
	require.Equal(t, role, "")
	require.Equal(t, nil, gotCreds, "issue in the remove function")
	newKeyInfo, err := genStore.GetKeyInfo(privKey.ID())
	require.Error(t, err)
	require.Equal(t, "", newKeyInfo.Role, "Key info should be empty")
	require.Equal(t, "", newKeyInfo.Gun, "Key info should be empty")
}

//Testing that remove behaves gracefully on a key that doesn't exist
func TestRemoveFromNativeStoreNoPanic(t *testing.T) {
	myKeyNativeStore, err := NewKeyNativeStore(defaultCredentialsStore)
	genStore := []KeyStore{myKeyNativeStore}[0]
	err = genStore.RemoveKey("randomkeythatshouldnotexistinnativestore(i hope)")
	require.Error(t, err)
}
