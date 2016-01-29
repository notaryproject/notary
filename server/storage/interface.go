package storage

import (
	"time"

	"github.com/docker/notary/tuf/data"
)

// KeyStore provides a minimal interface for managing key persistence.  Any
// or all of these functions can optionally clean up expired pending keys, or
// no-longer-active non-pending keys.
type KeyStore interface {
	// GetLatestKey the most recently created, non-expired key for the given
	// GUN and role. If no keys exist for the GUN+role, returns an ErrNoKeys.
	GetLatestKey(gun, role string) (*ManagedPublicKey, error)

	// HasAnyKeys returns true if any non-expired keys exist for the given GUN,
	// role, and key IDs.
	HasAnyKeys(gun, role string, keyIDs []string) (bool, error)

	// AddKey adds the given public key for the given GUN and role, with an
	// expiration time.  The key is added as a pending key - MarkActiveKeys
	// must be called to make it active.
	AddKey(gun, role string, key data.PublicKey, expires time.Time) error

	// MarkActiveKeys marks the following key IDs as active.
	// This does not fail if any of the key IDs doesn't exist.
	MarkActiveKeys(gun, role string, keyIDs []string) error
}

// MetaStore holds the methods that are used for a Metadata Store
type MetaStore interface {
	// UpdateCurrent adds new metadata version for the given GUN if and only
	// if it's a new role, or the version is greater than the current version
	// for the role. Otherwise an error is returned.
	UpdateCurrent(gun string, update MetaUpdate) error

	// UpdateMany adds multiple new metadata for the given GUN.  It can even
	// add multiple versions for the same role, so long as those versions are
	// all unique and greater than any current versions.  Otherwise,
	// none of the metadata is added, and an error is be returned.
	UpdateMany(gun string, updates []MetaUpdate) error

	// GetCurrent returns the data part of the metadata for the latest version
	// of the given GUN and role.  If there is no data for the given GUN and
	// role, an error is returned.
	GetCurrent(gun, tufRole string) (data []byte, err error)

	// GetChecksum return the given tuf role file for the GUN with the
	// provided checksum. If the given (gun, role, checksum) are not
	// found, it returns storage.ErrNotFound
	GetChecksum(gun, tufRole, checksum string) (data []byte, err error)

	// Delete removes all metadata for a given GUN.  It does not return an
	// error if no metadata exists for the given GUN.
	Delete(gun string) error

	KeyStore
}
