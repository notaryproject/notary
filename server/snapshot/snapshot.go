package snapshot

import (
	"encoding/json"

	"github.com/Sirupsen/logrus"

	"github.com/docker/notary/server/storage"
	"github.com/docker/notary/tuf"
	"github.com/docker/notary/tuf/data"
	"github.com/docker/notary/tuf/keys"
	"github.com/docker/notary/tuf/signed"
)

// GetOrCreateSnapshot either returns the exisiting latest snapshot, or uses
// whatever the most recent snapshot is to create the next one, only updating
// the expiry time and version.
func GetOrCreateSnapshot(gun string, store storage.MetaStore, cryptoService signed.CryptoService) ([]byte, error) {
	d, err := store.GetCurrent(gun, data.CanonicalSnapshotRole)
	if err != nil {
		return nil, err
	}

	sn := &data.SignedSnapshot{}
	if d != nil {
		err := json.Unmarshal(d, sn)
		if err != nil {
			logrus.Error("Failed to unmarshal existing snapshot")
			return nil, err
		}

		if !snapshotExpired(sn) {
			return d, nil
		}
	}

	metaUpdate, err := createSnapshot(gun, sn, store, cryptoService)
	if err != nil {
		logrus.Error("Failed to create a new snapshot")
		return nil, err
	}
	if err = store.UpdateCurrent(gun, *metaUpdate); err != nil {
		return nil, err
	}
	return metaUpdate.Data, nil
}

// snapshotExpired simply checks if the snapshot is past its expiry time
func snapshotExpired(sn *data.SignedSnapshot) bool {
	return signed.IsExpired(sn.Signed.Expires)
}

// createSnapshot uses an existing snapshot to create a new one.
// Important things to be aware of:
//   - It requires that a snapshot already exists. We create snapshots
//     on upload so there should always be an existing snapshot if this
//     gets called.
//   - It doesn't update what roles are present in the snapshot, as those
//     were validated during upload.
func createSnapshot(gun string, sn *data.SignedSnapshot, store storage.MetaStore, cryptoService signed.CryptoService) (
	*storage.MetaUpdate, error) {

	kdb := keys.NewDB()
	repo := tuf.NewRepo(kdb, cryptoService)

	// load the current root to ensure we use the correct timestamp key.
	root, err := store.GetCurrent(gun, data.CanonicalRootRole)
	if err != nil {
		return nil, err
	}
	r := &data.SignedRoot{}
	err = json.Unmarshal(root, r)
	if err != nil {
		// couldn't parse root
		return nil, err
	}
	repo.SetRoot(r)
	return NewSnapshotUpdate(sn, repo)
}

// NewSnapshotUpdate produces a new snapshot and returns it as a metadata update, given the
// previous snapshot and the TUF repo.
func NewSnapshotUpdate(prev *data.SignedSnapshot, repo *tuf.Repo) (*storage.MetaUpdate, error) {
	if prev != nil {
		repo.SetSnapshot(prev) // SetSnapshot never errors
	} else {
		// this will only occurr if no snapshot has ever been created for the repository
		if err := repo.InitSnapshot(); err != nil {
			return nil, err
		}
	}
	sgnd, err := repo.SignSnapshot(data.DefaultExpires(data.CanonicalSnapshotRole))
	if err != nil {
		return nil, err
	}
	sgndJSON, err := json.Marshal(sgnd)
	if err != nil {
		return nil, err
	}
	return &storage.MetaUpdate{
		Role:    data.CanonicalSnapshotRole,
		Version: repo.Snapshot.Signed.Version,
		Data:    sgndJSON,
	}, nil
}
