package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/docker/notary/tuf/data"
)

type ver struct {
	version int
	data    []byte
}

type storedKey struct {
	ManagedPublicKey
	expires   *time.Time
	createdAt time.Time
}

// MemStorage is really just designed for dev and testing. It is very
// inefficient in many scenarios
type MemStorage struct {
	lock      sync.Mutex
	tufMeta   map[string][]*ver
	keys      map[string]map[string]storedKey
	checksums map[string]map[string][]byte
}

// NewMemStorage instantiates a memStorage instance
func NewMemStorage() *MemStorage {
	return &MemStorage{
		tufMeta:   make(map[string][]*ver),
		keys:      make(map[string]map[string]storedKey),
		checksums: make(map[string]map[string][]byte),
	}
}

// UpdateCurrent updates the meta data for a specific role
func (st *MemStorage) UpdateCurrent(gun string, update MetaUpdate) error {
	id := entryKey(gun, update.Role)
	st.lock.Lock()
	defer st.lock.Unlock()
	if space, ok := st.tufMeta[id]; ok {
		for _, v := range space {
			if v.version >= update.Version {
				return &ErrOldVersion{}
			}
		}
	}
	st.tufMeta[id] = append(st.tufMeta[id], &ver{version: update.Version, data: update.Data})
	checksumBytes := sha256.Sum256(update.Data)
	checksum := hex.EncodeToString(checksumBytes[:])

	_, ok := st.checksums[gun]
	if !ok {
		st.checksums[gun] = make(map[string][]byte)
	}
	st.checksums[gun][checksum] = update.Data
	return nil
}

// UpdateMany updates multiple TUF records
func (st *MemStorage) UpdateMany(gun string, updates []MetaUpdate) error {
	for _, u := range updates {
		st.UpdateCurrent(gun, u)
	}
	return nil
}

// GetCurrent returns the metadata for a given role, under a GUN
func (st *MemStorage) GetCurrent(gun, role string) (data []byte, err error) {
	id := entryKey(gun, role)
	st.lock.Lock()
	defer st.lock.Unlock()
	space, ok := st.tufMeta[id]
	if !ok || len(space) == 0 {
		return nil, ErrNotFound{}
	}
	return space[len(space)-1].data, nil
}

// GetChecksum returns the metadata for a given role, under a GUN
func (st *MemStorage) GetChecksum(gun, role, checksum string) (data []byte, err error) {
	st.lock.Lock()
	defer st.lock.Unlock()
	data, ok := st.checksums[gun][checksum]
	if !ok || len(data) == 0 {
		return nil, ErrNotFound{}
	}
	return data, nil
}

// Delete deletes all the metadata for a given GUN
func (st *MemStorage) Delete(gun string) error {
	st.lock.Lock()
	defer st.lock.Unlock()
	for k := range st.tufMeta {
		if strings.HasPrefix(k, gun) {
			delete(st.tufMeta, k)
		}
	}
	delete(st.checksums, gun)
	return nil
}

// GetLatestKey returns the most recently created non-expired public key for a
// given gun for a given role - we do not do any key cleanup in this function
func (st *MemStorage) GetLatestKey(gun, role string) (*ManagedPublicKey, error) {
	// no need for lock. It's ok to return nil if an update
	// wasn't observed
	id := entryKey(gun, role)
	err := ErrNoKey{Gun: gun, Role: role}

	keys, ok := st.keys[id]
	if !ok {
		return nil, err
	}
	if len(keys) > 0 {
		var (
			latest storedKey
			found  bool
		)
		now := time.Now().Add(1 * time.Minute) // add a buffer so it doesn't expire immediately
		for _, k := range keys {
			if k.expires == nil || k.expires.After(now) {
				if k.createdAt.After(latest.createdAt) {
					latest = k
					found = true
				}
			}
		}
		if found {
			return &ManagedPublicKey{PublicKey: latest.PublicKey, Pending: latest.Pending}, nil
		}
	}
	return nil, err
}

// HasAnyKeys returns true if any non-expired keys exist for the given GUN, role, and key IDs.
func (st *MemStorage) HasAnyKeys(gun, role string, keyIDs []string) (bool, error) {
	// no need for lock. It's ok to return nil if an update
	// wasn't observed
	id := entryKey(gun, role)
	keys, ok := st.keys[id]
	if !ok {
		return false, nil
	}

	now := time.Now().Add(1 * time.Minute) // add a buffer

	for _, keyID := range keyIDs {
		key, ok := keys[keyID]
		if ok && key.expires.After(now) {
			return true, nil
		}
	}

	return false, nil
}

// AddKey sets a key under a gun and role - this optionally cleans up expired keys
// for this gun and role
func (st *MemStorage) AddKey(gun, role string, key data.PublicKey, expires time.Time) error {
	st.lock.Lock()
	defer st.lock.Unlock()

	// we hold the lock so nothing will be able to race to write a key
	// between checking and setting
	id := entryKey(gun, role)
	allKeys, ok := st.keys[id]
	if !ok {
		st.keys[id] = make(map[string]storedKey)
		allKeys = st.keys[id]
	}
	if _, ok := allKeys[key.ID()]; ok {
		return ErrKeyExists{Gun: gun, Role: role, KeyID: key.ID()}
	}

	allKeys[key.ID()] = storedKey{
		ManagedPublicKey: ManagedPublicKey{PublicKey: key, Pending: true},
		createdAt:        time.Now(),
		expires:          &expires,
	}
	return nil
}

// MarkActiveKeys marks the following key IDs as active.
// This does not fail if any of the key IDs doesn't exist.
func (st *MemStorage) MarkActiveKeys(gun, role string, keyIDs []string) error {
	st.lock.Lock()
	defer st.lock.Unlock()

	id := entryKey(gun, role)
	keymap, ok := st.keys[id]
	if ok {
		for _, keyID := range keyIDs {
			key, ok := keymap[keyID]
			if ok {
				keymap[keyID] = storedKey{
					ManagedPublicKey: ManagedPublicKey{PublicKey: key.PublicKey, Pending: false},
					createdAt:        key.createdAt,
					expires:          nil,
				}
			}
		}
	}
	return nil
}

func entryKey(gun, role string) string {
	return fmt.Sprintf("%s.%s", gun, role)
}
