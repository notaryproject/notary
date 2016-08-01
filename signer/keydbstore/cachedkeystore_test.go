package keydbstore

import (
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/docker/notary/trustmanager"
	"github.com/docker/notary/tuf/data"
	"github.com/docker/notary/tuf/utils"
	"github.com/stretchr/testify/require"
)

// gets a key from the DB store, and asserts that the key is the expected key
func requireGetKeySuccess(t *testing.T, dbStore trustmanager.KeyStore, expectedRole string, expectedKey data.PrivateKey) {
	retrKey, role, err := dbStore.GetKey(expectedKey.ID())
	require.NoError(t, err)
	require.Equal(t, expectedRole, role)
	require.Equal(t, retrKey.Private(), expectedKey.Private())
	require.Equal(t, retrKey.Algorithm(), expectedKey.Algorithm())
	require.Equal(t, retrKey.Public(), expectedKey.Public())
	require.Equal(t, retrKey.SignatureAlgorithm(), expectedKey.SignatureAlgorithm())
}

// closes the DB connection first so we can test that the successful get was
// from the cache
func requireGetKeySuccessFromCache(t *testing.T, cachedStore, underlyingStore trustmanager.KeyStore, expectedRole string, expectedKey data.PrivateKey) {
	require.NoError(t, underlyingStore.RemoveKey(expectedKey.ID()))
	requireGetKeySuccess(t, cachedStore, expectedRole, expectedKey)
}

func requireGetKeyFailure(t *testing.T, dbStore trustmanager.KeyStore, keyID string) {
	_, _, err := dbStore.GetKey(keyID)
	require.IsType(t, trustmanager.ErrKeyNotFound{}, err)
}

type unAddableKeyStore struct {
	trustmanager.KeyStore
}

func (u unAddableKeyStore) AddKey(_ trustmanager.KeyInfo, _ data.PrivateKey) error {
	return fmt.Errorf("Can't add to keystore!")
}

type unRemoveableKeyStore struct {
	trustmanager.KeyStore
	failToRemove bool
}

func (u unRemoveableKeyStore) RemoveKey(keyID string) error {
	if u.failToRemove {
		return fmt.Errorf("Can't remove from keystore!")
	}
	return u.KeyStore.RemoveKey(keyID)
}

// Getting a key, on succcess, populates the cache.
func TestGetSuccessPopulatesCache(t *testing.T) {
	underlying := trustmanager.NewKeyMemoryStore(constRetriever)
	cached := NewCachedKeyStore(underlying)

	testKey, err := utils.GenerateECDSAKey(rand.Reader)
	require.NoError(t, err)

	// nothing there yet
	requireGetKeyFailure(t, cached, testKey.ID())

	// Add key to underlying store only
	err = underlying.AddKey(trustmanager.KeyInfo{Role: data.CanonicalTimestampRole, Gun: "gun"}, testKey)
	require.NoError(t, err)

	// getting for the first time is successful, and after that getting from cache should be too
	requireGetKeySuccess(t, cached, data.CanonicalTimestampRole, testKey)
	requireGetKeySuccessFromCache(t, cached, underlying, data.CanonicalTimestampRole, testKey)
}

// Creating a key, on success, populates the cache, but does not do so on failure
func TestAddKeyPopulatesCacheIfSuccessful(t *testing.T) {
	var underlying trustmanager.KeyStore
	underlying = trustmanager.NewKeyMemoryStore(constRetriever)
	cached := NewCachedKeyStore(underlying)

	testKeys := make([]data.PrivateKey, 2)
	for i := 0; i < 2; i++ {
		privKey, err := utils.GenerateECDSAKey(rand.Reader)
		require.NoError(t, err)
		testKeys[i] = privKey
	}

	// Writing in the keystore succeeds
	err := cached.AddKey(trustmanager.KeyInfo{Role: data.CanonicalTimestampRole, Gun: "gun"}, testKeys[0])
	require.NoError(t, err)

	// Now even if it's deleted from the underlying database, it's fine because it's cached
	requireGetKeySuccessFromCache(t, cached, underlying, data.CanonicalTimestampRole, testKeys[0])

	// Writing in the keystore fails
	underlying = unAddableKeyStore{KeyStore: underlying}
	cached = NewCachedKeyStore(underlying)
	err = cached.AddKey(trustmanager.KeyInfo{Role: data.CanonicalTimestampRole, Gun: "gun"}, testKeys[1])
	require.Error(t, err)

	// And now it can't be found in either DB
	requireGetKeyFailure(t, cached, testKeys[1].ID())
}

// Deleting a key, no matter whether we succeed in the underlying layer or not, evicts the cached key.
func TestDeleteKeyRemovesKeyFromCache(t *testing.T) {
	underlying := trustmanager.NewKeyMemoryStore(constRetriever)
	cached := NewCachedKeyStore(underlying)

	testKey, err := utils.GenerateECDSAKey(rand.Reader)
	require.NoError(t, err)

	// Write the key, which puts it in the cache
	err = cached.AddKey(trustmanager.KeyInfo{Role: data.CanonicalTimestampRole, Gun: "gun"}, testKey)
	require.NoError(t, err)

	// Deleting removes the key from the cache and the underlying store
	err = cached.RemoveKey(testKey.ID())
	require.NoError(t, err)
	requireGetKeyFailure(t, cached, testKey.ID())

	// Now set up an underlying store where the key can't be deleted
	failingUnderlying := unRemoveableKeyStore{KeyStore: underlying, failToRemove: true}
	cached = NewCachedKeyStore(failingUnderlying)
	err = cached.AddKey(trustmanager.KeyInfo{Role: data.CanonicalTimestampRole, Gun: "gun"}, testKey)
	require.NoError(t, err)

	// Deleting fails to remove the key from the underlying store
	err = cached.RemoveKey(testKey.ID())
	require.Error(t, err)
	requireGetKeySuccess(t, failingUnderlying, data.CanonicalTimestampRole, testKey)

	// now actually remove the key from the underlying store to test that it's gone from the cache
	failingUnderlying.failToRemove = false
	require.NoError(t, failingUnderlying.RemoveKey(testKey.ID()))

	// and it's not in the cache
	requireGetKeyFailure(t, cached, testKey.ID())
}
