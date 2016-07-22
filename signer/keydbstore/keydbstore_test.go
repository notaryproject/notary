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
	return "", false, errors.New("password alias no found")
}

type keyRotator interface {
	trustmanager.KeyStore
	RotateKeyPassphrase(keyID, newPassphraseAlias string) error
}

// A key can only be added to the DB once
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

// a key can be deleted
func testCreateDelete(t *testing.T, dbStore trustmanager.KeyStore) {
	testKey, err := utils.GenerateECDSAKey(rand.Reader)
	require.NoError(t, err)

	// Add a key to the DB
	err = dbStore.AddKey(trustmanager.KeyInfo{Role: "myrole", Gun: "mygun"}, testKey)
	require.NoError(t, err)
	// Currently we ignore roles
	requireGetKeySuccess(t, dbStore, "myrole", testKey)

	// Deleting the key should succeed
	err = dbStore.RemoveKey(testKey.ID())
	require.NoError(t, err)
	requireGetKeyFailure(t, dbStore, testKey.ID())

	// Deleting the key again should succeed even though it's not in the DB
	err = dbStore.RemoveKey(testKey.ID())
	require.NoError(t, err)
	requireGetKeyFailure(t, dbStore, testKey.ID())
}

// key rotation is successful provided the other alias is valid
func testKeyRotation(t *testing.T, dbStore keyRotator, newValidAlias string) data.PrivateKey {
	testKey, err := utils.GenerateECDSAKey(rand.Reader)
	require.NoError(t, err)

	// Test writing new key in database/cache
	err = dbStore.AddKey(trustmanager.KeyInfo{Role: data.CanonicalTimestampRole, Gun: "gun/ignored"}, testKey)
	require.NoError(t, err)

	// Try rotating the key to a valid alias
	err = dbStore.RotateKeyPassphrase(testKey.ID(), newValidAlias)
	require.NoError(t, err)

	// Try rotating the key to an invalid alias
	err = dbStore.RotateKeyPassphrase(testKey.ID(), "invalidAlias")
	require.Error(t, err, "there should be no password for invalidAlias so rotation should fail")

	return testKey
}
