// +build pkcs11

package common

import (
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/theupdateframework/notary"
	"github.com/theupdateframework/notary/trustmanager"
	"github.com/theupdateframework/notary/tuf/data"
)

type HardwareStore struct {
	PassRetriever notary.PassRetriever
	Keys          map[string]HardwareSlot
	BackupStore   trustmanager.KeyStore
	LibLoader     Pkcs11LibLoader
}

// NewHardwareStore returns a Store, given a backup key store to write any
// generated keys to (usually a KeyFileStore)
func NewHardwareStore(backupStore trustmanager.KeyStore, passphraseRetriever notary.PassRetriever) (*HardwareStore, error) {

	s := &HardwareStore{
		PassRetriever: passphraseRetriever,
		Keys:          make(map[string]HardwareSlot),
		BackupStore:   backupStore,
		LibLoader:     DefaultLoader,
	}
	s.ListKeys() // populate keys field
	return s, nil
}

// Name returns a user friendly name for the location this store
func (s HardwareStore) Name() string {
	return hardwareName
}

func (s *HardwareStore) SetLibLoader(loader Pkcs11LibLoader) {
	s.LibLoader = loader
}

// ListKeys returns a list of keys in the hardwarestore
func (s *HardwareStore) ListKeys() map[string]trustmanager.KeyInfo {
	if len(s.Keys) > 0 {
		return BuildKeyMap(s.Keys)
	}
	ctx, session, err := hardwareKeyStore.SetupHSMEnv(s.LibLoader)
	if err != nil {
		logrus.Debugf("No %s found, using alternative key storage: %s", hardwareName, err.Error())
		return nil
	}
	defer Cleanup(ctx, session)

	keys, err := hardwareKeyStore.HardwareListKeys(ctx, session)
	if err != nil {
		logrus.Debugf("Failed to list key from the %s: %s", hardwareName, err.Error())
		return nil
	}
	s.Keys = keys

	return BuildKeyMap(keys)
}

// AddKey puts a key inside the Hardwarestore, as well as writing it to the backup store
func (s *HardwareStore) AddKey(keyInfo trustmanager.KeyInfo, privKey data.PrivateKey) error {
	added, err := s.addKey(privKey.ID(), keyInfo.Role, privKey)
	if err != nil {
		return err
	}
	if added && s.BackupStore != nil {

		err = s.BackupStore.AddKey(keyInfo, privKey)
		if err != nil {
			defer s.RemoveKey(privKey.ID())
			return ErrBackupFailed{err: err.Error()}
		}
	}
	return nil
}

// Only add if we haven't seen the key already.  Return whether the key was
// added.
func (s *HardwareStore) addKey(keyID string, role data.RoleName, privKey data.PrivateKey) (bool, error) {

	if role != data.CanonicalRootRole {
		return false, fmt.Errorf(
			"%s only supports storing root keys, got %s for key: %s", hardwareName, role, keyID)
	}

	ctx, session, err := hardwareKeyStore.SetupHSMEnv(s.LibLoader)
	if err != nil {
		logrus.Debugf("No %s found, using alternative key storage: %s", hardwareName, err.Error())
		return false, err
	}
	defer Cleanup(ctx, session)

	if k, ok := s.Keys[keyID]; ok {
		if k.Role == role {
			return false, nil
		}
	}

	slot, err := hardwareKeyStore.GetNextEmptySlot(ctx, session)
	if err != nil {
		logrus.Debugf("Failed to get an empty %s slot: %s", hardwareName, err.Error())
		return false, err
	}
	logrus.Debugf("Attempting to store key using %s slot %v", hardwareName, slot)
	key := HardwareSlot{
		Role:   role,
		SlotID: slot,
		KeyID:  keyID,
	}
	//err = hardwareKeyStore.AddECDSAKey(ctx, session, privKey, slot, s.PassRetriever, role)
	err = hardwareKeyStore.AddECDSAKey(ctx, session, privKey, key, s.PassRetriever, role)
	if err == nil {
		s.Keys[privKey.ID()] = key
		return true, nil
	}
	logrus.Debugf("Failed to add key to %s: %v", hardwareName, err)

	return false, err
}

// GetKey retrieves a key from the Hardwarestore only (it does not look inside the
// backup store)
func (s *HardwareStore) GetKey(keyID string) (data.PrivateKey, data.RoleName, error) {
	ctx, session, err := hardwareKeyStore.SetupHSMEnv(s.LibLoader)
	if err != nil {
		logrus.Debugf("No %s found, using alternative key storage: %s", hardwareName, err.Error())
		if _, ok := err.(ErrHSMNotPresent); ok {
			err = trustmanager.ErrKeyNotFound{KeyID: keyID}
		}
		return nil, "", err
	}
	defer Cleanup(ctx, session)

	key, ok := s.Keys[keyID]
	if !ok {
		return nil, "", trustmanager.ErrKeyNotFound{KeyID: keyID}
	}

	pubKey, alias, err := hardwareKeyStore.GetECDSAKey(ctx, session, key, s.PassRetriever)
	if err != nil {
		logrus.Debugf("Failed to get key from slot %s: %s", key.SlotID, err.Error())
		return nil, "", err
	}
	if pubKey.ID() != keyID {
		return nil, "", fmt.Errorf("expected root key: %s, but found: %s", keyID, pubKey.ID())
	}

	// privkey is not a privatekey itself, but an object that contains the slot containing the privatekey
	privKey := NewHardwarePrivateKey(key, *pubKey, s.PassRetriever)
	if privKey == nil {
		return nil, "", errors.New("could not initialize new HardwarePrivateKey")
	}

	return privKey, alias, err
}

// RemoveKey deletes a key from the Hardwarestore only (it does not remove it from the
// backup store)
func (s *HardwareStore) RemoveKey(keyID string) error {
	ctx, session, err := hardwareKeyStore.SetupHSMEnv(s.LibLoader)
	if err != nil {
		logrus.Debugf("No %s found, using alternative key storage: %s", hardwareName, err.Error())
		return nil
	}
	defer Cleanup(ctx, session)

	key, ok := s.Keys[keyID]
	if !ok {
		e := fmt.Sprintf("Key not present in %s", hardwareName)
		return errors.New(e)
	}
	err = hardwareKeyStore.HardwareRemoveKey(ctx, session, key, s.PassRetriever, keyID)
	if err == nil {
		delete(s.Keys, keyID)
	} else {
		logrus.Debugf("Failed to remove from the %s KeyID %s: %v", hardwareName, keyID, err)
	}

	return err
}

// GetKeyInfo is not yet implemented
func (s *HardwareStore) GetKeyInfo(keyID string) (trustmanager.KeyInfo, error) {
	return trustmanager.KeyInfo{}, fmt.Errorf("Not yet implemented")
}
