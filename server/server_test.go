package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	_ "github.com/docker/distribution/registry/auth/silly"
	"github.com/docker/notary/server/handlers"
	"github.com/docker/notary/server/storage"
	"github.com/docker/notary/tuf/data"
	"github.com/docker/notary/tuf/signed"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func TestRunBadAddr(t *testing.T) {
	err := Run(
		context.Background(),
		"testAddr",
		nil,
		signed.NewEd25519(),
		"",
		nil,
	)
	assert.Error(t, err, "Passed bad addr, Run should have failed")
}

func TestRunReservedPort(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())

	err := Run(
		ctx,
		"localhost:80",
		nil,
		signed.NewEd25519(),
		"",
		nil,
	)

	assert.Error(t, err)
	assert.IsType(t, &net.OpError{}, err)
	assert.True(
		t,
		strings.Contains(err.Error(), "bind: permission denied"),
		"Received unexpected err: %s",
		err.Error(),
	)
}

func TestMetricsEndpoint(t *testing.T) {
	handler := RootHandler(nil, context.Background(), signed.NewEd25519())
	ts := httptest.NewServer(handler)
	defer ts.Close()

	res, err := http.Get(ts.URL + "/_notary_server/metrics")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)
}

// GetKeys supports only the timestamp and snapshot key endpoints
func TestGetKeysEndpoint(t *testing.T) {
	ctx := context.WithValue(
		context.Background(), "metaStore", storage.NewMemStorage())
	ctx = context.WithValue(ctx, "keyAlgorithm", data.ED25519Key)

	handler := RootHandler(nil, ctx, signed.NewEd25519())
	ts := httptest.NewServer(handler)
	defer ts.Close()

	rolesToStatus := map[string]int{
		data.CanonicalTimestampRole: http.StatusOK,
		data.CanonicalSnapshotRole:  http.StatusOK,
		data.CanonicalTargetsRole:   http.StatusNotFound,
		data.CanonicalRootRole:      http.StatusNotFound,
		"somerandomrole":            http.StatusNotFound,
	}

	for role, expectedStatus := range rolesToStatus {
		res, err := http.Get(
			fmt.Sprintf("%s/v2/gun/_trust/tuf/%s.key", ts.URL, role))
		assert.NoError(t, err)
		assert.Equal(t, expectedStatus, res.StatusCode)
	}
}

// This just checks the URL routing is working correctly.
// More detailed tests for this path including negative
// tests are located in /server/handlers/
func TestGetRoleByHash(t *testing.T) {
	store := storage.NewMemStorage()

	ts := data.SignedTimestamp{
		Signatures: make([]data.Signature, 0),
		Signed: data.Timestamp{
			Type:    data.TUFTypes["timestamp"],
			Version: 1,
			Expires: data.DefaultExpires("timestamp"),
		},
	}
	j, err := json.Marshal(&ts)
	assert.NoError(t, err)
	update := storage.MetaUpdate{
		Role:    data.CanonicalTimestampRole,
		Version: 1,
		Data:    j,
	}
	checksumBytes := sha256.Sum256(j)
	checksum := hex.EncodeToString(checksumBytes[:])

	store.UpdateCurrent("gun", update)

	// create and add a newer timestamp. We're going to try and request
	// the older version we created above.
	ts = data.SignedTimestamp{
		Signatures: make([]data.Signature, 0),
		Signed: data.Timestamp{
			Type:    data.TUFTypes["timestamp"],
			Version: 2,
			Expires: data.DefaultExpires("timestamp"),
		},
	}
	newJ, err := json.Marshal(&ts)
	assert.NoError(t, err)
	update = storage.MetaUpdate{
		Role:    data.CanonicalTimestampRole,
		Version: 2,
		Data:    newJ,
	}
	store.UpdateCurrent("gun", update)

	ctx := context.WithValue(
		context.Background(), "metaStore", store)

	ctx = context.WithValue(ctx, "keyAlgorithm", data.ED25519Key)

	handler := RootHandler(nil, ctx, signed.NewEd25519())
	serv := httptest.NewServer(handler)
	defer serv.Close()

	res, err := http.Get(fmt.Sprintf(
		"%s/v2/gun/_trust/tuf/%s.%s.json",
		serv.URL,
		data.CanonicalTimestampRole,
		checksum,
	))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)

	body, err := ioutil.ReadAll(res.Body)
	assert.NoError(t, err)
	defer res.Body.Close()
	// if content is equal, checksums are guaranteed to be equal
	assert.EqualValues(t, j, body)
}

func TestListVersions(t *testing.T) {
	tempBaseDir, err := ioutil.TempDir("", "notary-test-")
	_, store := storage.SetUpSQLite(t, tempBaseDir)
	defer os.RemoveAll(tempBaseDir)

	ts := data.SignedTimestamp{
		Signatures: make([]data.Signature, 0),
		Signed: data.Timestamp{
			Type:    data.TUFTypes["timestamp"],
			Version: 1,
			Expires: data.DefaultExpires("timestamp"),
		},
	}
	tsJSON1, err := json.Marshal(&ts)
	assert.NoError(t, err)
	update := storage.MetaUpdate{
		Role:    data.CanonicalTimestampRole,
		Version: 1,
		Data:    tsJSON1,
	}
	store.UpdateCurrent("gun", update)

	// create and add a newer timestamp. We're going to try and request
	// the older version we created above.
	ts = data.SignedTimestamp{
		Signatures: make([]data.Signature, 0),
		Signed: data.Timestamp{
			Type:    data.TUFTypes["timestamp"],
			Version: 2,
			Expires: data.DefaultExpires("timestamp"),
		},
	}
	tsJSON2, err := json.Marshal(&ts)
	assert.NoError(t, err)
	update = storage.MetaUpdate{
		Role:    data.CanonicalTimestampRole,
		Version: 2,
		Data:    tsJSON2,
	}
	store.UpdateCurrent("gun", update)
	checksumBytes := sha256.Sum256(tsJSON2)
	checksum := hex.EncodeToString(checksumBytes[:])

	ctx := context.WithValue(
		context.Background(), "metaStore", store)

	ctx = context.WithValue(ctx, "keyAlgorithm", data.ED25519Key)

	handler := RootHandler(nil, ctx, signed.NewEd25519())
	serv := httptest.NewServer(handler)
	defer serv.Close()

	res, err := http.Get(fmt.Sprintf(
		"%s/v2/gun/_trust/tuf/versions/%s.json",
		serv.URL,
		data.CanonicalTimestampRole,
	))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)

	body, err := ioutil.ReadAll(res.Body)
	assert.NoError(t, err)
	defer res.Body.Close()

	vers := handlers.VersionResponse{}
	err = json.Unmarshal(body, &vers)
	assert.NoError(t, err)
	assert.Len(t, vers.Versions, 2)
	assert.EqualValues(t, tsJSON2, vers.Versions[0])
	assert.EqualValues(t, tsJSON1, vers.Versions[1])

	res, err = http.Get(fmt.Sprintf(
		"%s/v2/gun/_trust/tuf/versions/%s.json?start=%s&number=1",
		serv.URL,
		data.CanonicalTimestampRole,
		checksum,
	))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)

	body, err = ioutil.ReadAll(res.Body)
	assert.NoError(t, err)
	defer res.Body.Close()

	vers = handlers.VersionResponse{}
	err = json.Unmarshal(body, &vers)
	assert.NoError(t, err)
	assert.Len(t, vers.Versions, 1)
	assert.EqualValues(t, tsJSON1, vers.Versions[0])
}
