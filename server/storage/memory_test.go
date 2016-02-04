package storage

import (
	"bytes"
	"testing"
	"time"

	"github.com/docker/notary/tuf/data"
	"github.com/stretchr/testify/assert"
)

func TestUpdateCurrent(t *testing.T) {
	s := NewMemStorage()
	s.UpdateCurrent("gun", MetaUpdate{"role", 1, []byte("test")})

	k := entryKey("gun", "role")
	gun, ok := s.tufMeta[k]
	v := gun[0]
	assert.True(t, ok, "Did not find gun in store")
	assert.Equal(t, 1, v.version, "Version mismatch. Expected 1, found %d", v.version)
	assert.Equal(t, []byte("test"), v.data, "Data was incorrect")
}

func TestGetCurrent(t *testing.T) {
	s := NewMemStorage()

	_, err := s.GetCurrent("gun", "role")
	assert.IsType(t, ErrNotFound{}, err, "Expected error to be ErrNotFound")

	s.UpdateCurrent("gun", MetaUpdate{"role", 1, []byte("test")})
	d, err := s.GetCurrent("gun", "role")
	assert.Nil(t, err, "Expected error to be nil")
	assert.Equal(t, []byte("test"), d, "Data was incorrect")
}

func TestDelete(t *testing.T) {
	s := NewMemStorage()
	s.UpdateCurrent("gun", MetaUpdate{"role", 1, []byte("test")})
	s.Delete("gun")

	k := entryKey("gun", "role")
	_, ok := s.tufMeta[k]
	assert.False(t, ok, "Found gun in store, should have been deleted")
}

// If there are no keys, getting the latest key results in an ErrNoKey
func TestGetNoKeys(t *testing.T) {
	s := NewMemStorage()
	_, err := s.GetLatestKey("gun", data.CanonicalTimestampRole)
	assert.Error(t, err)
	assert.IsType(t, ErrNoKey{}, err)

	// getting a key for a different gun or different role fails too
	k := data.NewPublicKey(data.RSAKey, []byte("test"))
	assert.NoError(t, s.AddKey("gun", data.CanonicalTimestampRole, k, time.Now().AddDate(1, 1, 1)))

	_, err = s.GetLatestKey("gun", data.CanonicalSnapshotRole)
	assert.Error(t, err)
	assert.IsType(t, ErrNoKey{}, err)

	_, err = s.GetLatestKey("othergun", data.CanonicalTimestampRole)
	assert.Error(t, err)
	assert.IsType(t, ErrNoKey{}, err)
}

// GetlatestKey returns the latest key, whether it is active or pending
func TestGetLatest(t *testing.T) {
	s := NewMemStorage()

	k1 := data.NewPublicKey(data.RSAKey, []byte("test1"))
	k2 := data.NewPublicKey(data.RSAKey, []byte("test2"))
	assert.NoError(t, s.AddKey("gun", data.CanonicalTimestampRole, k1, time.Now().AddDate(1, 1, 1)))
	assert.NoError(t, s.AddKey("gun", data.CanonicalTimestampRole, k2, time.Now().AddDate(1, 1, 1)))

	key, err := s.GetLatestKey("gun", data.CanonicalTimestampRole)
	assert.NoError(t, err)
	assert.Equal(t, k2.ID(), key.ID())
	assert.True(t, key.Pending)

	assert.NoError(t, s.MarkActiveKeys("gun", data.CanonicalTimestampRole, []string{k1.ID(), k2.ID()}))

	key, err = s.GetLatestKey("gun", data.CanonicalTimestampRole)
	assert.NoError(t, err)
	assert.Equal(t, k2.ID(), key.ID())
	assert.False(t, key.Pending)
}

// If a key is expired, even if it's the most recently created, GetLatestKey will ignore it.
func TestGetLatestKeyExcludesExpired(t *testing.T) {
	s := NewMemStorage()

	k1 := data.NewPublicKey(data.RSAKey, []byte("test1"))
	k2 := data.NewPublicKey(data.RSAKey, []byte("test2"))
	assert.NoError(t, s.AddKey("gun", data.CanonicalTimestampRole, k1, time.Now().AddDate(1, 1, 1)))
	assert.NoError(t, s.AddKey("gun", data.CanonicalTimestampRole, k2, time.Now().AddDate(-100, -1, -1)))

	key, err := s.GetLatestKey("gun", data.CanonicalTimestampRole)
	assert.NoError(t, err)
	assert.Equal(t, k1.ID(), key.ID())

	s = NewMemStorage()
	assert.NoError(t, s.AddKey("gun", data.CanonicalTimestampRole, k1, time.Now().AddDate(-100, -1, -1)))
	_, err = s.GetLatestKey("gun", data.CanonicalTimestampRole)
	assert.Error(t, err)
	assert.IsType(t, ErrNoKey{}, err)
}

// If there are no non-expired keys which match the given key IDs, HasAnyKeys will return false.
// Otherwise if even a single key exists, returns true
func TestHasAnyKeys(t *testing.T) {
	s := NewMemStorage()
	gun := "gun"
	role := data.CanonicalTimestampRole

	k1 := data.NewPublicKey(data.RSAKey, []byte("test1"))
	k2 := data.NewPublicKey(data.ECDSAKey, []byte("test2"))
	assert.NoError(t, s.AddKey(gun, role, k1, time.Now().AddDate(1, 1, 1)))
	assert.NoError(t, s.AddKey(gun, role, k2, time.Now().AddDate(-100, -1, -1)))

	yesno, err := s.HasAnyKeys(gun, role, []string{"123", "abc"})
	assert.NoError(t, err)
	assert.False(t, yesno)

	yesno, err = s.HasAnyKeys(gun, role, []string{k2.ID()})
	assert.NoError(t, err)
	assert.False(t, yesno)

	yesno, err = s.HasAnyKeys(gun, role, []string{k1.ID()})
	assert.NoError(t, err)
	assert.True(t, yesno)
}

// Adding a key means that we can get it.  We can't add the same key again, though.
// Keys are always added as pending.
func TestAddKeyDuplicate(t *testing.T) {
	s := NewMemStorage()

	key := data.NewPublicKey(data.RSAKey, []byte("test"))
	assert.NoError(t, s.AddKey("gun", data.CanonicalTimestampRole, key, time.Now().AddDate(100, 0, 0)))

	k, err := s.GetLatestKey("gun", data.CanonicalTimestampRole)
	assert.NoError(t, err)
	assert.Equal(t, key.Algorithm(), k.Algorithm(), "Expected algorithm to be rsa")
	assert.True(t, bytes.Equal(key.Public(), k.Public()), "Public key did not match expected")
	assert.True(t, k.Pending)

	err = s.AddKey("gun", data.CanonicalTimestampRole, key, time.Now().AddDate(1, 1, 1))
	assert.Error(t, err)
	existsErr, ok := err.(ErrKeyExists)
	assert.True(t, ok, "Cannot add same key a second time to the same gun and role")
	assert.Equal(t, existsErr.Gun, "gun")
	assert.Equal(t, existsErr.Role, data.CanonicalTimestampRole)
	assert.Equal(t, existsErr.KeyID, key.ID())
}

// We can add multiple keys to multiple roles, and GetLatestKey will get only
// the latest of those keys of the given role.
func TestAddMultipleKeys(t *testing.T) {
	s := NewMemStorage()

	roles := []string{data.CanonicalTimestampRole, data.CanonicalSnapshotRole}
	latestKeys := make(map[string]data.PublicKey)

	for i, keyAlgo := range []string{data.ECDSAKey, data.RSAKey} {
		for _, role := range roles {
			key := data.NewPublicKey(keyAlgo, []byte(keyAlgo))
			assert.NoError(t, s.AddKey("gun", role, key, time.Now().AddDate(1, 1, 1)))
			if i > 0 {
				latestKeys[role] = key
			}
		}
	}

	for _, role := range roles {
		key, err := s.GetLatestKey("gun", role)
		assert.NoError(t, err)
		assert.Equal(t, latestKeys[role].ID(), key.ID())
	}
}

// Marking nonexistent keys as active does not fail - it's just a no-op.
func TestMarkActiveKeysNonexistent(t *testing.T) {
	s := NewMemStorage()
	assert.NoError(t, s.MarkActiveKeys("gun", data.CanonicalSnapshotRole, []string{"1234"}))
}

// The list of keys passed to MarkActiveKeys does not all have to exist.  MarkActiveKeys
// will mark those that exist as active, and ignore the others.
func TestMarkActiveKeys(t *testing.T) {
	s := NewMemStorage()

	key := data.NewPublicKey(data.RSAKey, []byte("test1"))
	roles := []string{data.CanonicalTimestampRole, data.CanonicalSnapshotRole}

	// both roles have the same key
	for _, role := range roles {
		assert.NoError(t, s.AddKey("gun", role, key, time.Now().AddDate(1, 1, 1)))
		k, err := s.GetLatestKey("gun", role)
		assert.NoError(t, err)
		assert.NotNil(t, k)
		assert.True(t, k.Pending)
	}

	// mark only the key for one role active
	assert.NoError(t, s.MarkActiveKeys("gun", data.CanonicalSnapshotRole, []string{key.ID(), "1234"}))

	// only the one marked active should be active
	for _, role := range roles {
		k, err := s.GetLatestKey("gun", role)
		assert.NoError(t, err)
		assert.NotNil(t, k)
		assert.Equal(t, role != data.CanonicalSnapshotRole, k.Pending)
	}
}

func TestGetChecksumNotFound(t *testing.T) {
	s := NewMemStorage()
	_, err := s.GetChecksum("gun", "root", "12345")
	assert.Error(t, err)
	assert.IsType(t, ErrNotFound{}, err)
}
