package storage

import "github.com/docker/notary/tuf/data"

// MetaUpdate packages up the fields required to update a TUF record
type MetaUpdate struct {
	Role    string
	Version int
	Data    []byte
}

// ManagedPublicKey is just a data.PublicKey along with a field that specifies
// when it was last used, so it might be cleaned up
type ManagedPublicKey struct {
	data.PublicKey
	Pending bool
}
