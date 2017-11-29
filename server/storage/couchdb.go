package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/theupdateframework/notary/storage/couchdb"
	"github.com/theupdateframework/notary/tuf/data"

	"github.com/flimzy/kivik"
)

// CDBTUFFile is a TUF file record
type CDBTUFFile struct {
	couchdb.ID
	couchdb.Timing
	GunRoleVersion []interface{} `json:"gun_role_version"`
	Gun            string        `json:"gun"`
	Role           string        `json:"role"`
	Version        int           `json:"version"`
	SHA256         string        `json:"sha256"`
	Data           []byte        `json:"data"`
	TSchecksum     string        `json:"timestamp_checksum"`
}

// TableName returns the table name for the record type
func (r CDBTUFFile) TableName() string {
	return TUFFileTableName
}

// CChange defines the the fields required for an object in the changefeed
type CChange struct {
	couchdb.ID
	CreatedAt time.Time `json:"created_at"`
	GUN       string    `json:"gun"`
	Version   int       `json:"version"`
	SHA256    string    `json:"sha256"`
	Category  string    `json:"category"`
}

// TableName sets a specific table name for Changefeed
func (cdb CChange) TableName() string {
	return ChangefeedTableName
}

// unmarshal in an anonymous struct
func cdbTUFFileFromJSON(data []byte) (interface{}, error) {
	a := struct {
		CreatedAt  time.Time `json:"created_at"`
		UpdatedAt  time.Time `json:"updated_at"`
		DeletedAt  time.Time `json:"deleted_at"`
		Gun        string    `json:"gun"`
		Role       string    `json:"role"`
		Version    int       `json:"version"`
		SHA256     string    `json:"sha256"`
		Data       []byte    `json:"data"`
		TSchecksum string    `json:"timestamp_checksum"`
	}{}
	if err := json.Unmarshal(data, &a); err != nil {
		return CDBTUFFile{}, err
	}
	return CDBTUFFile{
		Timing: couchdb.Timing{
			CreatedAt: a.CreatedAt,
			UpdatedAt: a.UpdatedAt,
			DeletedAt: a.DeletedAt,
		},
		GunRoleVersion: []interface{}{a.Gun, a.Role, a.Version},
		Gun:            a.Gun,
		Role:           a.Role,
		Version:        a.Version,
		SHA256:         a.SHA256,
		Data:           a.Data,
		TSchecksum:     a.TSchecksum,
	}, nil
}

func cdbChangeFromJSON(data []byte) (interface{}, error) {
	res := CChange{}
	if err := json.Unmarshal(data, &res); err != nil {
		return CChange{}, err
	}
	return res, nil
}

// CouchDB implements a MetaStore against the Couchdb Database
type CouchDB struct {
	dbName   string
	client   *kivik.Client
	user     string
	password string
}

// NewCouchDBStorage initializes a CouchDB object
func NewCouchDBStorage(dbName, user, password string, client *kivik.Client) CouchDB {
	return CouchDB{
		dbName:   dbName,
		client:   client,
		user:     user,
		password: password,
	}
}

func genID(gun, role string, version int) string {
	return fmt.Sprintf("%s.%s.%d", gun, role, version)
}

// UpdateCurrent adds new metadata version for the given GUN if and only
// if it's a new role, or the version is greater than the current version
// for the role. Otherwise an error is returned.
func (cdb CouchDB) UpdateCurrent(gun data.GUN, update MetaUpdate) error {

	file := CDBTUFFile{}

	db, err := couchdb.DB(cdb.client, cdb.dbName, file.TableName())
	if err != nil {
		return err
	}

	id := genID(gun.String(), update.Role.String(), update.Version)

	row, _ := db.Get(context.TODO(), id)
	if row != nil {
		return ErrOldVersion{}
	}
	// empty string is the zero value for tsChecksum in the CDBTUFFile struct.
	// Therefore we can just call through to updateCurrentWithTSChecksum passing
	// "" for the tsChecksum value.
	if err := cdb.updateCurrentWithTSChecksum(gun.String(), "", update); err != nil {
		return err
	}
	if update.Role == data.CanonicalTimestampRole {
		tsChecksumBytes := sha256.Sum256(update.Data)
		return cdb.writeChange(
			gun.String(),
			update.Version,
			hex.EncodeToString(tsChecksumBytes[:]),
			changeCategoryUpdate,
		)
	}
	return nil
}

// updateCurrentWithTSChecksum adds new metadata version for the given GUN with an associated
// checksum for the timestamp it belongs to, to afford us transaction-like functionality
func (cdb CouchDB) updateCurrentWithTSChecksum(gun, tsChecksum string, update MetaUpdate) error {
	now := time.Now()
	checksum := sha256.Sum256(update.Data)

	id := genID(gun, update.Role.String(), update.Version)

	file := CDBTUFFile{
		Timing: couchdb.Timing{
			CreatedAt: now,
			UpdatedAt: now,
		},
		ID: couchdb.ID{
			ID: id,
		},
		GunRoleVersion: []interface{}{gun, update.Role, update.Version},
		Gun:            gun,
		Role:           update.Role.String(),
		Version:        update.Version,
		SHA256:         hex.EncodeToString(checksum[:]),
		TSchecksum:     tsChecksum,
		Data:           update.Data,
	}

	db, err := couchdb.DB(cdb.client, cdb.dbName, file.TableName())
	if err != nil {
		return err
	}

	_, err = db.Put(context.TODO(), id, file)
	if err != nil {
		// Error message from DB 'Conflict: Document update conflict'
		if strings.Contains(err.Error(), "Document update conflict") {
			return ErrOldVersion{}
		}
		return err
	}

	return nil
}

// Used for sorting updates alphabetically by role name, such that timestamp is always last:
// Ordering: root, snapshot, targets, targets/* (delegations), timestamp
type couchUpdateSorter []MetaUpdate

func (u couchUpdateSorter) Len() int      { return len(u) }
func (u couchUpdateSorter) Swap(i, j int) { u[i], u[j] = u[j], u[i] }
func (u couchUpdateSorter) Less(i, j int) bool {
	return u[i].Role < u[j].Role
}

// UpdateMany adds multiple new metadata for the given GUN. CouchDB does
// not support transactions, therefore we will attempt to insert the timestamp
// last as this represents a published version of the repo.  However, we will
// insert all other role data in alphabetical order first, and also include the
// associated timestamp checksum so that we can easily roll back this pseudotransaction
func (cdb CouchDB) UpdateMany(gun data.GUN, updates []MetaUpdate) error {
	// find the timestamp first and save its checksum
	// then apply the updates in alphabetic role order with the timestamp last
	// if there are any failures, we roll back in the same alphabetic order
	var (
		tsChecksum string
		tsVersion  int
	)
	for _, up := range updates {
		if up.Role == data.CanonicalTimestampRole {
			tsChecksumBytes := sha256.Sum256(up.Data)
			tsChecksum = hex.EncodeToString(tsChecksumBytes[:])
			tsVersion = up.Version
			break
		}
	}

	// alphabetize the updates by Role name
	sort.Stable(couchUpdateSorter(updates))

	for _, up := range updates {
		if err := cdb.updateCurrentWithTSChecksum(gun.String(), tsChecksum, up); err != nil {
			// roll back with best-effort deletion, and then error out
			rollbackErr := cdb.deleteByTSChecksum(tsChecksum)
			if rollbackErr != nil {
				logrus.Errorf("Unable to rollback DB conflict - items with timestamp_checksum %s: %v",
					tsChecksum, rollbackErr)
			}
			return err
		}
	}

	// if the update included a timestamp, write a change object
	if tsChecksum != "" {
		return cdb.writeChange(gun.String(), tsVersion, tsChecksum, changeCategoryUpdate)
	}
	return nil
}

// GetCurrent returns the modification date and data part of the metadata for
// the latest version of the given GUN and role.  If there is no data for
// the given GUN and role, an error is returned.
func (cdb CouchDB) GetCurrent(gun data.GUN, role data.RoleName) (created *time.Time, data []byte, err error) {

	file := CDBTUFFile{}
	db, err := couchdb.DB(cdb.client, cdb.dbName, file.TableName())
	if err != nil {
		return nil, nil, err
	}

	rows, err := db.Find(context.TODO(), map[string]interface{}{
		"selector": map[string]interface{}{
			"gun":  gun.String(),
			"role": role.String(),
			"version": map[string]int{
				"$gte": 0,
			},
		},
		"sort":  []map[string]string{{"version": "desc"}},
		"limit": 1,
	})
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil, ErrNotFound{}
	}

	err = rows.ScanDoc(&file)
	if err != nil {
		return nil, nil, err
	}

	return &file.CreatedAt, file.Data, nil
}

// GetChecksum returns the given TUF role file and creation date for the
// GUN with the provided checksum. If the given (gun, role, checksum) are
// not found, it returns storage.ErrNotFound
func (cdb CouchDB) GetChecksum(gun data.GUN, role data.RoleName, checksum string) (created *time.Time, data []byte, err error) {
	var file CDBTUFFile

	db, err := couchdb.DB(cdb.client, cdb.dbName, file.TableName())
	if err != nil {
		return nil, nil, err
	}
	rows, err := db.Find(context.TODO(), map[string]interface{}{
		"selector": map[string]interface{}{
			"gun":    gun.String(),
			"role":   role.String(),
			"sha256": checksum,
		},
	})
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil, ErrNotFound{}
	}

	err = rows.ScanDoc(&file)
	if err != nil {
		return nil, nil, err
	}

	return &file.CreatedAt, file.Data, nil
}

// GetVersion gets a specific TUF record by its version
func (cdb CouchDB) GetVersion(gun data.GUN, role data.RoleName, version int) (*time.Time, []byte, error) {
	var file CDBTUFFile

	db, err := couchdb.DB(cdb.client, cdb.dbName, file.TableName())
	if err != nil {
		return nil, nil, err
	}

	rows, err := db.Find(context.TODO(), map[string]interface{}{
		"selector": map[string]interface{}{
			"gun":     gun.String(),
			"role":    role.String(),
			"version": version,
		},
	})
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil, ErrNotFound{}
	}

	err = rows.ScanDoc(&file)
	if err != nil {
		return nil, nil, err
	}

	return &file.CreatedAt, file.Data, nil
}

// Delete removes all metadata for a given GUN.  It does not return an
// error if no metadata exists for the given GUN.
func (cdb CouchDB) Delete(gun data.GUN) error {
	file := CDBTUFFile{}

	db, err := couchdb.DB(cdb.client, cdb.dbName, file.TableName())
	if err != nil {
		return err
	}

	rows, err := db.Find(context.TODO(), map[string]interface{}{
		"selector": map[string]interface{}{
			"gun": gun.String(),
		},
	})
	if err != nil {
		return err
	}
	defer rows.Close()

	deleted := 0

	for rows.Next() {
		if err := rows.ScanDoc(&file); err != nil {
			return fmt.Errorf("unable to scan document: %s", err)
		}
		if _, err = db.Delete(context.TODO(), file.ID.ID, file.ID.Rev); err != nil {
			return fmt.Errorf("unable to delete %s from database: %s", gun, err.Error())
		}
		deleted = deleted + 1
	}
	if deleted > 0 {
		return cdb.writeChange(gun.String(), 0, "", changeCategoryDeletion)
	}
	return nil
}

// deleteByTSChecksum removes all metadata by a timestamp checksum, used for rolling back a "transaction"
// from a call to couchdb's UpdateMany
func (cdb CouchDB) deleteByTSChecksum(tsChecksum string) error {
	file := CDBTUFFile{}

	db, err := couchdb.DB(cdb.client, cdb.dbName, file.TableName())
	if err != nil {
		return err
	}

	rows, err := db.Find(context.TODO(), map[string]interface{}{
		"selector": map[string]interface{}{
			"timestamp_checksum": tsChecksum,
		},
	})
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		if err := rows.ScanDoc(&file); err != nil {
			return fmt.Errorf("unable to scan document: %s", err)
		}

		if _, err = db.Delete(context.TODO(), file.ID.ID, file.ID.Rev); err != nil {
			return fmt.Errorf("unable to delete timestamp checksum data: %s from database: %s", tsChecksum, err.Error())
		}
	}

	// DO NOT WRITE CHANGE! THIS IS USED _ONLY_ TO ROLLBACK A FAILED INSERT
	return nil
}

// Bootstrap sets up the database and tables, also creating the notary server user with appropriate db permission
func (cdb CouchDB) Bootstrap() error {
	if err := couchdb.SetupDB(cdb.client, cdb.dbName, []couchdb.Table{
		TUFFilesCouchTable,
		ChangeCouchTable,
	}); err != nil {
		return err
	}
	return couchdb.CreateAndGrantDBUser(cdb.client, cdb.dbName, cdb.user, cdb.password)
}

// CheckHealth checks that all tables and databases exist and are query-able
func (cdb CouchDB) CheckHealth() error {
	for _, tablename := range CouchTableNames {
		db, err := couchdb.DB(cdb.client, cdb.dbName, tablename)
		if err != nil {
			return fmt.Errorf("Could not connect to the DB %s: %s", cdb.dbName, err)
		}
		if _, err = db.Stats(context.TODO()); err != nil {
			return err
		}
	}
	return nil
}

func (cdb CouchDB) writeChange(gun string, version int, sha256, category string) error {
	now := time.Now()
	ch := CChange{
		CreatedAt: now,
		GUN:       gun,
		Version:   version,
		SHA256:    sha256,
		Category:  category,
	}
	db, err := couchdb.DB(cdb.client, cdb.dbName, ch.TableName())
	if err != nil {
		return err
	}

	_, _, err = db.CreateDoc(context.TODO(), ch)
	return err
}

func (cdb CouchDB) getChanges(rows *kivik.Rows) []Change {
	var (
		changes []Change
		change  Change
		cchange CChange
	)

	for rows.Next() {
		rows.ScanDoc(&cchange)

		change.ID = cchange.ID.ID
		change.CreatedAt = cchange.CreatedAt
		change.GUN = cchange.GUN
		change.Version = cchange.Version
		change.SHA256 = cchange.SHA256
		change.Category = cchange.Category

		changes = append(changes, change)
	}
	return changes
}

// GetChanges returns up to pageSize changes starting from changeID. It uses the
// blackout to account for CouchDB's eventual consistency model
func (cdb CouchDB) GetChanges(changeID string, records int, filterName string) ([]Change, error) {
	var (
		id        int64
		err       error
		createdAt time.Time
		selector  map[string]interface{}
		sorter    []map[string]string
	)

	db, err := couchdb.DB(cdb.client, cdb.dbName, Change{}.TableName())
	if err != nil {
		return nil, err
	}

	switch changeID {
	case "":
	case "0":
		id = 0
	case "-1":
		id = -1
	default:
		id = 1
		createdAt, err = getCreatedAt(db, changeID)
		if err != nil {
			return nil, err
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
		selector = map[string]interface{}{
			"gun": filterName,
		}
	} else {
		selector = map[string]interface{}{
			"_id": map[string]interface{}{
				"$gt": nil,
			},
		}
	}
	if reversed {
		sorter = []map[string]string{{"created_at": "desc"}}
		if id > 0 {
			selector["created_at"] = map[string]interface{}{
				"$lt": createdAt,
			}
		}
	} else {
		sorter = []map[string]string{{"created_at": "asc"}}
		if changeID != "0" {
			selector["created_at"] = map[string]interface{}{
				"$gt": createdAt,
			}
		}
	}
	query := map[string]interface{}{
		"selector": selector,
		"sort":     sorter,
		"limit":    records,
	}

	rows, err := db.Find(context.TODO(), query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	changes := cdb.getChanges(rows)

	defer func() {
		if reversed {
			// results are currently newest to oldest, should be oldest to newest
			for i, j := 0, len(changes)-1; i < j; i, j = i+1, j-1 {
				changes[i], changes[j] = changes[j], changes[i]
			}
		}
	}()

	return changes, nil
}

func getCreatedAt(db *kivik.DB, id string) (time.Time, error) {
	query := map[string]interface{}{
		"selector": map[string]interface{}{
			"_id": id,
		},
		"limit": 1,
	}
	rows, err := db.Find(context.TODO(), query)
	if err != nil {
		return time.Now(), err
	}
	if !rows.Next() {
		return time.Now(), fmt.Errorf("Could not find entry with ID %s", id)
	}
	var change CChange
	rows.ScanDoc(&change)
	return change.CreatedAt, nil
}
