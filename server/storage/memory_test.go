// +build !mysqldb,!rethinkdb,!postgresqldb

package storage

import (
	"testing"

	"github.com/docker/notary/tuf/data"
	"github.com/stretchr/testify/require"
)

func assertExpectedMemoryTUFMeta(t *testing.T, expected []StoredTUFMeta, s *MemStorage) {
	for _, tufObj := range expected {
		k := entryKey(tufObj.Gun, tufObj.Role)
		versionList, ok := s.tufMeta[k]
		require.True(t, ok, "Did not find this gun+role in store")
		byVersion := make(map[int]ver)
		for _, v := range versionList {
			byVersion[v.version] = v
		}

		v, ok := byVersion[tufObj.Version]
		require.True(t, ok, "Did not find version %d in store", tufObj.Version)
		require.Equal(t, tufObj.Data, v.data, "Data was incorrect")
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
// but will return an error if the to-be-added version already exists in the DB.
func TestMemoryUpdateCurrentVersionCheckOldVersionExists(t *testing.T) {
	s := NewMemStorage()
	expected := testUpdateCurrentVersionCheck(t, s, true)
	assertExpectedMemoryTUFMeta(t, expected, s)
}

// UpdateCurrent will successfully add a new (higher) version of an existing TUF file,
// but will return an error if the to-be-added version does not exist in the DB, but
// is older than an existing version in the DB.
func TestMemoryUpdateCurrentVersionCheckOldVersionNotExist(t *testing.T) {
	s := NewMemStorage()
	expected := testUpdateCurrentVersionCheck(t, s, false)
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

func TestGetChecksumNotFound(t *testing.T) {
	s := NewMemStorage()
	_, _, err := s.GetChecksum("gun", "root", "12345")
	require.Error(t, err)
	require.IsType(t, ErrNotFound{}, err)
}

func TestMemoryGetChanges(t *testing.T) {
	s := NewMemStorage()

	// non-int changeID
	c, err := s.GetChanges("foo", 10, "", false)
	require.Error(t, err)
	require.Len(t, c, 0)

	// add some records
	require.NoError(t, s.UpdateMany("alpine", []MetaUpdate{
		{
			Role:    data.CanonicalTimestampRole,
			Version: 1,
			Data:    []byte{},
		},
		{
			Role:    data.CanonicalTimestampRole,
			Version: 2,
			Data:    []byte{},
		},
		{
			Role:    data.CanonicalTimestampRole,
			Version: 3,
			Data:    []byte{},
		},
		{
			Role:    data.CanonicalTimestampRole,
			Version: 4,
			Data:    []byte{},
		},
	}))
	require.NoError(t, s.UpdateMany("busybox", []MetaUpdate{
		{
			Role:    data.CanonicalTimestampRole,
			Version: 1,
			Data:    []byte{},
		},
		{
			Role:    data.CanonicalTimestampRole,
			Version: 2,
			Data:    []byte{},
		},
		{
			Role:    data.CanonicalTimestampRole,
			Version: 3,
			Data:    []byte{},
		},
		{
			Role:    data.CanonicalTimestampRole,
			Version: 4,
			Data:    []byte{},
		},
	}))

	c, err = s.GetChanges("0", 8, "", false)
	require.NoError(t, err)
	require.Len(t, c, 8)
	for i := 0; i < 4; i++ {
		require.Equal(t, uint(i+1), c[i].ID)
		require.Equal(t, "alpine", c[i].GUN)
		require.Equal(t, i+1, c[i].Version)
	}
	for i := 4; i < 8; i++ {
		require.Equal(t, uint(i+1), c[i].ID)
		require.Equal(t, "busybox", c[i].GUN)
		require.Equal(t, i-3, c[i].Version)
	}

	c, err = s.GetChanges("0", 4, "", false)
	require.NoError(t, err)
	require.Len(t, c, 4)
	for i := 0; i < 4; i++ {
		require.Equal(t, uint(i+1), c[i].ID)
		require.Equal(t, "alpine", c[i].GUN)
		require.Equal(t, i+1, c[i].Version)
	}

	c, err = s.GetChanges("-1", 4, "", false)
	require.NoError(t, err)
	require.Len(t, c, 4)
	for i := 0; i < 4; i++ {
		require.Equal(t, uint(i+5), c[i].ID)
		require.Equal(t, "busybox", c[i].GUN)
		require.Equal(t, i+1, c[i].Version)
	}

	c, err = s.GetChanges("10", 4, "", false)
	require.NoError(t, err)
	require.Len(t, c, 0)

	c, err = s.GetChanges("10", 4, "", true)
	require.NoError(t, err)
	require.Len(t, c, 4)
	for i := 0; i < 4; i++ {
		require.Equal(t, uint(i+5), c[i].ID)
		require.Equal(t, "busybox", c[i].GUN)
		require.Equal(t, i+1, c[i].Version)
	}

	c, err = s.GetChanges("7", 4, "", true)
	require.NoError(t, err)
	require.Len(t, c, 4)
	for i := 0; i < 2; i++ {
		require.Equal(t, uint(i+3), c[i].ID)
		require.Equal(t, "alpine", c[i].GUN)
		require.Equal(t, i+3, c[i].Version)
	}
	for i := 2; i < 4; i++ {
		require.Equal(t, uint(i+3), c[i].ID)
		require.Equal(t, "busybox", c[i].GUN)
		require.Equal(t, i-1, c[i].Version)
	}

	c, err = s.GetChanges("7", 2, "alpine", true)
	require.NoError(t, err)
	require.Len(t, c, 2)
	for i := 0; i < 2; i++ {
		require.Equal(t, uint(i+3), c[i].ID)
		require.Equal(t, "alpine", c[i].GUN)
		require.Equal(t, i+3, c[i].Version)
	}

	c, err = s.GetChanges("0", 8, "busybox", false)
	require.NoError(t, err)
	require.Len(t, c, 4)
	for i := 0; i < 4; i++ {
		require.Equal(t, uint(i+5), c[i].ID)
		require.Equal(t, "busybox", c[i].GUN)
		require.Equal(t, i+1, c[i].Version)
	}
}
