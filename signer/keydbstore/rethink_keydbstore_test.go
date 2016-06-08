package keydbstore

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/docker/notary/storage/rethinkdb"
	"github.com/stretchr/testify/require"
)

func TestRDBTUFFileMarshalling(t *testing.T) {
	rdb := RDBPrivateKey{
		Timing: rethinkdb.Timing{
			CreatedAt: time.Now().AddDate(-1, -1, -1),
			UpdatedAt: time.Now().AddDate(0, -5, 0),
			DeletedAt: time.Time{},
		},
		KeyID:           "56ee4a23129fc22c6cb4b4ba5f78d730c91ab6def514e80d807c947bb21f0d63",
		EncryptionAlg:   "A256GCM",
		KeywrapAlg:      "PBES2-HS256+A128KW",
		Algorithm:       "ecdsa",
		PassphraseAlias: "timestamp_1",
		Public:          "Hello world public",
		Private:         "Hello world private",
	}
	marshalled, err := json.Marshal(rdb)
	require.NoError(t, err)

	unmarshalled := RDBPrivateKey{}
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
