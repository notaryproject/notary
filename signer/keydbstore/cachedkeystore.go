package keydbstore

import (
	"sync"

	"github.com/docker/notary/trustmanager"
	"github.com/docker/notary/tuf/data"
)

// Note:  once trustmanager's file KeyStore has been flattened, this can be moved to trustmanager

type cachedKeyStore struct {
	trustmanager.KeyStore
	lock       *sync.Mutex
	cachedKeys map[string]*cachedKey
}

type cachedKey struct {
	role string
	key  data.PrivateKey
}

// NewCachedKeyStore returns a new trustmanager.KeyStore that includes caching
func NewCachedKeyStore(baseStore trustmanager.KeyStore) trustmanager.KeyStore {
	return &cachedKeyStore{
		KeyStore:   baseStore,
		lock:       &sync.Mutex{},
		cachedKeys: make(map[string]*cachedKey),
	}
}

// AddKey stores the contents of a private key. Both role and gun are ignored,
// we always use Key IDs as name, and don't support aliases
func (s *cachedKeyStore) AddKey(keyInfo trustmanager.KeyInfo, privKey data.PrivateKey) error {
	if err := s.KeyStore.AddKey(keyInfo, privKey); err != nil {
		return err
	}

	// Add the private key to our cache
	s.lock.Lock()
	defer s.lock.Unlock()
	s.cachedKeys[privKey.ID()] = &cachedKey{
		role: keyInfo.Role,
		key:  privKey,
	}

	return nil
}

// GetKey returns the PrivateKey given a KeyID
func (s *cachedKeyStore) GetKey(keyID string) (data.PrivateKey, string, error) {
	cachedKeyEntry, ok := s.cachedKeys[keyID]
	if ok {
		return cachedKeyEntry.key, cachedKeyEntry.role, nil
	}

	// retrieve the key from the underlying store and put it into the cache
	privKey, role, err := s.KeyStore.GetKey(keyID)
	if err == nil {
		s.lock.Lock()
		defer s.lock.Unlock()
		// Add the key to cache
		s.cachedKeys[privKey.ID()] = &cachedKey{key: privKey, role: role}
		return privKey, role, nil
	}
	return nil, "", err
}

// RemoveKey removes the key from the keyfilestore
func (s *cachedKeyStore) RemoveKey(keyID string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	delete(s.cachedKeys, keyID)
	return s.KeyStore.RemoveKey(keyID)
}
