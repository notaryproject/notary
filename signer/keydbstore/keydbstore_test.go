package keydbstore

import (
	"crypto/rand"
	"errors"
	"fmt"
	"testing"

	"github.com/docker/notary/trustmanager"
	"github.com/docker/notary/tuf/data"
	"github.com/docker/notary/tuf/utils"
	"github.com/stretchr/testify/require"
)

func constRetriever(string, string, bool, int) (string, bool, error) {
	return "passphrase_1", false, nil
}

var validAliases = []string{"validAlias1", "validAlias2"}

func multiAliasRetriever(_, alias string, _ bool, _ int) (string, bool, error) {
	for i, validAlias := range validAliases {
		if alias == validAlias {
			return fmt.Sprintf("passphrase_%d", i), false, nil
		}
	}
	return "", false, errors.New("password alias no found")
}

type keyRotator interface {
	trustmanager.KeyStore
	RotateKeyPassphrase(keyID, newPassphraseAlias string) error
}

// A key can only be added to the DB once
func testKeyCanOnlyBeAddedOnce(t *testing.T, dbStore trustmanager.KeyStore) {
	testKey, err := utils.GenerateECDSAKey(rand.Reader)
	require.NoError(t, err)

	// Test writing new key in database alone, not cache
	err = dbStore.AddKey(trustmanager.KeyInfo{Role: data.CanonicalTimestampRole, Gun: "gun/ignored"}, testKey)
	require.NoError(t, err)
	// Currently we ignore roles
	requireGetKeySuccess(t, dbStore, "", testKey)

	// Test writing the same key in the database. Should fail.
	err = dbStore.AddKey(trustmanager.KeyInfo{Role: data.CanonicalTimestampRole, Gun: "gun/ignored"}, testKey)
	require.Error(t, err, "failed to add private key to database:")

	anotherTestKey, err := utils.GenerateECDSAKey(rand.Reader)
	require.NoError(t, err)

	// Test writing new key succeeds
	err = dbStore.AddKey(trustmanager.KeyInfo{Role: data.CanonicalTimestampRole, Gun: "gun/ignored"}, anotherTestKey)
	require.NoError(t, err)
}

// a key can be deleted
func testCreateDelete(t *testing.T, dbStore trustmanager.KeyStore) {
	testKey, err := utils.GenerateECDSAKey(rand.Reader)
	require.NoError(t, err)

	// Add a key to the DB
	err = dbStore.AddKey(trustmanager.KeyInfo{Role: "", Gun: ""}, testKey)
	require.NoError(t, err)
	// Currently we ignore roles
	requireGetKeySuccess(t, dbStore, "", testKey)

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
func testKeyRotation(t *testing.T, dbStore keyRotator, newValidAlias string) {
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
}
