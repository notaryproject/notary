package snapshot

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/docker/notary/server/storage"
	"github.com/docker/notary/tuf/data"
	"github.com/docker/notary/tuf/signed"
	"github.com/docker/notary/tuf/testutils"
)

func TestSnapshotExpired(t *testing.T) {
	sn := &data.SignedSnapshot{
		Signatures: nil,
		Signed: data.Snapshot{
			Expires: time.Now().AddDate(-1, 0, 0),
		},
	}
	assert.True(t, snapshotExpired(sn), "Snapshot should have expired")
}

func TestSnapshotNotExpired(t *testing.T) {
	sn := &data.SignedSnapshot{
		Signatures: nil,
		Signed: data.Snapshot{
			Expires: time.Now().AddDate(1, 0, 0),
		},
	}
	assert.False(t, snapshotExpired(sn), "Snapshot should NOT have expired")
}

func TestGetSnapshotNotExists(t *testing.T) {
	store := storage.NewMemStorage()
	_, repo, crypto, err := testutils.EmptyRepo("gun")
	assert.NoError(t, err)

	// store the root, so we know that the failure to create a snapshot is due to the
	// lack of previous snapshot
	rootJSON, _ := json.Marshal(repo.Root)
	store.UpdateCurrent("gun", storage.MetaUpdate{Role: data.CanonicalRootRole, Version: 1, Data: rootJSON})

	_, err = GetOrCreateSnapshot("gun", store, crypto)
	assert.Error(t, err)
}

func TestGetSnapshotCurrValid(t *testing.T) {
	store := storage.NewMemStorage()
	crypto := signed.NewEd25519()

	// create snapshot key
	k, err := crypto.Create(data.CanonicalSnapshotRole, data.ED25519Key)
	assert.NoError(t, err)
	assert.NoError(t, store.AddKey("gun", data.CanonicalSnapshotRole, k, time.Now().AddDate(1, 1, 1)))

	newData := []byte{2}
	currMeta, err := data.NewFileMeta(bytes.NewReader(newData), "sha256")
	assert.NoError(t, err)

	snapshot := &data.SignedSnapshot{
		Signed: data.Snapshot{
			Expires: data.DefaultExpires(data.CanonicalSnapshotRole),
			Meta: data.Files{
				data.CanonicalRootRole: currMeta,
			},
		},
	}
	snapJSON, _ := json.Marshal(snapshot)

	// test when db is missing the role data
	store.UpdateCurrent("gun", storage.MetaUpdate{Role: "snapshot", Version: 0, Data: snapJSON})
	_, err = GetOrCreateSnapshot("gun", store, crypto)
	assert.NoError(t, err)

	// test when db has the role data
	store.UpdateCurrent("gun", storage.MetaUpdate{Role: "root", Version: 0, Data: newData})
	_, err = GetOrCreateSnapshot("gun", store, crypto)
	assert.NoError(t, err)

	// test when db role data is expired
	store.UpdateCurrent("gun", storage.MetaUpdate{Role: "root", Version: 1, Data: []byte{3}})
	_, err = GetOrCreateSnapshot("gun", store, crypto)
	assert.NoError(t, err)
}

// If the current snapshot is expired, GetOrCreateSnapshot will produce a new one
func TestGetSnapshotCurrExpired(t *testing.T) {
	store := storage.NewMemStorage()
	_, repo, crypto, err := testutils.EmptyRepo("gun")
	assert.NoError(t, err)

	// store the root and snapshot, since both are required to sign a new snapshot
	rootJSON, _ := json.Marshal(repo.Root)
	store.UpdateCurrent("gun",
		storage.MetaUpdate{Role: data.CanonicalRootRole, Version: repo.Root.Signed.Version, Data: rootJSON})

	repo.Snapshot.Signed.Expires = time.Time{}
	snapJSON, _ := json.Marshal(repo.Snapshot)
	store.UpdateCurrent("gun",
		storage.MetaUpdate{Role: data.CanonicalSnapshotRole, Version: repo.Snapshot.Signed.Version, Data: snapJSON})

	_, err = GetOrCreateSnapshot("gun", store, crypto)
	assert.NoError(t, err)
}

// If the current snapshot is corrupted, GetOrCreateSnapshot cannot produce a new one
func TestGetSnapshotCurrCorrupt(t *testing.T) {
	store := storage.NewMemStorage()
	_, repo, crypto, err := testutils.EmptyRepo("gun")
	assert.NoError(t, err)

	// store the root, so we know that the failure to create a snapshot is due to the
	// lack of previous snapshot
	rootJSON, _ := json.Marshal(repo.Root)
	store.UpdateCurrent("gun",
		storage.MetaUpdate{Role: data.CanonicalRootRole, Version: repo.Root.Signed.Version, Data: rootJSON})

	snapshot := &data.SignedSnapshot{}
	snapJSON, _ := json.Marshal(snapshot)

	store.UpdateCurrent("gun",
		storage.MetaUpdate{Role: data.CanonicalSnapshotRole, Version: repo.Snapshot.Signed.Version,
			Data: snapJSON[1:]})
	_, err = GetOrCreateSnapshot("gun", store, crypto)
	assert.Error(t, err)
}

func TestCreateSnapshotNoKeyInCrypto(t *testing.T) {
	store := storage.NewMemStorage()
	_, repo, _, err := testutils.EmptyRepo("gun")
	assert.NoError(t, err)

	// store the root and snapshot, so we know that the failure to create a snapshot is
	// not due to the lack of root or lack of previous snapshot
	rootJSON, _ := json.Marshal(repo.Root)
	store.UpdateCurrent("gun",
		storage.MetaUpdate{Role: data.CanonicalRootRole, Version: repo.Root.Signed.Version, Data: rootJSON})

	// expire the snapshot in order to force it to re-sign
	repo.Snapshot.Signed.Expires = time.Time{}
	snapJSON, _ := json.Marshal(repo.Snapshot)
	store.UpdateCurrent("gun",
		storage.MetaUpdate{Role: data.CanonicalSnapshotRole, Version: repo.Snapshot.Signed.Version, Data: snapJSON})

	// pass it a new cryptoservice without the key
	_, err = GetOrCreateSnapshot("gun", store, signed.NewEd25519())
	assert.Error(t, err)
}
