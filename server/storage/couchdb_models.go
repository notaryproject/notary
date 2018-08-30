package storage

import (
	"github.com/theupdateframework/notary/storage/couchdb"
)

var (
	// TUFFilesCouchTable is the table definition of notary server's TUF metadata files
	TUFFilesCouchTable = couchdb.Table{
		Name: CDBTUFFile{}.TableName(),
		Indexes: map[string]interface{}{
			"1": []string{"version"},
			"2": []string{"gun", "role", "version"},
			"3": []string{"gun", "role", "sha256"},
			"4": []string{"gun"},
			"5": []string{"timestamp_checksum"},
			"6": []string{"gun", "created_at"},
			"7": []string{"_id", "created_at"},
			"8": []string{"gun", "role"},
		},
	}

	// ChangeCouchTable is the table definition for changefeed objects
	ChangeCouchTable = couchdb.Table{
		Name: Change{}.TableName(),
		Indexes: map[string]interface{}{
			"1": []map[string]string{{"created_at": "desc"}},
			"2": []map[string]string{{"created_at": "asc"}},
		},
		JSONUnmarshaller: cdbChangeFromJSON,
	}

	// CouchTableNames holds the names of all server tables
	CouchTableNames = []string{
		TUFFilesCouchTable.Name,
		ChangeCouchTable.Name,
	}
)
