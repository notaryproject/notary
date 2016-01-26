package timestamp

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/docker/notary/tuf/data"
	"github.com/docker/notary/tuf/signed"
	"github.com/docker/notary/tuf/testutils"
	"github.com/stretchr/testify/assert"

	"github.com/docker/notary/server/storage"
)

func TestTimestampExpired(t *testing.T) {
	ts := &data.SignedTimestamp{
		Signatures: nil,
		Signed: data.Timestamp{
			Expires: time.Now().AddDate(-1, 0, 0),
		},
	}
	assert.True(t, timestampExpired(ts), "Timestamp should have expired")
}

func TestTimestampNotExpired(t *testing.T) {
	ts := &data.SignedTimestamp{
		Signatures: nil,
		Signed: data.Timestamp{
			Expires: time.Now().AddDate(1, 0, 0),
		},
	}
	assert.False(t, timestampExpired(ts), "Timestamp should NOT have expired")
}

func TestGetTimestampKey(t *testing.T) {
	store := storage.NewMemStorage()
	crypto := signed.NewEd25519()
	k, err := GetOrCreateTimestampKey("gun", store, crypto, data.ED25519Key)
	assert.Nil(t, err, "Expected nil error")
	assert.NotNil(t, k, "Key should not be nil")

	k2, err := GetOrCreateTimestampKey("gun", store, crypto, data.ED25519Key)

	assert.Nil(t, err, "Expected nil error")

	// trying to get the same key again should return the same value
	assert.Equal(t, k, k2, "Did not receive same key when attempting to recreate.")
	assert.NotNil(t, k2, "Key should not be nil")
}

func TestGetTimestamp(t *testing.T) {
	store := storage.NewMemStorage()
	_, repo, crypto, err := testutils.EmptyRepo("gun")
	assert.NoError(t, err)

	rootJSON, err := json.Marshal(repo.Root)
	assert.NoError(t, err)
	snapJSON, err := json.Marshal(repo.Snapshot)
	assert.NoError(t, err)

	store.UpdateCurrent("gun", storage.MetaUpdate{Role: "root", Version: 0, Data: rootJSON})
	store.UpdateCurrent("gun", storage.MetaUpdate{Role: "snapshot", Version: 0, Data: snapJSON})
	// create a key to be used by GetTimestamp
	_, err = GetOrCreateTimestampKey("gun", store, crypto, data.ED25519Key)
	assert.Nil(t, err, "GetKey errored")

	_, err = GetOrCreateTimestamp("gun", store, crypto)
	assert.Nil(t, err, "GetTimestamp errored")
}

func TestGetTimestampNewSnapshot(t *testing.T) {
	store := storage.NewMemStorage()
	_, repo, crypto, err := testutils.EmptyRepo("gun")
	assert.NoError(t, err)

	rootJSON, err := json.Marshal(repo.Root)
	assert.NoError(t, err)
	snapJSON, err := json.Marshal(repo.Snapshot)
	assert.NoError(t, err)

	store.UpdateCurrent("gun", storage.MetaUpdate{Role: "root", Version: 0, Data: rootJSON})
	store.UpdateCurrent("gun", storage.MetaUpdate{Role: "snapshot", Version: 0, Data: snapJSON})
	// create a key to be used by GetTimestamp
	_, err = GetOrCreateTimestampKey("gun", store, crypto, data.ED25519Key)
	assert.Nil(t, err, "GetKey errored")

	ts1, err := GetOrCreateTimestamp("gun", store, crypto)
	assert.Nil(t, err, "GetTimestamp errored")

	snapshot := data.SignedSnapshot{}
	snapshot.Signed.Version = 1
	snapJSON, _ = json.Marshal(snapshot)

	store.UpdateCurrent("gun", storage.MetaUpdate{Role: "snapshot", Version: 1, Data: snapJSON})

	ts2, err := GetOrCreateTimestamp("gun", store, crypto)
	assert.NoError(t, err, "GetTimestamp errored")

	assert.NotEqual(t, ts1, ts2, "Timestamp was not regenerated when snapshot changed")

	ts := &data.SignedTimestamp{}
	err = json.Unmarshal(ts2, &ts)
	meta, err := data.NewFileMeta(bytes.NewReader(ts1), "sha256")
	assert.EqualValues(t, meta, ts.Signed.Meta[data.PreviousTSName])
}
