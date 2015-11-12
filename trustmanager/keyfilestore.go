package trustmanager

import (
	"path/filepath"
	"strings"
	"sync"

	"github.com/docker/notary/pkg/passphrase"
	"github.com/docker/notary/tuf/data"
)

const (
	rootKeysSubdir    = "root_keys"
	nonRootKeysSubdir = "tuf_keys"
)

// KeyFileStore persists and manages private keys on disk
type KeyFileStore struct {
	sync.Mutex
	SimpleFileStore
	passphrase.Retriever
	cachedKeys map[string]*cachedKey
}

// KeyMemoryStore manages private keys in memory
type KeyMemoryStore struct {
	sync.Mutex
	MemoryFileStore
	passphrase.Retriever
	cachedKeys map[string]*cachedKey
}

// NewKeyFileStore returns a new KeyFileStore creating a private directory to
// hold the keys.
func NewKeyFileStore(baseDir string, passphraseRetriever passphrase.Retriever) (*KeyFileStore, error) {
	fileStore, err := NewPrivateSimpleFileStore(baseDir, keyExtension)
	if err != nil {
		return nil, err
	}
	cachedKeys := make(map[string]*cachedKey)

	return &KeyFileStore{SimpleFileStore: *fileStore,
		Retriever:  passphraseRetriever,
		cachedKeys: cachedKeys}, nil
}

// AddKey stores the contents of a PEM-encoded private key as a PEM block
func (s *KeyFileStore) AddKey(name, alias string, privKey data.PrivateKey) error {
	s.Lock()
	defer s.Unlock()
	return addKey(s, s.Retriever, s.cachedKeys, name, alias, privKey)
}

// GetKey returns the PrivateKey given a KeyID
func (s *KeyFileStore) GetKey(name string) (data.PrivateKey, string, error) {
	s.Lock()
	defer s.Unlock()
	return getKey(s, s.Retriever, s.cachedKeys, name)
}

// ListKeys returns a list of unique PublicKeys present on the KeyFileStore.
func (s *KeyFileStore) ListKeys() map[string]string {
	return listKeys(s)
}

// RemoveKey removes the key from the keyfilestore
func (s *KeyFileStore) RemoveKey(name string) error {
	s.Lock()
	defer s.Unlock()
	return removeKey(s, s.cachedKeys, name)
}

// ExportKey exportes the encrypted bytes from the keystore and writes it to
// dest.
func (s *KeyFileStore) ExportKey(name string) ([]byte, error) {
	keyBytes, err := getRawKey(s, name)
	if err != nil {
		return nil, err
	}
	return keyBytes, nil
}

// ImportKey imports the private key in the encrypted bytes into the keystore
// with the given key ID and alias.
func (s *KeyFileStore) ImportKey(pemBytes []byte, alias string) error {
	return importKey(s, s.Retriever, s.cachedKeys, alias, pemBytes)
}

// NewKeyMemoryStore returns a new KeyMemoryStore which holds keys in memory
func NewKeyMemoryStore(passphraseRetriever passphrase.Retriever) *KeyMemoryStore {
	memStore := NewMemoryFileStore()
	cachedKeys := make(map[string]*cachedKey)

	return &KeyMemoryStore{MemoryFileStore: *memStore,
		Retriever:  passphraseRetriever,
		cachedKeys: cachedKeys}
}

// AddKey stores the contents of a PEM-encoded private key as a PEM block
func (s *KeyMemoryStore) AddKey(name, alias string, privKey data.PrivateKey) error {
	s.Lock()
	defer s.Unlock()
	return addKey(s, s.Retriever, s.cachedKeys, name, alias, privKey)
}

// GetKey returns the PrivateKey given a KeyID
func (s *KeyMemoryStore) GetKey(name string) (data.PrivateKey, string, error) {
	s.Lock()
	defer s.Unlock()
	return getKey(s, s.Retriever, s.cachedKeys, name)
}

// ListKeys returns a list of unique PublicKeys present on the KeyFileStore.
func (s *KeyMemoryStore) ListKeys() map[string]string {
	return listKeys(s)
}

// RemoveKey removes the key from the keystore
func (s *KeyMemoryStore) RemoveKey(name string) error {
	s.Lock()
	defer s.Unlock()
	return removeKey(s, s.cachedKeys, name)
}

// ExportKey exportes the encrypted bytes from the keystore and writes it to
// dest.
func (s *KeyMemoryStore) ExportKey(name string) ([]byte, error) {
	keyBytes, err := getRawKey(s, name)
	if err != nil {
		return nil, err
	}
	return keyBytes, nil
}

// ImportKey imports the private key in the encrypted bytes into the keystore
// with the given key ID and alias.
func (s *KeyMemoryStore) ImportKey(pemBytes []byte, alias string) error {
	return importKey(s, s.Retriever, s.cachedKeys, alias, pemBytes)
}

func addKey(s LimitedFileStore, passphraseRetriever passphrase.Retriever, cachedKeys map[string]*cachedKey, name, alias string, privKey data.PrivateKey) error {

	var (
		chosenPassphrase string
		giveup           bool
		err              error
	)

	for attempts := 0; ; attempts++ {
		chosenPassphrase, giveup, err = passphraseRetriever(name, alias, true, attempts)
		if err != nil {
			continue
		}
		if giveup {
			return ErrAttemptsExceeded{}
		}
		if attempts > 10 {
			return ErrAttemptsExceeded{}
		}
		break
	}

	return encryptAndAddKey(s, chosenPassphrase, cachedKeys, name, alias, privKey)
}

func getKeyAlias(s LimitedFileStore, name string) (string, error) {
	var keyAlias string
	for _, f := range s.ListFiles() {
		// Remove the .key so .Get succeeds
		keyName := strings.TrimSpace(strings.TrimSuffix(f, filepath.Ext(f)))
		name = strings.TrimSpace(name)

		// Attempts to match the specific file that we want the alias for
		if !strings.Contains(keyName, name) {
			continue
		}

		// Attempt to get the alias from the PEM encoded file.
		keyBytes, err := s.Get(keyName)
		if err == nil {
			keyAlias = GetPemKeyAlias(keyBytes)
		}

		// If we found an alias return
		if keyAlias != "" {
			return keyAlias, nil
		}

		// Given a full path to a key, take the filename, remove the extension
		// and attempt to split into keyID and alias
		// Example: /tmp/abcd_root.key, should end up in keyComponents as ['abcd','root']
		keyComponents := strings.Split(filepath.Base(keyName), "_")
		// If we can't get the alias from the PEM and can't split into components,
		// this has to be one of the old root keys
		if len(keyComponents) != 2 {
			return data.CanonicalRootRole, nil
		}

		return keyComponents[1], nil
	}

	return "", &ErrKeyNotFound{KeyID: name}
}

// GetKey returns the PrivateKey given a KeyID
func getKey(s LimitedFileStore, passphraseRetriever passphrase.Retriever, cachedKeys map[string]*cachedKey, name string) (data.PrivateKey, string, error) {
	cachedKeyEntry, ok := cachedKeys[name]
	if ok {
		return cachedKeyEntry.key, cachedKeyEntry.alias, nil
	}
	var (
		err      error
		keyAlias string
	)

	keyBytes, err := getRawKey(s, name)
	if err != nil {
		return nil, "", err
	}

	// See if the key is encrypted. If its encrypted we'll fail to parse the private key
	privKey, keyAlias, err := ParsePEMPrivateKey(keyBytes, "")
	if err != nil {
		privKey, _, err = getPasswdDecryptBytes(s, passphraseRetriever, keyBytes, name, keyAlias)
		if err != nil {
			return nil, "", err
		}
	}
	cachedKeys[name] = &cachedKey{alias: keyAlias, key: privKey}
	return privKey, keyAlias, nil
}

// ListKeys returns a map of unique PublicKeys present on the KeyFileStore and
// their corresponding aliases.
func listKeys(s LimitedFileStore) map[string]string {
	keyIDMap := make(map[string]string)
	var (
		keyName  string
		keyAlias string
		keyID    string
	)
	for _, f := range s.ListFiles() {
		// Remove the .key for GET
		keyName = strings.TrimSuffix(f, filepath.Ext(f))

		// Attempt to get the alias from the PEM encoded file.
		keyBytes, err := s.Get(keyName)
		if err == nil {
			keyAlias = GetPemKeyAlias(keyBytes)
		}

		keyID = filepath.Base(keyName)

		if keyAlias == "" {
			// Given a full path to a key, take the filename, remove the extension
			// and attempt to split into keyID and alias
			// Example: /tmp/abcd_root.key, should end up in keyComponents as ['abcd','root']
			keyComponents := strings.Split(filepath.Base(keyName), "_")
			if len(keyComponents) == 2 {
				keyID = keyComponents[0]
				keyAlias = keyComponents[1]
			} else {
				// If we can't get the alias from the PEM and can't split into components,
				// this has to be one of the old root keys
				keyAlias = data.CanonicalRootRole
			}
		}

		keyIDMap[keyID] = keyAlias
	}

	return keyIDMap
}

// RemoveKey removes the key from the keyfilestore
func removeKey(s LimitedFileStore, cachedKeys map[string]*cachedKey, name string) error {
	// Attempt to delete key from both the new and old style of key names
	keyAlias, err := getKeyAlias(s, strings.TrimSpace(name))
	if err != nil {
		return err
	}

	delete(cachedKeys, name)

	// Attempt to delete the key with the new format (no _alias)
	err = s.Remove(filepath.Join(getSubdir(keyAlias), name))
	if err == nil {
		return nil
	}

	// If we didn't return, let's attempt to delete the key in the old format (with _alias)
	err = s.Remove(filepath.Join(getSubdir(keyAlias), name+"_"+keyAlias))
	if err == nil {
		return nil
	}

	return err
}

// Assumes 2 subdirectories, 1 containing root keys and 1 containing tuf keys
func getSubdir(alias string) string {
	if alias == "root" {
		return rootKeysSubdir
	}
	return nonRootKeysSubdir
}

// Given a key ID, gets the bytes and alias belonging to that key if the key
// exists
func getRawKey(s LimitedFileStore, name string) ([]byte, error) {
	// Attempt to retrieve key from both the new and the old style of keys
	keyAlias, err := getKeyAlias(s, strings.TrimSpace(name))
	if err != nil {
		return nil, err
	}

	// Attempt to get the file with the new format (no _alias)
	keyBytes, err := s.Get(filepath.Join(getSubdir(keyAlias), name))
	if err == nil {
		return keyBytes, nil
	}

	// If we didn't return, let's attempt to get the key in the old format (with _alias)
	keyBytes, err = s.Get(filepath.Join(getSubdir(keyAlias), name+"_"+keyAlias))
	if err == nil {
		return keyBytes, nil
	}

	return nil, err
}

// Get the password to decript the given pem bytes.  Return the password,
// because it is useful for importing
func getPasswdDecryptBytes(s LimitedFileStore, passphraseRetriever passphrase.Retriever, pemBytes []byte, name, alias string) (data.PrivateKey, string, error) {
	var (
		passwd  string
		retErr  error
		privKey data.PrivateKey
	)

	for attempts := 0; ; attempts++ {
		var (
			giveup bool
			err    error
		)
		passwd, giveup, err = passphraseRetriever(name, alias, false, attempts)
		// Check if the passphrase retriever got an error or if it is telling us to give up
		if giveup || err != nil {
			return nil, "", ErrPasswordInvalid{}
		}
		if attempts > 10 {
			return nil, "", ErrAttemptsExceeded{}
		}

		// Try to convert PEM encoded bytes back to a PrivateKey using the passphrase
		privKey, alias, err = ParsePEMPrivateKey(pemBytes, passwd)
		if err != nil {
			retErr = ErrPasswordInvalid{}
		} else {
			// We managed to parse the PrivateKey. We've succeeded!
			retErr = nil
			break
		}
	}
	if retErr != nil {
		return nil, "", retErr
	}
	return privKey, passwd, nil
}

func encryptAndAddKey(s LimitedFileStore, passwd string, cachedKeys map[string]*cachedKey, name, alias string, privKey data.PrivateKey) error {

	var (
		pemPrivKey []byte
		err        error
	)

	if passwd != "" {
		pemPrivKey, err = EncryptPrivateKey(privKey, passwd, alias)
	} else {
		pemPrivKey, err = KeyToPEM(privKey, alias)
	}
	if err != nil {
		return err
	}

	cachedKeys[name] = &cachedKey{alias: alias, key: privKey}
	return s.Add(filepath.Join(getSubdir(alias), name), pemPrivKey)
}

func importKey(s LimitedFileStore, passphraseRetriever passphrase.Retriever, cachedKeys map[string]*cachedKey, alias string, pemBytes []byte) error {

	privKey, passphrase, err := getPasswdDecryptBytes(s, passphraseRetriever, pemBytes, "imported", alias)

	if err != nil {
		return err
	}

	return encryptAndAddKey(
		s, passphrase, cachedKeys, privKey.ID(), alias, privKey)
}
