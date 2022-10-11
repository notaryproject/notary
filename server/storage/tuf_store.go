package storage

import (
	"encoding/hex"
	"fmt"
	"time"

	"github.com/docker/go/canonical/json"
	"github.com/theupdateframework/notary"
	"github.com/theupdateframework/notary/storage"
	"github.com/theupdateframework/notary/tuf/data"
)

// TUFMetaStorage wraps a MetaStore in order to walk the TUF tree for GetCurrent in a consistent manner,
// by always starting from a current timestamp and then looking up other data by hash
type TUFMetaStorage struct {
	MetaStore
}

// NewTUFMetaStorage instantiates a TUFMetaStorage instance
func NewTUFMetaStorage(m MetaStore) *TUFMetaStorage {
	return &TUFMetaStorage{
		MetaStore: m,
	}
}

// GetCurrent gets a specific TUF record, by walking from the current Timestamp to other metadata by checksum
func (tms TUFMetaStorage) GetCurrent(gun data.GUN, tufRole data.RoleName) (*time.Time, []byte, error) {
	timestampTime, timestampJSON, err := tms.MetaStore.GetCurrent(gun, data.CanonicalTimestampRole)
	if err != nil {
		return nil, nil, err
	}
	// If we wanted data for the timestamp role, we're done here
	if tufRole == data.CanonicalTimestampRole {
		return timestampTime, timestampJSON, nil
	}

	// If we want to lookup another role, walk to it via current timestamp --> snapshot by checksum --> desired role
	timestampMeta := &data.SignedTimestamp{}
	if err := json.Unmarshal(timestampJSON, timestampMeta); err != nil {
		return nil, nil, fmt.Errorf("could not parse current timestamp")
	}
	snapshotChecksums, err := timestampMeta.GetSnapshot()
	if err != nil || snapshotChecksums == nil {
		return nil, nil, fmt.Errorf("could not retrieve latest snapshot checksum")
	}
	snapshotSHA256Bytes, ok := snapshotChecksums.Hashes[notary.SHA256]
	if !ok {
		return nil, nil, fmt.Errorf("could not retrieve latest snapshot sha256")
	}
	snapshotSHA256Hex := hex.EncodeToString(snapshotSHA256Bytes[:])

	// Get the snapshot from the underlying store by checksum
	snapshotTime, snapshotJSON, err := tms.GetChecksum(gun, data.CanonicalSnapshotRole, snapshotSHA256Hex)
	if err != nil {
		return nil, nil, err
	}

	// If we wanted data for the snapshot role, we're done here
	if tufRole == data.CanonicalSnapshotRole {
		return snapshotTime, snapshotJSON, nil
	}

	// If it's a different role, we should have the checksum in snapshot metadata, and we can use it to GetChecksum()
	snapshotMeta := &data.SignedSnapshot{}
	if err := json.Unmarshal(snapshotJSON, snapshotMeta); err != nil {
		return nil, nil, fmt.Errorf("could not parse current snapshot")
	}
	roleMeta, err := snapshotMeta.GetMeta(tufRole)
	if err != nil {
		return nil, nil, err
	}
	roleSHA256Bytes, ok := roleMeta.Hashes[notary.SHA256]
	if !ok {
		return nil, nil, fmt.Errorf("could not retrieve latest %s sha256", tufRole)
	}
	roleSHA256Hex := hex.EncodeToString(roleSHA256Bytes[:])

	roleTime, roleJSON, err := tms.GetChecksum(gun, tufRole, roleSHA256Hex)
	if err != nil {
		return nil, nil, err
	}

	return roleTime, roleJSON, nil
}

// Bootstrap the store with tables if possible
func (tms TUFMetaStorage) Bootstrap() error {
	if s, ok := tms.MetaStore.(storage.Bootstrapper); ok {
		return s.Bootstrap()
	}
	return fmt.Errorf("store does not support bootstrapping")
}
