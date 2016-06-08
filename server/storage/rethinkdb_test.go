package storage

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/docker/notary/storage/rethinkdb"
	"github.com/stretchr/testify/require"
)

func TestRDBTUFFileMarshalling(t *testing.T) {
	rdb := RDBTUFFile{
		Timing: rethinkdb.Timing{
			CreatedAt: time.Now().AddDate(-1, -1, -1),
			UpdatedAt: time.Now().AddDate(0, -5, 0),
			DeletedAt: time.Time{},
		},
		GunRoleVersion: []interface{}{"completely", "invalid", "garbage"},
		Gun:            "namespaced/name",
		Role:           "timestamp",
		Version:        5,
		Sha256:         "56ee4a23129fc22c6cb4b4ba5f78d730c91ab6def514e80d807c947bb21f0d63",
		Data:           []byte("Hello world"),
		TSchecksum:     "ebe6b6e082c94ef24043f1786a7046432506c3d193a47c299ed48ff4413ad7b0",
	}
	marshalled, err := json.Marshal(rdb)
	require.NoError(t, err)

	unmarshalled := RDBTUFFile{}
	require.NoError(t, json.Unmarshal(marshalled, &unmarshalled))

	// There is some weirdness with comparing time.Time due to a location pointer,
	// so let's use time.Time's equal function to compare times, and then re-assign
	// the timing struct to compare the rest of the RDBTUFFile struct
	require.True(t, rdb.CreatedAt.Equal(unmarshalled.CreatedAt))
	require.True(t, rdb.UpdatedAt.Equal(unmarshalled.UpdatedAt))
	require.True(t, rdb.DeletedAt.Equal(unmarshalled.DeletedAt))
	unmarshalled.Timing = rdb.Timing

	rdb.GunRoleVersion = []interface{}{rdb.Gun, rdb.Role, rdb.Version}
	require.Equal(t, rdb, unmarshalled)
}

func TestRDBTUFKeyMarshalling(t *testing.T) {
	rdb := RDBKey{
		Timing: rethinkdb.Timing{
			CreatedAt: time.Now().AddDate(-1, -1, -1),
			UpdatedAt: time.Now().AddDate(0, -5, 0),
			DeletedAt: time.Time{},
		},
		Gun:    "namespaced/name",
		Role:   "timestamp",
		Cipher: "ecdsa",
		Public: []byte("Hello world"),
	}
	marshalled, err := json.Marshal(rdb)
	require.NoError(t, err)

	unmarshalled := RDBKey{}
	require.NoError(t, json.Unmarshal(marshalled, &unmarshalled))

	// There is some weirdness with comparing time.Time due to a location pointer,
	// so let's use time.Time's equal function to compare times, and then re-assign
	// the timing struct to compare the rest of the RDBTUFFile struct
	require.True(t, rdb.CreatedAt.Equal(unmarshalled.CreatedAt))
	require.True(t, rdb.UpdatedAt.Equal(unmarshalled.UpdatedAt))
	require.True(t, rdb.DeletedAt.Equal(unmarshalled.DeletedAt))
	unmarshalled.Timing = rdb.Timing

	require.Equal(t, rdb, unmarshalled)
}
