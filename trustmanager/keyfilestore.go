package trustmanager

import (
	"path/filepath"
	"strings"
	"sync"

	"errors"
	"fmt"

	"github.com/docker/notary/pkg/passphrase"
	"github.com/endophage/gotuf/data"
)

const (
	keyExtension = "key"
)

// KeyStore is a generic interface for private key storage
type KeyStore interface {
	LimitedFileStore

	AddKey(name, alias string, privKey data.PrivateKey) error
	GetKey(name string) (data.PrivateKey, string, error)
	ListKeys() []string
	RemoveKey(name string) error
}

type cachedKey struct {
	alias string
	key   data.PrivateKey
}

// PassphraseRetriever is a callback function that should retrieve a passphrase
// for a given named key. If it should be treated as new passphrase (e.g. with
// confirmation), createNew will be true. Attempts is passed in so that implementers
// decide how many chances to give to a human, for example.
type PassphraseRetriever func(keyId, alias string, createNew bool, attempts int) (passphrase string, giveup bool, err error)

// KeyFileStore persists and manages private keys on disk
type KeyFileStore struct {
	sync.Mutex
	SimpleFileStore
	PassphraseRetriever
	cachedKeys map[string]*cachedKey
}

// KeyMemoryStore manages private keys in memory
type KeyMemoryStore struct {
	sync.Mutex
	MemoryFileStore
	PassphraseRetriever
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
		PassphraseRetriever: passphraseRetriever,
		cachedKeys:          cachedKeys}, nil
}

// AddKey stores the contents of a PEM-encoded private key as a PEM block
func (s *KeyFileStore) AddKey(name, alias string, privKey data.PrivateKey) error {
	s.Lock()
	defer s.Unlock()
	return addKey(s, s.PassphraseRetriever, s.cachedKeys, name, alias, privKey)
}

// GetKey returns the PrivateKey given a KeyID
func (s *KeyFileStore) GetKey(name string) (data.PrivateKey, string, error) {
	s.Lock()
	defer s.Unlock()
	return getKey(s, s.PassphraseRetriever, s.cachedKeys, name)
}

// ListKeys returns a list of unique PublicKeys present on the KeyFileStore.
// There might be symlinks associating Certificate IDs to Public Keys, so this
// method only returns the IDs that aren't symlinks
func (s *KeyFileStore) ListKeys() []string {
	return listKeys(s)
}

// RemoveKey removes the key from the keyfilestore
func (s *KeyFileStore) RemoveKey(name string) error {
	s.Lock()
	defer s.Unlock()
	return removeKey(s, s.cachedKeys, name)
}

// NewKeyMemoryStore returns a new KeyMemoryStore which holds keys in memory
func NewKeyMemoryStore(passphraseRetriever passphrase.Retriever) *KeyMemoryStore {
	memStore := NewMemoryFileStore()
	cachedKeys := make(map[string]*cachedKey)

	return &KeyMemoryStore{MemoryFileStore: *memStore,
		PassphraseRetriever: passphraseRetriever,
		cachedKeys:          cachedKeys}
}

// AddKey stores the contents of a PEM-encoded private key as a PEM block
func (s *KeyMemoryStore) AddKey(name, alias string, privKey data.PrivateKey) error {
	s.Lock()
	defer s.Unlock()
	return addKey(s, s.PassphraseRetriever, s.cachedKeys, name, alias, privKey)
}

// GetKey returns the PrivateKey given a KeyID
func (s *KeyMemoryStore) GetKey(name string) (data.PrivateKey, string, error) {
	s.Lock()
	defer s.Unlock()
	return getKey(s, s.PassphraseRetriever, s.cachedKeys, name)
}

// ListKeys returns a list of unique PublicKeys present on the KeyFileStore.
// There might be symlinks associating Certificate IDs to Public Keys, so this
// method only returns the IDs that aren't symlinks
func (s *KeyMemoryStore) ListKeys() []string {
	return listKeys(s)
}

// RemoveKey removes the key from the keystore
func (s *KeyMemoryStore) RemoveKey(name string) error {
	s.Lock()
	defer s.Unlock()
	return removeKey(s, s.cachedKeys, name)
}

func addKey(s LimitedFileStore, passphraseRetriever PassphraseRetriever, cachedKeys map[string]*cachedKey, name, alias string, privKey data.PrivateKey) error {
	pemPrivKey, err := KeyToPEM(privKey)
	if err != nil {
		return err
	}

	attempts := 0
	passphrase := ""
	giveup := false
	for {
		passphrase, giveup, err = passphraseRetriever(name, alias, true, attempts)
		if err != nil {
			attempts++
			continue
		}
		if giveup {
			return errors.New("obtaining passphrase failed")
		}
		if attempts > 10 {
			return errors.New("maximum number of passphrase attempts exceeded")
		}
		break
	}

	if passphrase != "" {
		pemPrivKey, err = EncryptPrivateKey(privKey, passphrase)
		if err != nil {
			return err
		}
	}

	cachedKeys[name] = &cachedKey{alias: alias, key: privKey}
	return s.Add(name+"_"+alias, pemPrivKey)
}

func getKeyAlias(s LimitedFileStore, keyID string) (string, error) {
	files := s.ListFiles(true)
	name := strings.TrimSpace(strings.TrimSuffix(filepath.Base(keyID), filepath.Ext(keyID)))

	for _, file := range files {
		filename := filepath.Base(file)

		if strings.HasPrefix(filename, name) {
			aliasPlusDotKey := strings.TrimPrefix(filename, name+"_")
			retVal := strings.TrimSuffix(aliasPlusDotKey, "."+keyExtension)
			return retVal, nil
		}
	}

	return "", fmt.Errorf("keyId %s has no alias", name)
}

// GetKey returns the PrivateKey given a KeyID
func getKey(s LimitedFileStore, passphraseRetriever PassphraseRetriever, cachedKeys map[string]*cachedKey, name string) (data.PrivateKey, string, error) {
	cachedKeyEntry, ok := cachedKeys[name]
	if ok {
		return cachedKeyEntry.key, cachedKeyEntry.alias, nil
	}
	keyAlias, err := getKeyAlias(s, name)
	if err != nil {
		return nil, "", err
	}

	keyBytes, err := s.Get(name + "_" + keyAlias)
	if err != nil {
		return nil, "", err
	}

	// See if the key is encrypted. If its encrypted we'll fail to parse the private key
	privKey, err := ParsePEMPrivateKey(keyBytes, "")
	if err != nil {
		// We need to decrypt the key, lets get a passphrase
		for attempts := 0; ; attempts++ {
			passphrase, giveup, err := passphraseRetriever(name, string(keyAlias), false, attempts)
			// Check if the passphrase retriever got an error or if it is telling us to give up
			if giveup || err != nil {
				return nil, "", errors.New("obtaining passphrase failed")
			}
			if attempts > 10 {
				return nil, "", errors.New("maximum number of passphrase attempts exceeded")
			}

			// Try to convert PEM encoded bytes back to a PrivateKey using the passphrase
			privKey, err = ParsePEMPrivateKey(keyBytes, passphrase)
			if err == nil {
				// We managed to parse the PrivateKey. We've succeeded!
				break
			}
		}
	}
	cachedKeys[name] = &cachedKey{alias: keyAlias, key: privKey}
	return privKey, keyAlias, nil
}

// ListKeys returns a list of unique PublicKeys present on the KeyFileStore.
// There might be symlinks associating Certificate IDs to Public Keys, so this
// method only returns the IDs that aren't symlinks
func listKeys(s LimitedFileStore) []string {
	var keyIDList []string

	for _, f := range s.ListFiles(false) {
		keyID := strings.TrimSpace(strings.TrimSuffix(f, filepath.Ext(f)))
		keyID = keyID[:strings.LastIndex(keyID, "_")]
		keyIDList = append(keyIDList, keyID)
	}
	return keyIDList
}

// RemoveKey removes the key from the keyfilestore
func removeKey(s LimitedFileStore, cachedKeys map[string]*cachedKey, name string) error {
	keyAlias, err := getKeyAlias(s, name)
	if err != nil {
		return err
	}

	delete(cachedKeys, name)

	return s.Remove(name + "_" + keyAlias)
}
