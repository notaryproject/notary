// +build !rethinkdb

package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"github.com/docker/notary/tuf/data"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/require"
)

func SetupSQLDB(t *testing.T, dbtype, dburl string) *SQLStorage {
	dbStore, err := NewSQLStorage(dbtype, dburl)
	require.NoError(t, err)

	// Create the DB tables
	err = CreateTUFTable(dbStore.DB)
	require.NoError(t, err)

	err = CreateKeyTable(dbStore.DB)
	require.NoError(t, err)

	// verify that the tables are empty
	var count int
	for _, model := range [2]interface{}{&TUFFile{}, &Key{}} {
		query := dbStore.DB.Model(model).Count(&count)
		require.NoError(t, query.Error)
		require.Equal(t, 0, count)
	}
	return dbStore
}

type sqldbSetupFunc func(*testing.T) (*SQLStorage, func())

var sqldbSetup sqldbSetupFunc

func assertExpectedGormTUFMeta(t *testing.T, expected []StoredTUFMeta, gormDB gorm.DB) {
	expectedGorm := make([]TUFFile, len(expected))
	for i, tufObj := range expected {
		expectedGorm[i] = TUFFile{
			Model:   gorm.Model{ID: uint(i + 1)},
			Gun:     tufObj.Gun,
			Role:    tufObj.Role,
			Version: tufObj.Version,
			Sha256:  tufObj.Sha256,
			Data:    tufObj.Data,
		}
	}

	// There should just be one row
	var rows []TUFFile
	query := gormDB.Select("id, gun, role, version, sha256, data").Find(&rows)
	require.NoError(t, query.Error)
	// to avoid issues with nil vs zero len list
	if len(expectedGorm) == 0 {
		require.Len(t, rows, 0)
	} else {
		require.Equal(t, expectedGorm, rows)
	}
}

// TestSQLUpdateCurrent asserts that UpdateCurrent will add a new TUF file
// if no previous version of that gun and role existed.
func TestSQLUpdateCurrentEmpty(t *testing.T) {
	dbStore, cleanup := sqldbSetup(t)
	defer cleanup()

	expected := testUpdateCurrentEmptyStore(t, dbStore)
	assertExpectedGormTUFMeta(t, expected, dbStore.DB)

	dbStore.DB.Close()
}

// TestSQLUpdateCurrentVersionCheckOldVersionExists asserts that UpdateCurrent will add a
// new (higher) version of an existing TUF file, and that an error is raised if
// trying to update to an older version of a TUF file that already exists.
func TestSQLUpdateCurrentVersionCheckOldVersionExists(t *testing.T) {
	dbStore, cleanup := sqldbSetup(t)
	defer cleanup()

	expected := testUpdateCurrentVersionCheck(t, dbStore, true)
	assertExpectedGormTUFMeta(t, expected, dbStore.DB)

	dbStore.DB.Close()
}

// TestSQLUpdateCurrentVersionCheckOldVersionNotExist asserts that UpdateCurrent will add a
// new (higher) version of an existing TUF file, and that an error is raised if
// trying to update to an older version of a TUF file that doesn't exist in the DB.
func TestSQLUpdateCurrentVersionCheckOldVersionNotExist(t *testing.T) {
	dbStore, cleanup := sqldbSetup(t)
	defer cleanup()

	expected := testUpdateCurrentVersionCheck(t, dbStore, false)
	assertExpectedGormTUFMeta(t, expected, dbStore.DB)

	dbStore.DB.Close()
}

// TestSQLUpdateManyNoConflicts asserts that inserting multiple updates succeeds if the
// updates do not conflict with each other or with the DB, even if there are
// 2 versions of the same role/gun in a non-monotonic order.
func TestSQLUpdateManyNoConflicts(t *testing.T) {
	dbStore, cleanup := sqldbSetup(t)
	defer cleanup()

	expected := testUpdateManyNoConflicts(t, dbStore)
	assertExpectedGormTUFMeta(t, expected, dbStore.DB)

	dbStore.DB.Close()
}

// TestSQLUpdateManyConflictRollback asserts that no data ends up in the DB if there is
// a conflict
func TestSQLUpdateManyConflictRollback(t *testing.T) {
	dbStore, cleanup := sqldbSetup(t)
	defer cleanup()

	expected := testUpdateManyConflictRollback(t, dbStore)
	assertExpectedGormTUFMeta(t, expected, dbStore.DB)

	dbStore.DB.Close()
}

// TestSQLDelete asserts that Delete will remove all TUF metadata, all versions,
// associated with a gun
func TestSQLDelete(t *testing.T) {
	dbStore, cleanup := sqldbSetup(t)
	defer cleanup()

	testDeleteSuccess(t, dbStore)
	assertExpectedGormTUFMeta(t, nil, dbStore.DB)

	dbStore.DB.Close()
}

func TestSQLGetKeyNoKey(t *testing.T) {
	dbStore, cleanup := sqldbSetup(t)
	defer cleanup()

	cipher, public, err := dbStore.GetKey("testGUN", data.CanonicalTimestampRole)
	require.Equal(t, "", cipher)
	require.Nil(t, public)
	require.IsType(t, &ErrNoKey{}, err,
		"Expected ErrNoKey from GetKey")

	query := dbStore.DB.Create(&Key{
		Gun:    "testGUN",
		Role:   data.CanonicalTimestampRole,
		Cipher: "testCipher",
		Public: []byte("1"),
	})
	require.NoError(
		t, query.Error, "Inserting timestamp into empty DB should succeed")

	cipher, public, err = dbStore.GetKey("testGUN", data.CanonicalTimestampRole)
	require.NoError(t, err)
	require.Equal(t, "testCipher", cipher,
		"Returned cipher was incorrect")
	require.Equal(t, []byte("1"), public, "Returned pubkey was incorrect")
}

func TestSQLSetKeyExists(t *testing.T) {
	dbStore, cleanup := sqldbSetup(t)
	defer cleanup()

	err := dbStore.SetKey("testGUN", data.CanonicalTimestampRole, "testCipher", []byte("1"))
	require.NoError(t, err, "Inserting timestamp into empty DB should succeed")

	err = dbStore.SetKey("testGUN", data.CanonicalTimestampRole, "testCipher", []byte("1"))
	require.Error(t, err)
	require.IsType(t, &ErrKeyExists{}, err,
		"Expected ErrKeyExists from SetKey")

	var rows []Key
	query := dbStore.DB.Select("id, gun, cipher, public").Find(&rows)
	require.NoError(t, query.Error)

	expected := Key{Gun: "testGUN", Cipher: "testCipher",
		Public: []byte("1")}
	expected.Model = gorm.Model{ID: 1}

	require.Equal(t, []Key{expected}, rows)

	dbStore.DB.Close()
}

func TestSQLSetKeyMultipleRoles(t *testing.T) {
	dbStore, cleanup := sqldbSetup(t)
	defer cleanup()

	err := dbStore.SetKey("testGUN", data.CanonicalTimestampRole, "testCipher", []byte("1"))
	require.NoError(t, err, "Inserting timestamp into empty DB should succeed")

	err = dbStore.SetKey("testGUN", data.CanonicalSnapshotRole, "testCipher", []byte("1"))
	require.NoError(t, err, "Inserting snapshot key into DB with timestamp key should succeed")

	var rows []Key
	query := dbStore.DB.Select("id, gun, role, cipher, public").Find(&rows)
	require.NoError(t, query.Error)

	expectedTS := Key{Gun: "testGUN", Role: "timestamp", Cipher: "testCipher",
		Public: []byte("1")}
	expectedTS.Model = gorm.Model{ID: 1}

	expectedSN := Key{Gun: "testGUN", Role: "snapshot", Cipher: "testCipher",
		Public: []byte("1")}
	expectedSN.Model = gorm.Model{ID: 2}

	require.Equal(t, []Key{expectedTS, expectedSN}, rows)

	dbStore.DB.Close()
}

func TestSQLSetKeyMultipleGuns(t *testing.T) {
	dbStore, cleanup := sqldbSetup(t)
	defer cleanup()

	err := dbStore.SetKey("testGUN", data.CanonicalTimestampRole, "testCipher", []byte("1"))
	require.NoError(t, err, "Inserting timestamp into empty DB should succeed")

	err = dbStore.SetKey("testAnotherGUN", data.CanonicalTimestampRole, "testCipher", []byte("1"))
	require.NoError(t, err, "Inserting snapshot key into DB with timestamp key should succeed")

	var rows []Key
	query := dbStore.DB.Select("id, gun, role, cipher, public").Find(&rows)
	require.NoError(t, query.Error)

	expected1 := Key{Gun: "testGUN", Role: "timestamp", Cipher: "testCipher",
		Public: []byte("1")}
	expected1.Model = gorm.Model{ID: 1}

	expected2 := Key{Gun: "testAnotherGUN", Role: "timestamp", Cipher: "testCipher",
		Public: []byte("1")}
	expected2.Model = gorm.Model{ID: 2}

	require.Equal(t, []Key{expected1, expected2}, rows)

	dbStore.DB.Close()
}

func TestSQLSetKeySameRoleGun(t *testing.T) {
	dbStore, cleanup := sqldbSetup(t)
	defer cleanup()

	err := dbStore.SetKey("testGUN", data.CanonicalTimestampRole, "testCipher", []byte("1"))
	require.NoError(t, err, "Inserting timestamp into empty DB should succeed")

	err = dbStore.SetKey("testGUN", data.CanonicalTimestampRole, "testCipher", []byte("2"))
	require.Error(t, err)
	require.IsType(t, &ErrKeyExists{}, err,
		"Expected ErrKeyExists from SetKey")

	dbStore.DB.Close()
}

// TestSQLDBCheckHealthTableMissing asserts that the health check fails if one or
// both the tables are missing.
func TestSQLDBCheckHealthTableMissing(t *testing.T) {
	dbStore, cleanup := sqldbSetup(t)
	defer cleanup()

	dbStore.DropTable(&TUFFile{})
	dbStore.DropTable(&Key{})

	// No tables, health check fails
	err := dbStore.CheckHealth()
	require.Error(t, err, "Cannot access table:")

	// only one table existing causes health check to fail
	CreateTUFTable(dbStore.DB)
	err = dbStore.CheckHealth()
	require.Error(t, err, "Cannot access table:")
	dbStore.DropTable(&TUFFile{})

	CreateKeyTable(dbStore.DB)
	err = dbStore.CheckHealth()
	require.Error(t, err, "Cannot access table:")
}

// TestSQLDBCheckHealthDBConnection asserts that if the DB is not connectable, the
// health check fails.
func TestSQLDBCheckHealthDBConnectionFail(t *testing.T) {
	dbStore, cleanup := sqldbSetup(t)
	defer cleanup()

	err := dbStore.Close()
	require.NoError(t, err)

	err = dbStore.CheckHealth()
	require.Error(t, err, "Cannot access table:")
}

// TestSQLDBCheckHealthSuceeds asserts that if the DB is connectable and both
// tables exist, the health check succeeds.
func TestSQLDBCheckHealthSucceeds(t *testing.T) {
	dbStore, cleanup := sqldbSetup(t)
	defer cleanup()

	err := dbStore.CheckHealth()
	require.NoError(t, err)
}

func TestSQLDBGetChecksum(t *testing.T) {
	dbStore, cleanup := sqldbSetup(t)
	defer cleanup()

	ts := data.SignedTimestamp{
		Signatures: make([]data.Signature, 0),
		Signed: data.Timestamp{
			SignedCommon: data.SignedCommon{
				Type:    data.TUFTypes[data.CanonicalTimestampRole],
				Version: 1,
				Expires: data.DefaultExpires(data.CanonicalTimestampRole),
			},
		},
	}
	j, err := json.Marshal(&ts)
	require.NoError(t, err)
	update := MetaUpdate{
		Role:    data.CanonicalTimestampRole,
		Version: 1,
		Data:    j,
	}
	checksumBytes := sha256.Sum256(j)
	checksum := hex.EncodeToString(checksumBytes[:])

	dbStore.UpdateCurrent("gun", update)

	// create and add a newer timestamp. We're going to try and get the one
	// created above by checksum
	ts = data.SignedTimestamp{
		Signatures: make([]data.Signature, 0),
		Signed: data.Timestamp{
			SignedCommon: data.SignedCommon{
				Type:    data.TUFTypes[data.CanonicalTimestampRole],
				Version: 2,
				Expires: data.DefaultExpires(data.CanonicalTimestampRole),
			},
		},
	}
	newJ, err := json.Marshal(&ts)
	require.NoError(t, err)
	update = MetaUpdate{
		Role:    data.CanonicalTimestampRole,
		Version: 2,
		Data:    newJ,
	}

	dbStore.UpdateCurrent("gun", update)

	cDate, data, err := dbStore.GetChecksum("gun", data.CanonicalTimestampRole, checksum)
	require.NoError(t, err)
	require.EqualValues(t, j, data)
	// the creation date was sometime wthin the last minute
	require.True(t, cDate.After(time.Now().Add(-1*time.Minute)))
	require.True(t, cDate.Before(time.Now().Add(5*time.Second)))
}

func TestSQLDBGetChecksumNotFound(t *testing.T) {
	dbStore, cleanup := sqldbSetup(t)
	defer cleanup()

	_, _, err := dbStore.GetChecksum("gun", data.CanonicalTimestampRole, "12345")
	require.Error(t, err)
	require.IsType(t, ErrNotFound{}, err)
}

func TestSQLTUFMetaStoreGetCurrent(t *testing.T) {
	dbStore, cleanup := sqldbSetup(t)
	defer cleanup()

	testTUFMetaStoreGetCurrent(t, dbStore)
}
