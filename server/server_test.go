package server

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "github.com/docker/distribution/registry/auth/silly"
	"github.com/stretchr/testify/require"
	"github.com/theupdateframework/notary"
	"github.com/theupdateframework/notary/server/storage"
	store "github.com/theupdateframework/notary/storage"
	"github.com/theupdateframework/notary/tuf/data"
	"github.com/theupdateframework/notary/tuf/signed"
	"github.com/theupdateframework/notary/tuf/testutils"
	tufutils "github.com/theupdateframework/notary/tuf/utils"
	"github.com/theupdateframework/notary/utils"
	"golang.org/x/net/context"
)

func TestRunBadAddr(t *testing.T) {
	done := make(chan bool)
	var err error

	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()

	go func() {
		err = Run(
			context.Background(),
			Config{
				Addr:  "testAddr",
				Trust: signed.NewEd25519(),
			},
		)
		done <- true
	}()

	select {
	case <-done:
		require.Error(t, err, "Passed bad addr, Run should have failed")
	case <-timer.C:
		require.Fail(t, "Passed bad addr, Run should have failed")
	}
}

func TestRepoPrefixMatches(t *testing.T) {
	var gun data.GUN = "docker.io/notary"
	meta, cs, err := testutils.NewRepoMetadata(gun)
	require.NoError(t, err)

	ctx := context.WithValue(context.Background(), notary.CtxKeyMetaStore, storage.NewMemStorage())
	ctx = context.WithValue(ctx, notary.CtxKeyKeyAlgo, data.ED25519Key)

	snChecksumBytes := sha256.Sum256(meta[data.CanonicalSnapshotRole])

	// successful gets
	handler := RootHandler(ctx, nil, cs, nil, nil, []string{"docker.io"})
	ts := httptest.NewServer(handler)

	url := fmt.Sprintf("%s/v2/%s/_trust/tuf/", ts.URL, gun)
	uploader, err := store.NewHTTPStore(url, "", "json", "key", http.DefaultTransport)
	require.NoError(t, err)

	// uploading is cool
	require.NoError(t, uploader.SetMulti(data.MetadataRoleMapToStringMap(meta)))
	// getting is cool
	_, err = uploader.GetSized(data.CanonicalSnapshotRole.String(), notary.MaxDownloadSize)
	require.NoError(t, err)

	_, err = uploader.GetSized(
		tufutils.ConsistentName(data.CanonicalSnapshotRole.String(), snChecksumBytes[:]), notary.MaxDownloadSize)
	require.NoError(t, err)

	_, err = uploader.GetKey(data.CanonicalTimestampRole)
	require.NoError(t, err)

	// the httpstore doesn't actually delete all, so we do it manually
	req, err := http.NewRequest("DELETE", url, nil)
	require.NoError(t, err)
	res, err := http.DefaultTransport.RoundTrip(req)
	require.NoError(t, err)
	defer res.Body.Close()
	require.Equal(t, http.StatusOK, res.StatusCode)
}

func TestRepoPrefixDoesNotMatch(t *testing.T) {
	var gun data.GUN = "docker.io/notary"
	meta, cs, err := testutils.NewRepoMetadata(gun)
	require.NoError(t, err)
	s := storage.NewMemStorage()

	ctx := context.WithValue(context.Background(), notary.CtxKeyMetaStore, s)
	ctx = context.WithValue(ctx, notary.CtxKeyKeyAlgo, data.ED25519Key)

	snChecksumBytes := sha256.Sum256(meta[data.CanonicalSnapshotRole])

	// successful gets
	handler := RootHandler(ctx, nil, cs, nil, nil, []string{"nope"})
	ts := httptest.NewServer(handler)

	url := fmt.Sprintf("%s/v2/%s/_trust/tuf/", ts.URL, gun)
	uploader, err := store.NewHTTPStore(url, "", "json", "key", http.DefaultTransport)
	require.NoError(t, err)

	require.Error(t, uploader.SetMulti(data.MetadataRoleMapToStringMap(meta)))

	// update the storage so we don't fail just because the metadata is missing
	for _, roleName := range data.BaseRoles {
		require.NoError(t, s.UpdateCurrent(gun, storage.MetaUpdate{
			Role:    roleName,
			Data:    meta[roleName],
			Version: 1,
		}))
	}

	_, err = uploader.GetSized(data.CanonicalSnapshotRole.String(), notary.MaxDownloadSize)
	require.Error(t, err)

	_, err = uploader.GetSized(
		tufutils.ConsistentName(data.CanonicalSnapshotRole.String(), snChecksumBytes[:]), notary.MaxDownloadSize)
	require.Error(t, err)

	_, err = uploader.GetKey(data.CanonicalTimestampRole)
	require.Error(t, err)

	// the httpstore doesn't actually delete all, so we do it manually
	req, err := http.NewRequest("DELETE", url, nil)
	require.NoError(t, err)
	res, err := http.DefaultTransport.RoundTrip(req)
	require.NoError(t, err)
	defer res.Body.Close()
	require.Equal(t, http.StatusNotFound, res.StatusCode)
}

func TestMetricsEndpoint(t *testing.T) {
	handler := RootHandler(context.Background(), nil, signed.NewEd25519(),
		nil, nil, nil)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	res, err := http.Get(ts.URL + "/metrics")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode)
}

// GetKeys supports only the timestamp and snapshot key endpoints
func TestGetKeysEndpoint(t *testing.T) {
	ctx := context.WithValue(
		context.Background(), notary.CtxKeyMetaStore, storage.NewMemStorage())
	ctx = context.WithValue(ctx, notary.CtxKeyKeyAlgo, data.ED25519Key)

	handler := RootHandler(ctx, nil, signed.NewEd25519(), nil, nil, nil)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	rolesToStatus := map[data.RoleName]int{
		data.CanonicalTimestampRole: http.StatusOK,
		data.CanonicalSnapshotRole:  http.StatusOK,
		data.CanonicalTargetsRole:   http.StatusNotFound,
		data.CanonicalRootRole:      http.StatusNotFound,
		"somerandomrole":            http.StatusNotFound,
	}

	for role, expectedStatus := range rolesToStatus {
		res, err := http.Get(
			fmt.Sprintf("%s/v2/gun/_trust/tuf/%s.key", ts.URL, role))
		require.NoError(t, err)
		require.Equal(t, expectedStatus, res.StatusCode)
	}
}

// This just checks the URL routing is working correctly and cache headers are set correctly.
// More detailed tests for this path including negative
// tests are located in /server/handlers/
func TestGetRoleByHash(t *testing.T) {
	store := storage.NewMemStorage()

	ts := data.SignedTimestamp{
		Signatures: make([]data.Signature, 0),
		Signed: data.Timestamp{
			SignedCommon: data.SignedCommon{
				Type:    data.TUFTypes[data.CanonicalTimestampRole],
				Version: 1,
				Expires: data.DefaultExpires(data.CanonicalTimestampRole),
			},
		},
	}
	j, err := json.Marshal(&ts)
	require.NoError(t, err)
	store.UpdateCurrent("gun", storage.MetaUpdate{
		Role:    data.CanonicalTimestampRole,
		Version: 1,
		Data:    j,
	})
	checksumBytes := sha256.Sum256(j)
	checksum := hex.EncodeToString(checksumBytes[:])

	// create and add a newer timestamp. We're going to try and request
	// the older version we created above.
	ts = data.SignedTimestamp{
		Signatures: make([]data.Signature, 0),
		Signed: data.Timestamp{
			SignedCommon: data.SignedCommon{
				Type:    data.TUFTypes[data.CanonicalTimestampRole],
				Version: 2,
				Expires: data.DefaultExpires(data.CanonicalTimestampRole),
			},
		},
	}
	newTS, err := json.Marshal(&ts)
	require.NoError(t, err)
	store.UpdateCurrent("gun", storage.MetaUpdate{
		Role:    data.CanonicalTimestampRole,
		Version: 1,
		Data:    newTS,
	})

	ctx := context.WithValue(
		context.Background(), notary.CtxKeyMetaStore, store)

	ctx = context.WithValue(ctx, notary.CtxKeyKeyAlgo, data.ED25519Key)

	ccc := utils.NewCacheControlConfig(10, false)
	handler := RootHandler(ctx, nil, signed.NewEd25519(), ccc, ccc, nil)
	serv := httptest.NewServer(handler)
	defer serv.Close()

	res, err := http.Get(fmt.Sprintf(
		"%s/v2/gun/_trust/tuf/%s.%s.json",
		serv.URL,
		data.CanonicalTimestampRole,
		checksum,
	))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode)
	// if content is equal, checksums are guaranteed to be equal
	verifyGetResponse(t, res, j)
}

// This just checks the URL routing is working correctly and cache headers are set correctly.
// More detailed tests for this path including negative
// tests are located in /server/handlers/
func TestGetRoleByVersion(t *testing.T) {
	store := storage.NewMemStorage()

	ts := data.SignedTimestamp{
		Signatures: make([]data.Signature, 0),
		Signed: data.Timestamp{
			SignedCommon: data.SignedCommon{
				Type:    data.TUFTypes[data.CanonicalTimestampRole],
				Version: 1,
				Expires: data.DefaultExpires(data.CanonicalTimestampRole),
			},
		},
	}
	j, err := json.Marshal(&ts)
	require.NoError(t, err)
	store.UpdateCurrent("gun", storage.MetaUpdate{
		Role:    data.CanonicalTimestampRole,
		Version: 1,
		Data:    j,
	})

	// create and add a newer timestamp. We're going to try and request
	// the older version we created above.
	ts = data.SignedTimestamp{
		Signatures: make([]data.Signature, 0),
		Signed: data.Timestamp{
			SignedCommon: data.SignedCommon{
				Type:    data.TUFTypes[data.CanonicalTimestampRole],
				Version: 2,
				Expires: data.DefaultExpires(data.CanonicalTimestampRole),
			},
		},
	}
	newTS, err := json.Marshal(&ts)
	require.NoError(t, err)
	store.UpdateCurrent("gun", storage.MetaUpdate{
		Role:    data.CanonicalTimestampRole,
		Version: 1,
		Data:    newTS,
	})

	ctx := context.WithValue(
		context.Background(), notary.CtxKeyMetaStore, store)

	ctx = context.WithValue(ctx, notary.CtxKeyKeyAlgo, data.ED25519Key)

	ccc := utils.NewCacheControlConfig(10, false)
	handler := RootHandler(ctx, nil, signed.NewEd25519(), ccc, ccc, nil)
	serv := httptest.NewServer(handler)
	defer serv.Close()

	res, err := http.Get(fmt.Sprintf(
		"%s/v2/gun/_trust/tuf/%d.%s.json",
		serv.URL,
		1,
		data.CanonicalTimestampRole,
	))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode)
	// if content is equal, checksums are guaranteed to be equal
	verifyGetResponse(t, res, j)
}

// This just checks the URL routing is working correctly and cache headers are set correctly.
// More detailed tests for this path including negative
// tests are located in /server/handlers/
func TestGetCurrentRole(t *testing.T) {
	store := storage.NewMemStorage()
	metadata, _, err := testutils.NewRepoMetadata("gun")
	require.NoError(t, err)

	// need both the snapshot and the timestamp, because when getting the current
	// timestamp the server checks to see if it's out of date (there's a new snapshot)
	// and if so, generates a new one
	store.UpdateCurrent("gun", storage.MetaUpdate{
		Role:    data.CanonicalSnapshotRole,
		Version: 1,
		Data:    metadata[data.CanonicalSnapshotRole],
	})
	store.UpdateCurrent("gun", storage.MetaUpdate{
		Role:    data.CanonicalTimestampRole,
		Version: 1,
		Data:    metadata[data.CanonicalTimestampRole],
	})

	ctx := context.WithValue(
		context.Background(), notary.CtxKeyMetaStore, store)

	ctx = context.WithValue(ctx, notary.CtxKeyKeyAlgo, data.ED25519Key)

	ccc := utils.NewCacheControlConfig(10, false)
	handler := RootHandler(ctx, nil, signed.NewEd25519(), ccc, ccc, nil)
	serv := httptest.NewServer(handler)
	defer serv.Close()

	res, err := http.Get(fmt.Sprintf(
		"%s/v2/gun/_trust/tuf/%s.json",
		serv.URL,
		data.CanonicalTimestampRole,
	))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode)
	verifyGetResponse(t, res, metadata[data.CanonicalTimestampRole])
}

// Verifies that the body is as expected  and that there are cache control headers
func verifyGetResponse(t *testing.T, r *http.Response, expectedBytes []byte) {
	body, err := ioutil.ReadAll(r.Body)
	require.NoError(t, err)
	require.True(t, bytes.Equal(expectedBytes, body))

	require.NotEqual(t, "", r.Header.Get("Cache-Control"))
	require.NotEqual(t, "", r.Header.Get("Last-Modified"))
	require.Equal(t, "", r.Header.Get("Pragma"))
}

// RotateKey supports only timestamp and snapshot key rotation
func TestRotateKeyEndpoint(t *testing.T) {
	ctx := context.WithValue(
		context.Background(), notary.CtxKeyMetaStore, storage.NewMemStorage())
	ctx = context.WithValue(ctx, notary.CtxKeyKeyAlgo, data.ED25519Key)

	ccc := utils.NewCacheControlConfig(10, false)
	handler := RootHandler(ctx, nil, signed.NewEd25519(), ccc, ccc, nil)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	rolesToStatus := map[data.RoleName]int{
		data.CanonicalTimestampRole: http.StatusOK,
		data.CanonicalSnapshotRole:  http.StatusOK,
		data.CanonicalTargetsRole:   http.StatusNotFound,
		data.CanonicalRootRole:      http.StatusNotFound,
		"targets/delegation":        http.StatusNotFound,
		"somerandomrole":            http.StatusNotFound,
	}
	var buf bytes.Buffer
	for role, expectedStatus := range rolesToStatus {
		res, err := http.Post(
			fmt.Sprintf("%s/v2/gun/_trust/tuf/%s.key", ts.URL, role),
			"text/plain", &buf)
		require.NoError(t, err)
		require.Equal(t, expectedStatus, res.StatusCode)
	}
}
