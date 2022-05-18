package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"github.com/theupdateframework/notary/tuf/data"
)

// SQLStorage implements a versioned store using a relational database.
// See server/storage/models.go
type SQLStorage struct {
	*gorm.DB
}

// NewSQLStorage is a convenience method to create a SQLStorage
func NewSQLStorage(dialect string, args ...interface{}) (*SQLStorage, error) {
	gormDB, err := gorm.Open(dialect, args...)
	if err != nil {
		return nil, err
	}
	return &SQLStorage{
		DB: gormDB,
	}, nil
}

// translateOldVersionError captures DB errors, and attempts to translate
// duplicate entry
func translateOldVersionError(err error) error {
	switch err := err.(type) {
	case *mysql.MySQLError:
		// https://dev.mysql.com/doc/refman/5.5/en/error-messages-server.html
		// 1022 = Can't write; duplicate key in table '%s'
		// 1062 = Duplicate entry '%s' for key %d
		if err.Number == 1022 || err.Number == 1062 {
			return ErrOldVersion{}
		}
	case pq.Error:
		// https://www.postgresql.org/docs/10/errcodes-appendix.html
		// 23505 = unique_violation
		if err.Code == "23505" {
			return ErrOldVersion{}
		}
	}
	return err
}

// UpdateCurrent updates a single TUF.
func (db *SQLStorage) UpdateCurrent(gun data.GUN, update MetaUpdate) error {
	// ensure we're not inserting an immediately old version - can't use the
	// struct, because that only works with non-zero values, and Version
	// can be 0.
	exists := db.Where("gun = ? and role = ? and version >= ?",
		gun.String(), update.Role.String(), update.Version).Take(&TUFFile{})

	if exists.Error == nil {
		return ErrOldVersion{}
	} else if !exists.RecordNotFound() {
		return exists.Error
	}

	// only take out the transaction once we're about to start writing
	tx, rb, err := db.getTransaction()
	if err != nil {
		return err
	}

	checksum := sha256.Sum256(update.Data)
	hexChecksum := hex.EncodeToString(checksum[:])

	if err := func() error {
		// write new TUFFile entry
		if err = translateOldVersionError(tx.Create(&TUFFile{
			Gun:     gun.String(),
			Role:    update.Role.String(),
			Version: update.Version,
			SHA256:  hexChecksum,
			Data:    update.Data,
		}).Error); err != nil {
			return err
		}

		// If we're publishing a timestamp, update the changefeed as this is
		// technically an new version of the TUF repo
		if update.Role == data.CanonicalTimestampRole {
			if err := db.writeChangefeed(tx, gun, update.Version, hexChecksum); err != nil {
				return err
			}
		}
		return nil
	}(); err != nil {
		return rb(err)
	}
	return tx.Commit().Error
}

type rollback func(error) error

func (db *SQLStorage) getTransaction() (*gorm.DB, rollback, error) {
	tx := db.Begin()
	if tx.Error != nil {
		return nil, nil, tx.Error
	}

	rb := func(err error) error {
		if rxErr := tx.Rollback().Error; rxErr != nil {
			logrus.Error("Failed on Tx rollback with error: ", rxErr.Error())
			return rxErr
		}
		return err
	}

	return tx, rb, nil
}

// UpdateMany atomically updates many TUF records in a single transaction
func (db *SQLStorage) UpdateMany(gun data.GUN, updates []MetaUpdate) error {
	if !allUpdatesUnique(updates) {
		// We would fail with a unique constraint violation later, so just bail out now
		return ErrOldVersion{}
	}

	minVersionsByRole := make(map[data.RoleName]int)
	for _, u := range updates {
		cur, ok := minVersionsByRole[u.Role]
		if !ok || u.Version < cur {
			minVersionsByRole[u.Role] = u.Version
		}
	}

	for role, minVersion := range minVersionsByRole {
		// If there are any files with version equal or higher than the minimum
		// version we're trying to insert, bail out now
		exists := db.Where("gun = ? and role = ? and version >= ?",
			gun.String(), role.String(), minVersion).Take(&TUFFile{})

		if exists.Error == nil {
			return ErrOldVersion{}
		} else if !exists.RecordNotFound() {
			return exists.Error
		}
	}

	tx, rb, err := db.getTransaction()
	if err != nil {
		return err
	}

	if err := func() error {
		for _, update := range updates {
			checksum := sha256.Sum256(update.Data)
			hexChecksum := hex.EncodeToString(checksum[:])

			result := tx.Create(&TUFFile{
				Gun:     gun.String(),
				Role:    update.Role.String(),
				Version: update.Version,
				Data:    update.Data,
				SHA256:  hexChecksum,
			})

			if result.Error != nil {
				return translateOldVersionError(result.Error)
			}

			if update.Role == data.CanonicalTimestampRole {
				if err := db.writeChangefeed(tx, gun, update.Version, hexChecksum); err != nil {
					return err
				}
			}
		}
		return nil
	}(); err != nil {
		return rb(err)
	}
	return tx.Commit().Error
}

func allUpdatesUnique(updates []MetaUpdate) bool {
	type roleVersion struct {
		Role    data.RoleName
		Version int
	}
	roleVersions := make(map[roleVersion]bool)
	for _, u := range updates {
		rv := roleVersion{u.Role, u.Version}
		if roleVersions[rv] {
			return false
		}
		roleVersions[rv] = true
	}
	return true
}

func (db *SQLStorage) writeChangefeed(tx *gorm.DB, gun data.GUN, version int, checksum string) error {
	c := &SQLChange{
		GUN:      gun.String(),
		Version:  version,
		SHA256:   checksum,
		Category: changeCategoryUpdate,
	}
	return tx.Create(c).Error
}

// GetCurrent gets a specific TUF record
func (db *SQLStorage) GetCurrent(gun data.GUN, tufRole data.RoleName) (*time.Time, []byte, error) {
	var row TUFFile
	q := db.Select("updated_at, data").Where(
		&TUFFile{Gun: gun.String(), Role: tufRole.String()}).Order("version desc").Take(&row)
	if err := isReadErr(q, row); err != nil {
		return nil, nil, err
	}
	return &(row.UpdatedAt), row.Data, nil
}

// GetChecksum gets a specific TUF record by its hex checksum
func (db *SQLStorage) GetChecksum(gun data.GUN, tufRole data.RoleName, checksum string) (*time.Time, []byte, error) {
	var row TUFFile
	q := db.Select("created_at, data").Where(
		&TUFFile{
			Gun:    gun.String(),
			Role:   tufRole.String(),
			SHA256: checksum,
		},
	).Take(&row)
	if err := isReadErr(q, row); err != nil {
		return nil, nil, err
	}
	return &(row.CreatedAt), row.Data, nil
}

// GetVersion gets a specific TUF record by its version
func (db *SQLStorage) GetVersion(gun data.GUN, tufRole data.RoleName, version int) (*time.Time, []byte, error) {
	var row TUFFile
	q := db.Select("created_at, data").Where(
		&TUFFile{
			Gun:     gun.String(),
			Role:    tufRole.String(),
			Version: version,
		},
	).Take(&row)
	if err := isReadErr(q, row); err != nil {
		return nil, nil, err
	}
	return &(row.CreatedAt), row.Data, nil
}

func isReadErr(q *gorm.DB, row TUFFile) error {
	if q.RecordNotFound() {
		return ErrNotFound{}
	} else if q.Error != nil {
		return q.Error
	}
	return nil
}

// Delete deletes all the records for a specific GUN - we have to do a hard delete using Unscoped
// otherwise we can't insert for that GUN again
func (db *SQLStorage) Delete(gun data.GUN) error {
	tx, rb, err := db.getTransaction()
	if err != nil {
		return err
	}
	if err := func() error {
		res := tx.Unscoped().Where(&TUFFile{Gun: gun.String()}).Delete(TUFFile{})
		if err := res.Error; err != nil {
			return err
		}
		// if there weren't actually any records for the GUN, don't write
		// a deletion change record.
		if res.RowsAffected == 0 {
			return nil
		}
		c := &SQLChange{
			GUN:      gun.String(),
			Category: changeCategoryDeletion,
		}
		return tx.Create(c).Error
	}(); err != nil {
		return rb(err)
	}
	return tx.Commit().Error
}

// CheckHealth asserts that the tuf_files table is present
func (db *SQLStorage) CheckHealth() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("Panic checking db health: %v", r)
		}
	}()

	tableOk := db.HasTable(&TUFFile{})
	if db.Error != nil {
		return db.Error
	}
	if !tableOk {
		return fmt.Errorf(
			"Cannot access table: %s", TUFFile{}.TableName())
	}
	return nil
}

// GetChanges returns up to pageSize changes starting from changeID.
func (db *SQLStorage) GetChanges(changeID string, records int, filterName string) ([]Change, error) {
	var (
		changes []Change
		query   = db.DB
		id      int64
		err     error
	)
	if changeID == "" {
		id = 0
	} else {
		id, err = strconv.ParseInt(changeID, 10, 32)
		if err != nil {
			return nil, ErrBadQuery{msg: fmt.Sprintf("change ID expected to be integer, provided ID was: %s", changeID)}
		}
	}

	// do what I mean, not what I said, i.e. if I passed a negative number for the ID
	// it's assumed I mean "start from latest and go backwards"
	reversed := id < 0
	if records < 0 {
		reversed = true
		records = -records
	}

	if filterName != "" {
		query = query.Where("gun = ?", filterName)
	}
	if reversed {
		if id > 0 {
			// only set the id check if we're not starting from "latest"
			query = query.Where("id < ?", id)
		}
		query = query.Order("id desc")
	} else {
		query = query.Where("id > ?", id).Order("id asc")
	}

	res := query.Limit(records).Find(&changes)
	if res.Error != nil {
		return nil, res.Error
	}

	if reversed {
		// results are currently newest to oldest, should be oldest to newest
		for i, j := 0, len(changes)-1; i < j; i, j = i+1, j-1 {
			changes[i], changes[j] = changes[j], changes[i]
		}
	}

	return changes, nil
}
