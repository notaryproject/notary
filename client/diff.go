package client

import (
	"errors"
	"encoding/hex"
	"encoding/json"
	"net/http"

	"github.com/sirupsen/logrus"

	store "github.com/theupdateframework/notary/storage"
	"github.com/theupdateframework/notary/trustpinning"
	"github.com/theupdateframework/notary/tuf"
	"github.com/theupdateframework/notary/tuf/data"
	"github.com/theupdateframework/notary/tuf/utils"
)

// Diff represents the difference between two versions of the same repo
type Diff struct {}

// NewDiff returns the different between two versions of the same TUF repo
func NewDiff(gun data.GUN, baseURL string, rt http.RoundTripper, hash1, hash2 string) (Diff, error) {
	remoteStore, err := getRemoteStore(baseURL, gun, rt)
	if err != nil {
		// baseURL is syntactically invalid
		return Diff{}, err
	}
	
	first, err := setupClient(gun, hash1, remoteStore)
	if err != nil {
		return Diff{}, err
	}
	second, err := setupClient(gun, hash2, remoteStore)
	if err != nil {
		return Diff{}, err
	}

	// download the relevant versions of the repos now that we have the bootstrapping
	// setup. 
	// N.B. lower case update will fail out immediately rather than attempting
	//      to download a newer root
	err = first.update()
	if err != nil {
		return Diff{}, errors.New("failed to download repo version " + hash1)
	}
	err = second.update()
	if err != nil {
		return Diff{}, errors.New("failed to download repo version " + hash2)
	}

	repoFirst, _, err := first.newBuilder.Finish()
	repoSecond, _, err := second.newBuilder.Finish()

	return diff(repoFirst, repoSecond)
}

func setupClient(gun data.GUN, tsHash string, remote store.RemoteStore) (*tufClient, error) {
	hashBytes, err := hex.DecodeString(tsHash)
	if err != nil {
		return nil, err
	}

	root, err := findRoot(hashBytes, remote)

	oldBuilder := tuf.NewRepoBuilder(gun, nil, trustpinning.TrustPinConfig{})
	if err := oldBuilder.Load(data.CanonicalRootRole, root, 0, true); err != nil {
		return nil, err
	}
	newBuilder := tuf.NewRepoBuilder(gun, nil, trustpinning.TrustPinConfig{})
	if err := newBuilder.Load(data.CanonicalRootRole, root, 0, true); err != nil {
		return nil, err
	}

	tClient := newTufClient(oldBuilder, newBuilder, remote, store.NewMemoryStore(nil))
	tClient.tsChecksum = hashBytes
	return tClient, nil
}

func diff(first, second *tuf.Repo) (Diff, error) {
	return Diff{}, nil
}

// walk ts -> snap -> root.
// TODO: verify checksums. Other verification will be done when we
// eventually call tufClient.update()
func findRoot(tsChecksum []byte, remote store.RemoteStore) ([]byte, error) {
	consistentTSName := utils.ConsistentName(
		data.CanonicalTimestampRole.String(),
		tsChecksum,
	)

	raw, err := remote.GetSized(consistentTSName, 0)
	if err != nil {
		logrus.Debugf("error downloading %s: %s", consistentTSName, err)
		return nil, err
	}

	ts := &data.SignedTimestamp{}
	err = json.Unmarshal(raw, ts)
	if err != nil {
		logrus.Debugf("error parsing %s: %s", consistentTSName, err)
		return nil, err
	}

	snapshotMeta := ts.Signed.Meta[data.CanonicalSnapshotRole.String()] 
	consistentSnapshotName := utils.ConsistentName(
		data.CanonicalSnapshotRole.String(),
		snapshotMeta.Hashes["sha256"],
	)
	raw, err = remote.GetSized(consistentSnapshotName, snapshotMeta.Length)
	if err != nil {
		logrus.Debugf("error downloading %s: %s", consistentSnapshotName, err)
		return nil, err
	}

	snap := &data.SignedSnapshot{}
	err = json.Unmarshal(raw, snap)
	if err != nil {
		logrus.Debugf("error parsing %s: %s", consistentSnapshotName, err)
		return nil, err
	}

	rootMeta := snap.Signed.Meta[data.CanonicalRootRole.String()] 
	consistentRootName := utils.ConsistentName(
		data.CanonicalRootRole.String(),
		rootMeta.Hashes["sha256"],
	)
	raw, err = remote.GetSized(consistentRootName, rootMeta.Length)
	if err != nil {
		logrus.Debugf("error downloading %s: %s", consistentRootName, err)
		return nil, err
	}

	return raw, err
}