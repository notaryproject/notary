package timestamp

import (
	"bytes"

	"github.com/docker/go/canonical/json"
	"github.com/docker/notary/tuf"
	"github.com/docker/notary/tuf/data"
	"github.com/docker/notary/tuf/keys"
	"github.com/docker/notary/tuf/signed"

	"github.com/Sirupsen/logrus"
	"github.com/docker/notary/server/storage"
)

// GetOrCreateTimestamp returns the current timestamp for the gun. This may mean
// a new timestamp is generated either because none exists, or because the current
// one has expired. Once generated, the timestamp is saved in the store.
func GetOrCreateTimestamp(gun string, store storage.MetaStore, cryptoService signed.CryptoService) ([]byte, error) {
	snapshot, err := store.GetCurrent(gun, "snapshot")
	if err != nil {
		return nil, err
	}
	d, err := store.GetCurrent(gun, "timestamp")
	if err != nil {
		if _, ok := err.(storage.ErrNotFound); !ok {
			logrus.Error("error retrieving timestamp: ", err.Error())
			return nil, err
		}
		logrus.Debug("No timestamp found, will proceed to create first timestamp")
	}
	var ts *data.SignedTimestamp
	if d != nil {
		ts = &data.SignedTimestamp{}
		err := json.Unmarshal(d, ts)
		if err != nil {
			logrus.Error("Failed to unmarshal existing timestamp")
			return nil, err
		}
		if !timestampExpired(ts) && !snapshotExpired(ts, snapshot) {
			return d, nil
		}
	}
	sgnd, version, err := CreateTimestamp(gun, ts, snapshot, store, cryptoService)
	if err != nil {
		logrus.Error("Failed to create a new timestamp")
		return nil, err
	}
	out, err := json.Marshal(sgnd)
	if err != nil {
		logrus.Error("Failed to marshal new timestamp")
		return nil, err
	}
	err = store.UpdateCurrent(gun, storage.MetaUpdate{Role: "timestamp", Version: version, Data: out})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// timestampExpired compares the current time to the expiry time of the timestamp
func timestampExpired(ts *data.SignedTimestamp) bool {
	return signed.IsExpired(ts.Signed.Expires)
}

func snapshotExpired(ts *data.SignedTimestamp, snapshot []byte) bool {
	meta, err := data.NewFileMeta(bytes.NewReader(snapshot), "sha256")
	if err != nil {
		// if we can't generate FileMeta from the current snapshot, we should
		// continue to serve the old timestamp if it isn't time expired
		// because we won't be able to generate a new one.
		return false
	}
	hash := meta.Hashes["sha256"]
	return !bytes.Equal(hash, ts.Signed.Meta["snapshot"].Hashes["sha256"])
}

// CreateTimestamp creates a new timestamp. If a prev timestamp is provided, it
// is assumed this is the immediately previous one, and the new one will have a
// version number one higher than prev. The store is used to lookup the current
// snapshot, this function does not save the newly generated timestamp.
func CreateTimestamp(gun string, prev *data.SignedTimestamp, snapshot []byte, store storage.MetaStore, cryptoService signed.CryptoService) (*data.Signed, int, error) {
	kdb := keys.NewDB()
	repo := tuf.NewRepo(kdb, cryptoService)

	// load the current root to ensure we use the correct timestamp key.
	root, err := store.GetCurrent(gun, "root")
	r := &data.SignedRoot{}
	err = json.Unmarshal(root, r)
	if err != nil {
		// couldn't parse root
		return nil, 0, err
	}
	repo.SetRoot(r)

	// load snapshot so we can include it in timestamp
	sn := &data.SignedSnapshot{}
	err = json.Unmarshal(snapshot, sn)
	if err != nil {
		// couldn't parse snapshot
		return nil, 0, err
	}
	repo.SetSnapshot(sn)

	if prev == nil {
		// no previous timestamp: generate first timestamp
		repo.InitTimestamp()
	} else {
		// set repo timestamp to previous timestamp to use as base for
		// generating new one
		repo.SetTimestamp(prev)
	}

	out, err := repo.SignTimestamp(
		data.DefaultExpires(data.CanonicalTimestampRole),
	)
	if err != nil {
		return nil, 0, err
	}
	return out, repo.Timestamp.Signed.Version, nil
}
