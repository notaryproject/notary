package storage

import (
	"github.com/docker/notary/storage/rethinkdb"
)

// These consts are the index names we've defined for RethinkDB
const (
	rdbSHA256Idx         = "sha256"
	rdbGunNamespaceIdx   = "gun_namespace"
	rdbGunRoleIdx        = "gun_role_namespace"
	rdbGunRoleSHA256Idx  = "gun_role_sha256"
	rdbGunRoleVersionIdx = "gun_role_version_namespace"
)

var (
	// TUFFilesRethinkTable is the table definition of notary server's TUF metadata files
	TUFFilesRethinkTable = rethinkdb.Table{
		Name:       RDBTUFFile{}.TableName(),
		PrimaryKey: rdbGunRoleVersionIdx,
		SecondaryIndexes: map[string][]string{
			rdbSHA256Idx:         nil,
			"gun":                nil,
			"timestamp_checksum": nil,
			rdbGunRoleIdx:        {"gun", "role", "namespace"},
			rdbGunRoleSHA256Idx:  {"gun", "role", "sha256"},
			rdbGunNamespaceIdx:   {"gun", "namespace"},
		},
		// this configuration guarantees linearizability of individual atomic operations on individual documents
		Config: map[string]string{
			"write_acks": "majority",
		},
		JSONUnmarshaller: rdbTUFFileFromJSON,
	}
)
