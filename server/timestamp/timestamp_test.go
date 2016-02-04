package timestamp

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/docker/notary/tuf/data"
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
	key, err := crypto.Create(data.CanonicalTimestampRole, data.ECDSAKey)
	assert.NoError(t, err)
	assert.NoError(t, store.AddKey("gun", data.CanonicalTimestampRole, key, time.Now().AddDate(1, 1, 1)))

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
	key, err := crypto.Create(data.CanonicalTimestampRole, data.ECDSAKey)
	assert.NoError(t, err)
	assert.NoError(t, store.AddKey("gun", data.CanonicalTimestampRole, key, time.Now().AddDate(1, 1, 1)))

	ts1, err := GetOrCreateTimestamp("gun", store, crypto)
	assert.Nil(t, err, "GetTimestamp errored")

	snapshot := data.SignedSnapshot{}
	snapshot.Signed.Version = 1
	snapJSON, _ = json.Marshal(snapshot)

	store.UpdateCurrent("gun", storage.MetaUpdate{Role: "snapshot", Version: 1, Data: snapJSON})

	ts2, err := GetOrCreateTimestamp("gun", store, crypto)
	assert.NoError(t, err, "GetTimestamp errored")

	assert.NotEqual(t, ts1, ts2, "Timestamp was not regenerated when snapshot changed")
}
