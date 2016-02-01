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
	crypto := signed.NewEd25519()

	_, err := GetOrCreateSnapshot("gun", store, crypto)
	assert.Error(t, err)
}

func TestGetSnapshotCurrValid(t *testing.T) {
	store := storage.NewMemStorage()
	crypto := signed.NewEd25519()

	_, err := GetOrCreateSnapshotKey("gun", store, crypto, data.ED25519Key)

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

func TestGetSnapshotCurrExpired(t *testing.T) {
	store := storage.NewMemStorage()
	crypto := signed.NewEd25519()

	_, err := GetOrCreateSnapshotKey("gun", store, crypto, data.ED25519Key)

	snapshot := &data.SignedSnapshot{}
	snapJSON, _ := json.Marshal(snapshot)

	store.UpdateCurrent("gun", storage.MetaUpdate{Role: "snapshot", Version: 0, Data: snapJSON})
	_, err = GetOrCreateSnapshot("gun", store, crypto)
	assert.NoError(t, err)
}

func TestGetSnapshotCurrCorrupt(t *testing.T) {
	store := storage.NewMemStorage()
	crypto := signed.NewEd25519()

	_, err := GetOrCreateSnapshotKey("gun", store, crypto, data.ED25519Key)

	snapshot := &data.SignedSnapshot{}
	snapJSON, _ := json.Marshal(snapshot)

	store.UpdateCurrent("gun", storage.MetaUpdate{Role: "snapshot", Version: 0, Data: snapJSON[1:]})
	_, err = GetOrCreateSnapshot("gun", store, crypto)
	assert.Error(t, err)
}

func TestCreateSnapshotNoKeyInStorage(t *testing.T) {
	store := storage.NewMemStorage()
	crypto := signed.NewEd25519()

	_, _, err := createSnapshot("gun", nil, store, crypto)
	assert.Error(t, err)
}

func TestCreateSnapshotNoKeyInCrypto(t *testing.T) {
	store := storage.NewMemStorage()
	crypto := signed.NewEd25519()

	_, err := GetOrCreateSnapshotKey("gun", store, crypto, data.ED25519Key)

	// reset crypto so the store has the key but crypto doesn't
	crypto = signed.NewEd25519()

	_, _, err = createSnapshot("gun", &data.SignedSnapshot{}, store, crypto)
	assert.Error(t, err)
}
