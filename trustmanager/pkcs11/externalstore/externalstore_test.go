// +build pkcs11

package externalstore

import (
	"crypto/rand"
	"errors"
	"strings"
	"testing"

	"github.com/miekg/pkcs11"
	"github.com/stretchr/testify/require"
	"github.com/theupdateframework/notary/passphrase"
	"github.com/theupdateframework/notary/trustmanager"
	"github.com/theupdateframework/notary/trustmanager/pkcs11/common"
	"github.com/theupdateframework/notary/tuf/data"
	"github.com/theupdateframework/notary/tuf/utils"
)

var (
	ret          = passphrase.ConstantRetriever("passphrase")
	testNumSlots = 10
)

func init() {
	ks := NewKeyStore()
	common.SetKeyStore(ks)
}

func getKeyStoreAndSession(t *testing.T) (*KeyStore, pkcs11.SessionHandle) {
	ks := NewKeyStore()
	session, err := ks.SetupHSMEnv()
	if err != nil {
		ks.Close()
		t.Fatalf("Error setting up HSMEnv: %v", err)
	}
	return ks, session
}

func cleanup(ks *KeyStore, session pkcs11.SessionHandle) {
	ks.Cleanup(session)
	ks.Close()
}

func TestConnection(t *testing.T) {
	ks := NewKeyStore()
	defer ks.Close()
	name := ks.Name()
	if name == "ExternalStore" {
		t.Fail()
	}
}

func TestAddAndGetECDSAKey(t *testing.T) {
	ks, session := getKeyStoreAndSession(t)
	defer cleanup(ks, session)
	privKey, err := utils.GenerateECDSAKey(rand.Reader)
	require.NoError(t, err)
	slotID, err := ks.GetNextEmptySlot(session)
	require.NoError(t, err)
	slot := common.HardwareSlot{
		Role:   data.CanonicalRootRole,
		SlotID: slotID,
		KeyID:  privKey.ID(),
	}
	err = ks.AddECDSAKey(session, privKey, slot, ret, data.CanonicalRootRole)
	require.NoError(t, err)
	pubKey, role, err := ks.GetECDSAKey(session, slot, ret)
	require.NoError(t, err)
	require.Equal(t, role, data.CanonicalRootRole)
	require.Equal(t, privKey.Public(), pubKey.Public())
}

// remove all Keys
func TestClearAllKeys(t *testing.T) {
	ks, session := getKeyStoreAndSession(t)
	defer cleanup(ks, session)
	list, err := ks.HardwareListKeys(session)
	if err != nil && !strings.Contains(err.Error(), "no keys found in") {
		require.NoError(t, err)
	}
	t.Logf("Found %d keys", len(list))
	i := 0
	for id, slot := range list {
		err = ks.HardwareRemoveKey(session, slot, ret, id)
		require.NoError(t, err)
		i++
	}
	t.Logf("Cleared %d keys", i)
}

// -----------------------------------------------------------------------------
// Start old Tests

func clearAllKeys(t *testing.T) {
	store, _ := common.NewHardwareStore(trustmanager.NewKeyMemoryStore(ret), ret)

	for k := range store.ListKeys() {
		err := store.RemoveKey(k)
		require.NoError(t, err)
	}
}

func testAddKey(t *testing.T, store trustmanager.KeyStore) (data.PrivateKey, error) {
	privKey, err := utils.GenerateECDSAKey(rand.Reader)
	require.NoError(t, err)

	err = store.AddKey(trustmanager.KeyInfo{Role: data.CanonicalRootRole, Gun: ""}, privKey)
	return privKey, err
}

func addMaxKeys(t *testing.T, store trustmanager.KeyStore) []string {
	var keys []string
	// create the maximum number of keys
	for i := 0; i < testNumSlots; i++ {
		privKey, err := testAddKey(t, store)
		require.NoError(t, err)
		keys = append(keys, privKey.ID())
	}
	return keys
}

// We can add keys enough times to fill up all the slots in the Externalstore.
// They are backed up, and we can then list them and get the keys.
func TestExternalstoreAddKeysAndRetrieve(t *testing.T) {

	if !common.IsAccessible() {
		t.Skip("Must have Hardwarestore access.")
	}

	clearAllKeys(t)

	// create 4 keys on the original store
	backup := trustmanager.NewKeyMemoryStore(ret)
	store, err := common.NewHardwareStore(backup, ret)
	require.NoError(t, err)
	keys := addMaxKeys(t, store)

	// create a new store, since we want to be sure the original store's cache
	// is not masking any issues
	cleanStore, err := common.NewHardwareStore(trustmanager.NewKeyMemoryStore(ret), ret)
	require.NoError(t, err)

	// All 4 keys should be in the original store, in the clean store (which
	// makes sure the keys are actually on the Externalstore and not on the original
	// store's cache, and on the backup store)
	for _, store := range []trustmanager.KeyStore{store, cleanStore, backup} {
		listedKeys := store.ListKeys()
		require.Len(t, listedKeys, testNumSlots)
		for _, k := range keys {
			r, ok := listedKeys[k]
			require.True(t, ok)
			require.Equal(t, data.CanonicalRootRole, r.Role)

			_, _, err := store.GetKey(k)
			require.NoError(t, err)
		}
	}
}

// Test that we can successfully keys enough times to fill up all the slots in the Externalstore, even without a backup store
func TestExternalstoreAddKeysWithoutBackup(t *testing.T) {
	if !common.IsAccessible() {
		t.Skip("Must have Hardwarestore access.")
	}
	clearAllKeys(t)

	// create 4 keys on the original store
	store, err := common.NewHardwareStore(nil, ret)
	require.NoError(t, err)
	keys := addMaxKeys(t, store)

	// create a new store, since we want to be sure the original store's cache
	// is not masking any issues
	cleanStore, err := common.NewHardwareStore(trustmanager.NewKeyMemoryStore(ret), ret)
	require.NoError(t, err)

	// All 4 keys should be in the original store, in the clean store (which
	// makes sure the keys are actually on the Externalstore and not on the original
	// store's cache)
	for _, store := range []trustmanager.KeyStore{store, cleanStore} {
		listedKeys := store.ListKeys()
		require.Len(t, listedKeys, testNumSlots)
		for _, k := range keys {
			r, ok := listedKeys[k]
			require.True(t, ok)
			require.Equal(t, data.CanonicalRootRole, r.Role)

			_, _, err := store.GetKey(k)
			require.NoError(t, err)
		}
	}
}

// If some random key in the middle was removed, adding a key will work (keys
// do not have to be deleted/added in order)
func TestExternalstoreAddKeyCanAddToMiddleSlot(t *testing.T) {
	if !common.IsAccessible() {
		t.Skip("Must have Hardwarestore access.")
	}
	clearAllKeys(t)

	// create 4 keys on the original store
	backup := trustmanager.NewKeyMemoryStore(ret)
	store, err := common.NewHardwareStore(backup, ret)
	require.NoError(t, err)
	keys := addMaxKeys(t, store)

	// delete one of the middle keys, and assert we can still create a new key
	keyIDToDelete := keys[testNumSlots/2]
	err = store.RemoveKey(keyIDToDelete)
	require.NoError(t, err)

	newKey, err := testAddKey(t, store)
	require.NoError(t, err)

	// create a new store, since we want to be sure the original store's cache
	// is not masking any issues
	cleanStore, err := common.NewHardwareStore(trustmanager.NewKeyMemoryStore(ret), ret)
	require.NoError(t, err)

	// The new key should be in the original store, in the new clean store, and
	// in the backup store.  The old key should not be in the original store,
	// or the new clean store.
	for _, store := range []trustmanager.KeyStore{store, cleanStore, backup} {
		// new key should appear in all stores
		gottenKey, _, err := store.GetKey(newKey.ID())
		require.NoError(t, err)
		require.Equal(t, gottenKey.ID(), newKey.ID())

		listedKeys := store.ListKeys()
		_, ok := listedKeys[newKey.ID()]
		require.True(t, ok)

		// old key should not be in the non-backup stores
		if store != backup {
			_, _, err := store.GetKey(keyIDToDelete)
			require.Error(t, err)
			_, ok = listedKeys[keyIDToDelete]
			require.False(t, ok)
		}
	}
}

type nonworkingBackup struct {
	trustmanager.GenericKeyStore
}

// AddKey stores the contents of a PEM-encoded private key as a PEM block
func (s *nonworkingBackup) AddKey(keyInfo trustmanager.KeyInfo, privKey data.PrivateKey) error {
	return errors.New("nope")
}

// If, when adding a key to the Externalstore, we can't back up the key, it should
// be removed from the Externalstore too because otherwise there is no way for
// the user to later get a backup of the key.
func TestExternalstoreAddKeyRollsBackIfCannotBackup(t *testing.T) {
	if !common.IsAccessible() {
		t.Skip("Must have Hardwarestore access.")
	}
	clearAllKeys(t)

	backup := &nonworkingBackup{
		GenericKeyStore: *trustmanager.NewKeyMemoryStore(ret),
	}
	store, err := common.NewHardwareStore(backup, ret)
	require.NoError(t, err)

	_, err = testAddKey(t, store)
	require.Error(t, err)
	require.IsType(t, common.ErrBackupFailed{}, err)

	// there should be no keys on the Externalstore
	require.Len(t, cleanListKeys(t), 0)
}

// If, when adding a key to the Externalstore, and it already exists, we succeed
// without adding it to the backup store.
func TestExternalstoreAddDuplicateKeySucceedsButDoesNotBackup(t *testing.T) {
	if !common.IsAccessible() {
		t.Skip("Must have Hardwarestore access.")
	}
	clearAllKeys(t)

	origStore, err := common.NewHardwareStore(trustmanager.NewKeyMemoryStore(ret), ret)
	require.NoError(t, err)

	key, err := testAddKey(t, origStore)
	require.NoError(t, err)

	backup := trustmanager.NewKeyMemoryStore(ret)
	cleanStore, err := common.NewHardwareStore(backup, ret)
	require.NoError(t, err)
	require.Len(t, cleanStore.ListKeys(), 1)

	err = cleanStore.AddKey(trustmanager.KeyInfo{Role: data.CanonicalRootRole, Gun: ""}, key)
	require.NoError(t, err)

	// there should be just 1 key on the Externalstore
	require.Len(t, cleanListKeys(t), 1)
	// nothing was added to the backup
	require.Len(t, backup.ListKeys(), 0)
}

// RemoveKey removes a key from the Externalstore, but not from the backup store.
func TestExternalstoreRemoveKey(t *testing.T) {
	if !common.IsAccessible() {
		t.Skip("Must have Hardwarestore access.")
	}
	clearAllKeys(t)

	backup := trustmanager.NewKeyMemoryStore(ret)
	store, err := common.NewHardwareStore(backup, ret)
	require.NoError(t, err)

	key, err := testAddKey(t, store)
	require.NoError(t, err)
	err = store.RemoveKey(key.ID())
	require.NoError(t, err)

	// key remains in the backup store
	backupKey, role, err := backup.GetKey(key.ID())
	require.NoError(t, err)
	require.Equal(t, data.CanonicalRootRole, role)
	require.Equal(t, key.ID(), backupKey.ID())

	// create a new store, since we want to be sure the original store's cache
	// is not masking any issues
	cleanStore, err := common.NewHardwareStore(trustmanager.NewKeyMemoryStore(ret), ret)
	require.NoError(t, err)

	// key is not in either the original store or the clean store
	for _, store := range []*common.HardwareStore{store, cleanStore} {
		_, _, err := store.GetKey(key.ID())
		require.Error(t, err)
	}
}

// If there are keys in the backup store but no keys in the Externalstore,
// listing and getting cannot access the keys in the backup store
func TestExternalstoreListAndGetKeysIgnoresBackup(t *testing.T) {
	if !common.IsAccessible() {
		t.Skip("Must have Hardwarestore access.")
	}
	clearAllKeys(t)

	backup := trustmanager.NewKeyMemoryStore(ret)
	key, err := testAddKey(t, backup)
	require.NoError(t, err)

	store, err := common.NewHardwareStore(trustmanager.NewKeyMemoryStore(ret), ret)
	require.NoError(t, err)
	require.Len(t, store.ListKeys(), 0)
	_, _, err = store.GetKey(key.ID())
	require.Error(t, err)
}

// Get a YubiPrivateKey.  Check that it has the right algorithm, etc, and
// specifically that you cannot get the private bytes out.  Assume we can
// sign something.
func TestExternalstoreKeyAndSign(t *testing.T) {
	if !common.IsAccessible() {
		t.Skip("Must have Hardwarestore access.")
	}
	clearAllKeys(t)

	store, err := common.NewHardwareStore(trustmanager.NewKeyMemoryStore(ret), ret)
	require.NoError(t, err)

	ecdsaPrivateKey, err := testAddKey(t, store)
	require.NoError(t, err)

	yubiPrivateKey, _, err := store.GetKey(ecdsaPrivateKey.ID())
	require.NoError(t, err)

	require.Equal(t, data.ECDSAKey, yubiPrivateKey.Algorithm())
	require.Equal(t, data.ECDSASignature, yubiPrivateKey.SignatureAlgorithm())
	require.Equal(t, ecdsaPrivateKey.Public(), yubiPrivateKey.Public())
	require.Nil(t, yubiPrivateKey.Private())

	// The signature should be verified, but the importing the verifiers causes
	// an import cycle.  A bigger refactor needs to be done to fix it.
	msg := []byte("Hello there")
	_, err = yubiPrivateKey.Sign(rand.Reader, msg, nil)
	require.NoError(t, err)
}

// Create a new store, so that we avoid any cache issues, and list keys
func cleanListKeys(t *testing.T) map[string]trustmanager.KeyInfo {
	cleanStore, err := common.NewHardwareStore(trustmanager.NewKeyMemoryStore(ret), ret)
	require.NoError(t, err)
	return cleanStore.ListKeys()
}
