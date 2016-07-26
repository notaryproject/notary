package keydbstore

import (
	"crypto/rand"
	"errors"
	"testing"

	"github.com/docker/notary/trustmanager"
	"github.com/docker/notary/tuf/data"
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
	MarkActive(keyID string) error
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

// marking a key as active is successful no matter what, but should only activate a key
// that exists in the DB.
// Returns the key that was used and one that was not
func testMarkKeyActive(t *testing.T, dbStore keyActivator) (data.PrivateKey, data.PrivateKey) {
	testKeys := make([]data.PrivateKey, 2)
	for i := 0; i < len(testKeys); i++ {
		testKey, err := utils.GenerateECDSAKey(rand.Reader)
		require.NoError(t, err)
		testKeys[i] = testKey

		// MarkActive succeeds whether or not a key exists
		require.NoError(t, dbStore.MarkActive(testKey.ID()))
		requireGetKeyFailure(t, dbStore, testKey.ID())

		// Add them to the DB
		err = dbStore.AddKey(trustmanager.KeyInfo{Role: data.CanonicalTimestampRole, Gun: "gun"}, testKey)
		require.NoError(t, err)
		requireGetKeySuccess(t, dbStore, data.CanonicalTimestampRole, testKey)
	}

	// MarkActive shoudlshould succeed on a key that exists
	require.NoError(t, dbStore.MarkActive(testKeys[0].ID()))

	return testKeys[0], testKeys[1]
}
