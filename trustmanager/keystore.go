package trustmanager

import (
	"encoding/pem"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"bytes"
	"github.com/Sirupsen/logrus"
	"github.com/docker/notary"
	store "github.com/docker/notary/storage"
	"github.com/docker/notary/tuf/data"
	"github.com/docker/notary/tuf/utils"
	"os"
)

type keyInfoMap map[string]KeyInfo

// KeyInfo stores the role, path, and gun for a corresponding private key ID
// It is assumed that each private key ID is unique
type KeyInfo struct {
	Gun  string
	Role string
}

// GenericKeyStore is a wrapper for Storage instances that provides
// translation between the []byte form and Public/PrivateKey objects
type GenericKeyStore struct {
	store Storage
	sync.Mutex
	notary.PassRetriever
	cachedKeys map[string]*cachedKey
	keyInfoMap
}

// NewKeyFileStore returns a new KeyFileStore creating a private directory to
// hold the keys.
func NewKeyFileStore(baseDir string, p notary.PassRetriever) (*GenericKeyStore, error) {
	fileStore, err := store.NewPrivateKeyFileStorage(baseDir, notary.KeyExtension)
	if err != nil {
		return nil, err
	}
	migrateTo0Dot4(fileStore)
	store := NewGenericKeyStore(fileStore, p)
	return store, nil
}

func migrateTo0Dot4(s Storage) {
	for _, file := range s.ListFiles() {
		keyID := filepath.Base(file)
		fileDir := filepath.Dir(file)
		d, _ := s.Get(file)
		block, _ := pem.Decode(d)
		if block == nil {
			logrus.Warn("Key data for ", file, " may have been tampered with/ is invalid. The key has not been migrated and may not be available")
			continue
		}
		if strings.HasPrefix(fileDir, notary.RootKeysSubdir) {
			fileDir = strings.TrimPrefix(fileDir, notary.RootKeysSubdir)
			if strings.Contains(keyID, "_") {
				role := strings.Split(keyID, "_")[1]
				keyID = strings.TrimSuffix(keyID, "_"+role)
				block.Headers["role"] = role
			}
		} else if strings.HasPrefix(fileDir, notary.NonRootKeysSubdir) {
			fileDir = strings.TrimPrefix(fileDir, notary.NonRootKeysSubdir)
			block.Headers["gun"] = fileDir[1:]
			if strings.Contains(keyID, "_") {
				role := strings.Split(keyID, "_")[1]
				keyID = strings.TrimSuffix(keyID, "_"+role)
				block.Headers["role"] = role
			}
		}
		var keyPEM bytes.Buffer
		_ = pem.Encode(&keyPEM, block)
		s.Set(keyID, keyPEM.Bytes())
	}
	os.RemoveAll(filepath.Join(s.Location(), notary.RootKeysSubdir))
	os.RemoveAll(filepath.Join(s.Location(), notary.NonRootKeysSubdir))
	os.RemoveAll(filepath.Join(s.Location(), "trusted_certificates"))
}

// NewKeyMemoryStore returns a new KeyMemoryStore which holds keys in memory
func NewKeyMemoryStore(p notary.PassRetriever) *GenericKeyStore {
	memStore := store.NewMemoryStore(nil)
	return NewGenericKeyStore(memStore, p)
}

// NewGenericKeyStore creates a GenericKeyStore wrapping the provided
// Storage instance, using the PassRetriever to enc/decrypt keys
func NewGenericKeyStore(s Storage, p notary.PassRetriever) *GenericKeyStore {
	ks := GenericKeyStore{
		store:         s,
		PassRetriever: p,
		cachedKeys:    make(map[string]*cachedKey),
		keyInfoMap:    make(keyInfoMap),
	}
	ks.loadKeyInfo()
	return &ks
}

func generateKeyInfoMap(s Storage) map[string]KeyInfo {
	keyInfoMap := make(map[string]KeyInfo)
	for _, keyPath := range s.ListFiles() {
		d, err := s.Get(keyPath)
		if err != nil {
			logrus.Error(err)
			continue
		}
		keyID, keyInfo, err := KeyInfoFromPEM(d, keyPath)
		if err != nil {
			logrus.Error(err)
			continue
		}
		keyInfoMap[keyID] = keyInfo
	}
	return keyInfoMap
}

func (s *GenericKeyStore) loadKeyInfo() {
	s.keyInfoMap = generateKeyInfoMap(s.store)
}

// GetKeyInfo returns the corresponding gun and role key info for a keyID
func (s *GenericKeyStore) GetKeyInfo(keyID string) (KeyInfo, error) {
	if info, ok := s.keyInfoMap[keyID]; ok {
		return info, nil
	}
	return KeyInfo{}, fmt.Errorf("Could not find info for keyID %s", keyID)
}

// AddKey stores the contents of a PEM-encoded private key as a PEM block
func (s *GenericKeyStore) AddKey(keyInfo KeyInfo, privKey data.PrivateKey) error {
	var (
		chosenPassphrase string
		giveup           bool
		err              error
		pemPrivKey       []byte
	)
	s.Lock()
	defer s.Unlock()
	if keyInfo.Role == data.CanonicalRootRole || data.IsDelegation(keyInfo.Role) || !data.ValidRole(keyInfo.Role) {
		keyInfo.Gun = ""
	}
	keyID := privKey.ID()
	for attempts := 0; ; attempts++ {
		chosenPassphrase, giveup, err = s.PassRetriever(keyID, keyInfo.Role, true, attempts)
		if err == nil {
			break
		}
		if giveup || attempts > 10 {
			return ErrAttemptsExceeded{}
		}
	}

	if chosenPassphrase != "" {
		pemPrivKey, err = utils.EncryptPrivateKey(privKey, keyInfo.Role, keyInfo.Gun, chosenPassphrase)
	} else {
		pemPrivKey, err = utils.KeyToPEM(privKey, keyInfo.Role, keyInfo.Gun)
	}

	if err != nil {
		return err
	}

	s.cachedKeys[keyID] = &cachedKey{alias: keyInfo.Role, key: privKey}
	err = s.store.Set(keyID, pemPrivKey)
	if err != nil {
		return err
	}
	s.keyInfoMap[privKey.ID()] = keyInfo
	return nil
}

// GetKey returns the PrivateKey given a KeyID
func (s *GenericKeyStore) GetKey(name string) (data.PrivateKey, string, error) {
	s.Lock()
	defer s.Unlock()

	cachedKeyEntry, ok := s.cachedKeys[name]
	if ok {
		return cachedKeyEntry.key, cachedKeyEntry.alias, nil
	}

	role, err := getKeyRole(s.store, name)
	if err != nil {
		return nil, "", err
	}

	keyBytes, err := s.store.Get(name)
	if err != nil {
		return nil, "", err
	}

	// See if the key is encrypted. If its encrypted we'll fail to parse the private key
	privKey, err := utils.ParsePEMPrivateKey(keyBytes, "")
	if err != nil {
		privKey, _, err = GetPasswdDecryptBytes(s.PassRetriever, keyBytes, name, string(role))
		if err != nil {
			return nil, "", err
		}
	}
	s.cachedKeys[name] = &cachedKey{alias: role, key: privKey}
	return privKey, role, nil
}

// ListKeys returns a list of unique PublicKeys present on the KeyFileStore, by returning a copy of the keyInfoMap
func (s *GenericKeyStore) ListKeys() map[string]KeyInfo {
	return copyKeyInfoMap(s.keyInfoMap)
}

// RemoveKey removes the key from the keyfilestore
func (s *GenericKeyStore) RemoveKey(keyID string) error {
	s.Lock()
	defer s.Unlock()
	delete(s.cachedKeys, keyID)

	// being in a subdirectory is for backwards compatibliity
	err := s.store.Remove(keyID)
	if err != nil {
		return err
	}

	delete(s.keyInfoMap, filepath.Base(keyID))
	return nil
}

// Name returns a user friendly name for the location this store
// keeps its data
func (s *GenericKeyStore) Name() string {
	return s.store.Location()
}

// copyKeyInfoMap returns a deep copy of the passed-in keyInfoMap
func copyKeyInfoMap(keyInfoMap map[string]KeyInfo) map[string]KeyInfo {
	copyMap := make(map[string]KeyInfo)
	for keyID, keyInfo := range keyInfoMap {
		copyMap[keyID] = KeyInfo{Role: keyInfo.Role, Gun: keyInfo.Gun}
	}
	return copyMap
}

// KeyInfoFromPEM attempts to get a keyID and KeyInfo from the filename and PEM bytes of a key
func KeyInfoFromPEM(pemBytes []byte, filename string) (string, KeyInfo, error) {
	//keyID, role, gun := inferKeyInfoFromKeyPath(filename)
	var keyID, role, gun string
	keyID = filepath.Base(filename)
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return "", KeyInfo{}, fmt.Errorf("could not decode PEM block for key %s", filename)
	}
	if keyRole, ok := block.Headers["role"]; ok {
		role = keyRole
	}
	if keyGun, ok := block.Headers["gun"]; ok {
		gun = keyGun
	}
	return keyID, KeyInfo{Gun: gun, Role: role}, nil
}

// getKeyRole finds the role for the given keyID. It attempts to look
// both in the newer format PEM headers, and also in the legacy filename
// format. It returns: the role, whether it was found in the legacy(0.1) format
// (true == legacy),  whether it was found in a notary0.3 format (true == notary0.3) and an error
func getKeyRole(s Storage, keyID string) (string, error) {
	name := strings.TrimSpace(strings.TrimSuffix(filepath.Base(keyID), filepath.Ext(keyID)))

	for _, file := range s.ListFiles() {
		filename := filepath.Base(file)
		if strings.HasPrefix(filename, name) {
			d, err := s.Get(file)
			if err != nil {
				return "", err
			}
			block, _ := pem.Decode(d)
			if block != nil {
				return block.Headers["role"], nil
			}
		}
	}
	return "", ErrKeyNotFound{KeyID: keyID}
}

// GetPasswdDecryptBytes gets the password to decrypt the given pem bytes.
// Returns the password and private key
func GetPasswdDecryptBytes(passphraseRetriever notary.PassRetriever, pemBytes []byte, name, alias string) (data.PrivateKey, string, error) {
	var (
		passwd  string
		privKey data.PrivateKey
	)
	for attempts := 0; ; attempts++ {
		var (
			giveup bool
			err    error
		)
		if attempts > 10 {
			return nil, "", ErrAttemptsExceeded{}
		}
		passwd, giveup, err = passphraseRetriever(name, alias, false, attempts)
		// Check if the passphrase retriever got an error or if it is telling us to give up
		if giveup || err != nil {
			return nil, "", ErrPasswordInvalid{}
		}

		// Try to convert PEM encoded bytes back to a PrivateKey using the passphrase
		privKey, err = utils.ParsePEMPrivateKey(pemBytes, passwd)
		if err == nil {
			// We managed to parse the PrivateKey. We've succeeded!
			break
		}
	}
	return privKey, passwd, nil
}
