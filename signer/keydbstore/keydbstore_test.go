package keydbstore

import (
	"crypto/rand"
	"errors"
	"fmt"
	"testing"

	"github.com/docker/notary/trustmanager"
	"github.com/docker/notary/tuf/data"
	"github.com/docker/notary/tuf/signed"
	"github.com/docker/notary/tuf/utils"
	"github.com/stretchr/testify/require"
)

func constRetriever(string, string, bool, int) (string, bool, error) {
	return "constantPass", false, nil
}

var validAliases = []string{"validAlias1", "validAlias2"}
var validAliasesAndPasswds = map[string]string{
	"validAlias1": "passphrase_1",
	"validAlias2": "passphrase_2",
}

func multiAliasRetriever(_, alias string, _ bool, _ int) (string, bool, error) {
	if passwd, ok := validAliasesAndPasswds[alias]; ok {
		return passwd, false, nil
	}
	return "", false, errors.New("password alias not found")
}

type keyRotator interface {
	trustmanager.KeyStore
	RotateKeyPassphrase(keyID, newPassphraseAlias string) error
}

type keyActivator interface {
	trustmanager.KeyStore
	GetPendingKey(trustmanager.KeyInfo) (data.PublicKey, error)
}

// A key can only be added to the DB once.  Returns a list of expected keys, and which keys are expected to exist.
func testKeyCanOnlyBeAddedOnce(t *testing.T, dbStore trustmanager.KeyStore) []data.PrivateKey {
	expectedKeys := make([]data.PrivateKey, 2)
	for i := 0; i < len(expectedKeys); i++ {
		testKey, err := utils.GenerateECDSAKey(rand.Reader)
		require.NoError(t, err)
		expectedKeys[i] = testKey
	}

	// Test writing new key in database alone, not cache
	err := dbStore.AddKey(trustmanager.KeyInfo{Role: data.CanonicalTimestampRole, Gun: "gun"}, expectedKeys[0])
	require.NoError(t, err)
	requireGetKeySuccess(t, dbStore, data.CanonicalTimestampRole, expectedKeys[0])

	// Test writing the same key in the database. Should fail.
	err = dbStore.AddKey(trustmanager.KeyInfo{Role: data.CanonicalTimestampRole, Gun: "gun"}, expectedKeys[0])
	require.Error(t, err, "failed to add private key to database:")

	// Test writing new key succeeds
	err = dbStore.AddKey(trustmanager.KeyInfo{Role: data.CanonicalTimestampRole, Gun: "gun"}, expectedKeys[1])
	require.NoError(t, err)

	return expectedKeys
}

// a key can be deleted - returns a list of expected keys
func testCreateDelete(t *testing.T, dbStore trustmanager.KeyStore) []data.PrivateKey {
	testKeys := make([]data.PrivateKey, 2)
	for i := 0; i < len(testKeys); i++ {
		testKey, err := utils.GenerateECDSAKey(rand.Reader)
		require.NoError(t, err)
		testKeys[i] = testKey

		// Add them to the DB
		err = dbStore.AddKey(trustmanager.KeyInfo{Role: data.CanonicalTimestampRole, Gun: "gun"}, testKey)
		require.NoError(t, err)
		requireGetKeySuccess(t, dbStore, data.CanonicalTimestampRole, testKey)
	}

	// Deleting the key should succeed and only remove the key that was deleted
	require.NoError(t, dbStore.RemoveKey(testKeys[0].ID()))
	requireGetKeyFailure(t, dbStore, testKeys[0].ID())
	requireGetKeySuccess(t, dbStore, data.CanonicalTimestampRole, testKeys[1])

	// Deleting the key again should succeed even though it's not in the DB
	require.NoError(t, dbStore.RemoveKey(testKeys[0].ID()))
	requireGetKeyFailure(t, dbStore, testKeys[0].ID())

	return testKeys[1:]
}

// key rotation is successful provided the other alias is valid.
// Returns the key that was rotated and one that was not rotated
func testKeyRotation(t *testing.T, dbStore keyRotator, newValidAlias string) (data.PrivateKey, data.PrivateKey) {
	testKeys := make([]data.PrivateKey, 2)
	for i := 0; i < len(testKeys); i++ {
		testKey, err := utils.GenerateECDSAKey(rand.Reader)
		require.NoError(t, err)
		testKeys[i] = testKey

		// Add them to the DB
		err = dbStore.AddKey(trustmanager.KeyInfo{Role: data.CanonicalTimestampRole, Gun: "gun"}, testKey)
		require.NoError(t, err)
	}

	// Try rotating the key to a valid alias
	err := dbStore.RotateKeyPassphrase(testKeys[0].ID(), newValidAlias)
	require.NoError(t, err)

	// Try rotating the key to an invalid alias
	err = dbStore.RotateKeyPassphrase(testKeys[0].ID(), "invalidAlias")
	require.Error(t, err, "there should be no password for invalidAlias so rotation should fail")

	return testKeys[0], testKeys[1]
}

type badReader struct{}

func (b badReader) Read([]byte) (n int, err error) {
	return 0, fmt.Errorf("Nope, not going to read")
}

// Signing with a key marks it as active if the signing is successful.  Marking as active is successful no matter what,
// but should only activate a key that exists in the DB.
// Returns the key that was used and one that was not
func testSigningWithKeyMarksAsActive(t *testing.T, dbStore trustmanager.KeyStore) (data.PrivateKey, data.PrivateKey) {
	testKeys := make([]data.PrivateKey, 3)
	for i := 0; i < len(testKeys); i++ {
		testKey, err := utils.GenerateECDSAKey(rand.Reader)
		require.NoError(t, err)

		// Add them to the DB
		err = dbStore.AddKey(trustmanager.KeyInfo{Role: data.CanonicalTimestampRole, Gun: "gun"}, testKey)
		require.NoError(t, err)
		requireGetKeySuccess(t, dbStore, data.CanonicalTimestampRole, testKey)

		// store the gotten key, because that key is special
		gottenKey, _, err := dbStore.GetKey(testKey.ID())
		require.NoError(t, err)
		testKeys[i] = gottenKey
	}

	// sign successfully with the first key - this key will become active
	msg := []byte("successful")
	sig, err := testKeys[0].Sign(rand.Reader, msg, nil)
	require.NoError(t, err)
	require.NoError(t, signed.Verifiers[data.ECDSASignature].Verify(
		data.PublicKeyFromPrivate(testKeys[0]), sig, msg))

	// sign unsuccessfully with the second key - this key should remain inactive
	sig, err = testKeys[1].Sign(badReader{}, []byte("unsuccessful"), nil)
	require.Error(t, err)
	require.Equal(t, "Nope, not going to read", err.Error())
	require.Nil(t, sig)

	// delete the third key from the DB - sign should still succeed, even though
	// this key cannot be marked as active anymore due to it not existing
	// (this probably won't return an error)
	require.NoError(t, dbStore.RemoveKey(testKeys[2].ID()))
	requireGetKeyFailure(t, dbStore, testKeys[2].ID())
	msg = []byte("successful, not active")
	sig, err = testKeys[2].Sign(rand.Reader, msg, nil)
	require.NoError(t, err)
	require.NoError(t, signed.Verifiers[data.ECDSASignature].Verify(
		data.PublicKeyFromPrivate(testKeys[2]), sig, msg))

	return testKeys[0], testKeys[1] // testKeys[2] should no longer exist in the DB
}

func testGetPendingKey(t *testing.T, dbStore keyActivator) (data.PrivateKey, data.PrivateKey) {
	// Create a test key and add it to the db such that it will be pending (never marked active)
	keyInfo := trustmanager.KeyInfo{Role: data.CanonicalSnapshotRole, Gun: "gun"}
	pendingTestKey, err := utils.GenerateECDSAKey(rand.Reader)
	require.NoError(t, err)
	requireGetKeyFailure(t, dbStore, pendingTestKey.ID())
	err = dbStore.AddKey(keyInfo, pendingTestKey)
	require.NoError(t, err)
	requireGetKeySuccess(t, dbStore, data.CanonicalSnapshotRole, pendingTestKey)

	retrievedKey, err := dbStore.GetPendingKey(keyInfo)
	require.NoError(t, err)
	require.Equal(t, pendingTestKey.Public(), retrievedKey.Public())

	// Now create an active key with the same keyInfo
	activeTestKey, err := utils.GenerateECDSAKey(rand.Reader)
	require.NoError(t, err)
	requireGetKeyFailure(t, dbStore, activeTestKey.ID())
	err = dbStore.AddKey(keyInfo, activeTestKey)
	require.NoError(t, err)
	requireGetKeySuccess(t, dbStore, data.CanonicalSnapshotRole, activeTestKey)

	// Mark as active by signing
	_, err = activeTestKey.Sign(rand.Reader, []byte("msg"), nil)
	require.NoError(t, err)

	// We should still get back the original pending key on GetPendingKey
	retrievedKey, err = dbStore.GetPendingKey(keyInfo)
	require.NoError(t, err)
	require.NotEqual(t, activeTestKey.Public(), retrievedKey.Public())
	require.Equal(t, pendingTestKey.Public(), retrievedKey.Public())

	return pendingTestKey, activeTestKey
}
