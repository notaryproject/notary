package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/docker/notary/tuf/data"
	"github.com/stretchr/testify/require"
)

type tufMeta struct {
	Gun     string
	Role    string
	Sha256  string
	Data    []byte
	Version int
}

func SampleCustomTUFObj(role, gun string, tufdata []byte, version int) tufMeta {
	checksum := sha256.Sum256(tufdata)
	hexChecksum := hex.EncodeToString(checksum[:])
	return tufMeta{
		Gun:     gun,
		Role:    role,
		Version: version,
		Sha256:  hexChecksum,
		Data:    tufdata,
	}
}

func SampleCustomUpdate(role string, tufdata []byte, version int) MetaUpdate {
	return MetaUpdate{
		Role:    role,
		Version: version,
		Data:    tufdata,
	}
}

func assertExpectedTUFMetaInStore(t *testing.T, s MetaStore, expected []tufMeta) {
	for _, tufObj := range expected {
		_, tufdata, err := s.GetCurrent(tufObj.Gun, tufObj.Role)
		require.NoError(t, err)
		require.Equal(t, tufObj.Data, tufdata)

		checksumBytes := sha256.Sum256(tufObj.Data)
		checksum := hex.EncodeToString(checksumBytes[:])

		_, tufdata, err = s.GetChecksum(tufObj.Gun, tufObj.Role, checksum)
		require.NoError(t, err)
		require.Equal(t, tufObj.Data, tufdata)
	}
}

// UpdateCurrent should succeed if there was no previous metadata of the same
// gun and role.  They should be gettable.
func testUpdateCurrentEmptyStore(t *testing.T, s MetaStore) []tufMeta {
	expected := make([]tufMeta, 0, 10)
	for _, role := range append(data.BaseRoles, "targets/a") {
		for _, gun := range []string{"gun1", "gun2"} {
			// Adding a new TUF file should succeed
			tufdata := []byte(role + gun)
			require.NoError(t, s.UpdateCurrent(gun, SampleCustomUpdate(role, tufdata, 1)))
			expected = append(expected, SampleCustomTUFObj(role, gun, tufdata, 1))
		}
	}

	assertExpectedTUFMetaInStore(t, s, expected)
	return expected
}

// UpdateCurrent will successfully add a new (higher) version of an existing TUF file,
// but will return an error if there is an older version of a TUF file.
func testUpdateCurrentVersionCheck(t *testing.T, s MetaStore) []tufMeta {
	role, gun, tufdata := data.CanonicalRootRole, "testGUN", []byte("1")

	// starting meta is version 1
	require.NoError(t, s.UpdateCurrent(gun, SampleCustomUpdate(role, tufdata, 1)))

	// inserting meta version immediately above it and skipping ahead will succeed
	require.NoError(t, s.UpdateCurrent(gun, SampleCustomUpdate(role, tufdata, 2)))
	require.NoError(t, s.UpdateCurrent(gun, SampleCustomUpdate(role, tufdata, 4)))

	// Inserting a version that already exists, or that is lower than the current version, will fail
	for _, version := range []int{3, 4} {
		err := s.UpdateCurrent(gun, SampleCustomUpdate(role, tufdata, version))
		require.Error(t, err, "Error should not be nil")
		require.IsType(t, &ErrOldVersion{}, err,
			"Expected ErrOldVersion error type, got: %v", err)
	}

	expected := []tufMeta{
		SampleCustomTUFObj(role, gun, tufdata, 1),
		SampleCustomTUFObj(role, gun, tufdata, 2),
		SampleCustomTUFObj(role, gun, tufdata, 4),
	}
	assertExpectedTUFMetaInStore(t, s, expected)
	return expected
}

// UpdateMany succeeds if the updates do not conflict with each other or with what's
// already in the DB
func testUpdateManyNoConflicts(t *testing.T, s MetaStore) []tufMeta {
	gun, tufdata := "testGUN", []byte("many")
	expected := make([]tufMeta, 4)
	updates := make([]MetaUpdate, 4)
	for i, role := range data.BaseRoles {
		expected[i] = SampleCustomTUFObj(role, gun, tufdata, 1)
		updates[i] = SampleCustomUpdate(role, tufdata, 1)
	}

	require.NoError(t, s.UpdateMany(gun, updates))

	// no conflicts with what's in DB or with itself
	for i, role := range data.BaseRoles {
		expected = append(expected, SampleCustomTUFObj(role, gun, tufdata, 2))
		updates[i] = SampleCustomUpdate(role, tufdata, 2)
	}

	require.NoError(t, s.UpdateMany(gun, updates))

	// and no conflicts if the same role and gun but different version is included
	// in the same update.  Even if they're out of order.
	updates = updates[:2]
	for i, version := range []int{4, 3} {
		role := data.CanonicalRootRole
		expected = append(expected, SampleCustomTUFObj(role, gun, tufdata, version))
		updates[i] = SampleCustomUpdate(role, tufdata, version)
	}

	require.NoError(t, s.UpdateMany(gun, updates))

	assertExpectedTUFMetaInStore(t, s, expected)
	return expected
}

// UpdateMany does not insert any rows (or at least rolls them back) if there
// are any conflicts.
func testUpdateManyConflictRollback(t *testing.T, s MetaStore) []tufMeta {
	gun := "testGUN"
	successBatch := make([]tufMeta, 4)
	updates := make([]MetaUpdate, 4)
	for i, role := range data.BaseRoles {
		tufdata := []byte(gun + "_" + role + "_1")
		successBatch[i] = SampleCustomTUFObj(role, gun, tufdata, 1)
		updates[i] = SampleCustomUpdate(role, tufdata, 1)
	}

	require.NoError(t, s.UpdateMany(gun, updates))

	// conflicts with what's in DB
	badBatch := make([]tufMeta, 4)
	for i, role := range data.BaseRoles {
		version := 2
		if role == data.CanonicalTargetsRole {
			version = 1
		}
		tufdata := []byte(fmt.Sprintf("%s_%s_%d_bad", gun, role, version))
		badBatch[i] = SampleCustomTUFObj(role, gun, tufdata, version)
		updates[i] = SampleCustomUpdate(role, tufdata, version)
	}

	err := s.UpdateMany(gun, updates)
	require.Error(t, err)
	require.IsType(t, &ErrOldVersion{}, err)

	// self-conflicting, in that it's a duplicate, but otherwise no DB conflicts
	duplicate := SampleCustomTUFObj(data.CanonicalTimestampRole, gun, []byte("duplicate"), 3)
	duplicateUpdate := SampleCustomUpdate(duplicate.Role, duplicate.Data, duplicate.Version)
	err = s.UpdateMany(gun, []MetaUpdate{duplicateUpdate, duplicateUpdate})
	require.Error(t, err)
	require.IsType(t, &ErrOldVersion{}, err)

	assertExpectedTUFMetaInStore(t, s, successBatch)

	for _, tufObj := range append(badBatch, duplicate) {
		checksumBytes := sha256.Sum256(tufObj.Data)
		checksum := hex.EncodeToString(checksumBytes[:])

		_, _, err = s.GetChecksum(tufObj.Gun, tufObj.Role, checksum)
		require.Error(t, err)
		require.IsType(t, ErrNotFound{}, err)
	}

	return successBatch
}

// Delete will remove all TUF metadata, all versions, associated with a gun
func testDeleteSuccess(t *testing.T, s MetaStore) {
	tufdata, gun := []byte("hello"), "testGUN"
	// If there is nothing in the DB, delete is a no-op success
	require.NoError(t, s.Delete(gun))

	// If there is data in the DB, all versions are deleted
	unexpected := make([]tufMeta, 0, 10)
	updates := make([]MetaUpdate, 0, 10)
	for _, role := range append(data.BaseRoles, "targets/a") {
		for version := 1; version < 3; version++ {
			unexpected = append(unexpected, SampleCustomTUFObj(role, gun, tufdata, version))
			updates = append(updates, SampleCustomUpdate(role, tufdata, version))
		}
	}
	require.NoError(t, s.UpdateMany(gun, updates))
	require.NoError(t, s.Delete(gun))

	for _, tufObj := range unexpected {
		_, _, err := s.GetCurrent(tufObj.Gun, tufObj.Role)
		require.IsType(t, ErrNotFound{}, err)

		checksumBytes := sha256.Sum256(tufObj.Data)
		checksum := hex.EncodeToString(checksumBytes[:])

		_, _, err = s.GetChecksum(tufObj.Gun, tufObj.Role, checksum)
		require.Error(t, err)
		require.IsType(t, ErrNotFound{}, err)
	}
}
