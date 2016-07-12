// +build !mysqldb,!rethinkdb

package storage

import (
	"testing"

	"github.com/docker/notary/tuf/data"
	"github.com/stretchr/testify/require"
)

func assertExpectedMemoryTUFMeta(t *testing.T, expected []tufMeta, s *MemStorage) {
	counter := make(map[string]int)
	for _, tufObj := range expected {
		k := entryKey(tufObj.Gun, tufObj.Role)
		gun, ok := s.tufMeta[k]
		require.True(t, len(gun) >= counter[k])
		v := gun[counter[k]]
		require.True(t, ok, "Did not find gun in store")
		require.Equal(t, tufObj.Version, v.version, "Version mismatch. Expected %d, found %d",
			tufObj.Version, v.version)
		require.Equal(t, tufObj.Data, v.data, "Data was incorrect")
		counter[k]++
	}
}

// UpdateCurrent should succeed if there was no previous metadata of the same
// gun and role.  They should be gettable.
func TestMemoryUpdateCurrentEmpty(t *testing.T) {
	s := NewMemStorage()
	expected := testUpdateCurrentEmptyStore(t, s)
	assertExpectedMemoryTUFMeta(t, expected, s)
}

// UpdateCurrent will successfully add a new (higher) version of an existing TUF file,
// but will return an error if there is an older version of a TUF file.
func TestMemoryUpdateCurrentVersionCheck(t *testing.T) {
	s := NewMemStorage()
	expected := testUpdateCurrentVersionCheck(t, s)
	assertExpectedMemoryTUFMeta(t, expected, s)
}

// UpdateMany succeeds if the updates do not conflict with each other or with what's
// already in the DB
func TestMemoryUpdateManyNoConflicts(t *testing.T) {
	s := NewMemStorage()
	expected := testUpdateManyNoConflicts(t, s)
	assertExpectedMemoryTUFMeta(t, expected, s)
}

// UpdateMany does not insert any rows (or at least rolls them back) if there
// are any conflicts.
func TestMemoryUpdateManyConflictRollback(t *testing.T) {
	s := NewMemStorage()
	expected := testUpdateManyConflictRollback(t, s)
	assertExpectedMemoryTUFMeta(t, expected, s)
}

// Delete will remove all TUF metadata, all versions, associated with a gun
func TestMemoryDeleteSuccess(t *testing.T) {
	s := NewMemStorage()
	testDeleteSuccess(t, s)
	assertExpectedMemoryTUFMeta(t, nil, s)
}

func TestGetCurrent(t *testing.T) {
	s := NewMemStorage()

	_, _, err := s.GetCurrent("gun", "role")
	require.IsType(t, ErrNotFound{}, err, "Expected error to be ErrNotFound")

	s.UpdateCurrent("gun", MetaUpdate{"role", 1, []byte("test")})
	_, d, err := s.GetCurrent("gun", "role")
	require.Nil(t, err, "Expected error to be nil")
	require.Equal(t, []byte("test"), d, "Data was incorrect")
}

func TestGetTimestampKey(t *testing.T) {
	s := NewMemStorage()

	s.SetKey("gun", data.CanonicalTimestampRole, data.RSAKey, []byte("test"))

	c, k, err := s.GetKey("gun", data.CanonicalTimestampRole)
	require.Nil(t, err, "Expected error to be nil")
	require.Equal(t, data.RSAKey, c, "Expected algorithm rsa, received %s", c)
	require.Equal(t, []byte("test"), k, "Key data was wrong")
}

func TestSetKey(t *testing.T) {
	s := NewMemStorage()
	err := s.SetKey("gun", data.CanonicalTimestampRole, data.RSAKey, []byte("test"))
	require.NoError(t, err)

	k := s.keys["gun"][data.CanonicalTimestampRole]
	require.Equal(t, data.RSAKey, k.algorithm, "Expected algorithm to be rsa, received %s", k.algorithm)
	require.Equal(t, []byte("test"), k.public, "Public key did not match expected")

}

func TestSetKeyMultipleRoles(t *testing.T) {
	s := NewMemStorage()
	err := s.SetKey("gun", data.CanonicalTimestampRole, data.RSAKey, []byte("test"))
	require.NoError(t, err)

	err = s.SetKey("gun", data.CanonicalSnapshotRole, data.RSAKey, []byte("test"))
	require.NoError(t, err)

	k := s.keys["gun"][data.CanonicalTimestampRole]
	require.Equal(t, data.RSAKey, k.algorithm, "Expected algorithm to be rsa, received %s", k.algorithm)
	require.Equal(t, []byte("test"), k.public, "Public key did not match expected")

	k = s.keys["gun"][data.CanonicalSnapshotRole]
	require.Equal(t, data.RSAKey, k.algorithm, "Expected algorithm to be rsa, received %s", k.algorithm)
	require.Equal(t, []byte("test"), k.public, "Public key did not match expected")
}

func TestSetKeySameRoleGun(t *testing.T) {
	s := NewMemStorage()
	err := s.SetKey("gun", data.CanonicalTimestampRole, data.RSAKey, []byte("test"))
	require.NoError(t, err)

	// set diff algo and bytes so we can confirm data didn't get replaced
	err = s.SetKey("gun", data.CanonicalTimestampRole, data.ECDSAKey, []byte("test2"))
	require.IsType(t, &ErrKeyExists{}, err, "Expected err to be ErrKeyExists")

	k := s.keys["gun"][data.CanonicalTimestampRole]
	require.Equal(t, data.RSAKey, k.algorithm, "Expected algorithm to be rsa, received %s", k.algorithm)
	require.Equal(t, []byte("test"), k.public, "Public key did not match expected")

}

func TestGetChecksumNotFound(t *testing.T) {
	s := NewMemStorage()
	_, _, err := s.GetChecksum("gun", "root", "12345")
	require.Error(t, err)
	require.IsType(t, ErrNotFound{}, err)
}
