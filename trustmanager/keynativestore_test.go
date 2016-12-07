package trustmanager

import (
	"crypto/rand"
	"testing"

	"github.com/docker/notary/passphrase"
	"github.com/docker/notary/tuf/utils"
	"github.com/stretchr/testify/require"
)

const testingPassphrase = "password"

func nativeStoreNotDefined(store *KeyNativeStore) bool {
	privKey, err := utils.GenerateECDSAKey(rand.Reader)
	myKeyInfo := KeyInfo{
		Gun:  "http://example.com/collections",
		Role: "Snapshot",
	}
	err = store.AddKey(myKeyInfo, privKey)
	defer store.RemoveKey(privKey.ID())
	if err != nil {
		return true
	}
	return false
}

//will only test for your OS- could need to be more generic
//Testing that add is working while simultaneously testing that getkey and getkeyinfo are working for a keychain we know exists
func TestAddGetKeyInNativeStore(t *testing.T) {
	myKeyNativeStore, err := NewKeyNativeStore(passphrase.ConstantRetriever(testingPassphrase))
	require.NoError(t, err)
	if nativeStoreNotDefined(myKeyNativeStore) {
		t.Skip("You need to make the native store in docker-credential-helpers and point your path to the binary")
	}
	privKey, err := utils.GenerateECDSAKey(rand.Reader)
	genStore := []KeyStore{myKeyNativeStore}[0]
	myKeyInfo := KeyInfo{
		Gun:  "http://example.com/collections",
		Role: "Snapshot",
	}
	err = genStore.AddKey(myKeyInfo, privKey)
	require.NoError(t, err)
	defer genStore.RemoveKey(privKey.ID())
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
	myKeyNativeStore, err := NewKeyNativeStore(passphrase.ConstantRetriever(testingPassphrase))
	require.NoError(t, err)
	if nativeStoreNotDefined(myKeyNativeStore) {
		t.Skip("You need to make the native store in docker-credential-helpers and point your path to the binary")
	}
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
//The inconsistent behavior of RemoveKey across operating systems is accounted for here
func TestRemoveFromNativeStoreNoPanic(t *testing.T) {
	myKeyNativeStore, err := NewKeyNativeStore(passphrase.ConstantRetriever(testingPassphrase))
	require.NoError(t, err)
	if nativeStoreNotDefined(myKeyNativeStore) {
		t.Skip("You need to make the native store in docker-credential-helpers and point your path to the binary")
	}
	genStore := []KeyStore{myKeyNativeStore}[0]
	err = genStore.RemoveKey("randomkeythatshouldnotexistinnativestore(i hope)")
	if defaultCredentialsStore == "secretservice" {
		require.NoError(t, err)
	}
	if defaultCredentialsStore == "osxkeychain" {
		require.Error(t, err)
	}
}

//Testing that Get exports correctly encrypted information given a certain passphrase
func TestGetFromNativeStore(t *testing.T) {
	myKeyNativeStore, err := NewKeyNativeStore(passphrase.ConstantRetriever(testingPassphrase))
	require.NoError(t, err)
	if nativeStoreNotDefined(myKeyNativeStore) {
		t.Skip("You need to make the native store in docker-credential-helpers and point your path to the binary")
	}
	privKey, _ := utils.GenerateECDSAKey(rand.Reader)
	myKeyInfo := KeyInfo{
		Gun:  "http://example.com/collections",
		Role: "Snapshot",
	}
	err = myKeyNativeStore.AddKey(myKeyInfo, privKey)
	defer myKeyNativeStore.RemoveKey(privKey.ID())
	encryptedSecret, err := myKeyNativeStore.Get(privKey.ID())
	require.NoError(t, err)
	// decrypt to test the encryption
	gotKey, err := utils.ParsePEMPrivateKey(encryptedSecret, testingPassphrase)
	require.Equal(t, privKey.Private(), gotKey.Private(), "Exported secret has not been encrypted properly as per the testingPassphrase")
}

//Testing that get behaves gracefully on a key that doesn't exist
func TestGetFromNativeStoreNoPanic(t *testing.T) {
	myKeyNativeStore, err := NewKeyNativeStore(passphrase.ConstantRetriever(testingPassphrase))
	require.NoError(t, err)
	if nativeStoreNotDefined(myKeyNativeStore) {
		t.Skip("You need to make the native store in docker-credential-helpers and point your path to the binary")
	}
	encryptedSecret, err := myKeyNativeStore.Get("randomkeythatshouldnotexistinnativestore(i hope)")
	require.Error(t, err)
	var expected []byte
	expected = nil
	require.Equal(t, encryptedSecret, expected, "Not expected behavior")
}
