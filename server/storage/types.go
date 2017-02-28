package storage

import "github.com/docker/notary/tuf/data"

// Namespace defines a context for metadata, for example for staging partially-signed metadata
// on a per-user basis
type Namespace string

func (n Namespace) String() string {
	return string(n)
}

// DefaultNamespace is the namespace all fully signed, validated metadata lives in
const DefaultNamespace Namespace = "default"

// MetaUpdate packages up the fields required to update a TUF record
type MetaUpdate struct {
	Role    data.RoleName
	Version int
	Data    []byte
}
