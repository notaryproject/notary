package storage

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/docker/go/canonical/json"
	"github.com/docker/notary/tuf/data"
	"github.com/docker/notary/tuf/validation"
	"github.com/stretchr/testify/require"
)

const testRoot = `{"signed":{"_type":"Root","consistent_snapshot":false,"expires":"2025-07-17T16:19:21.101698314-07:00","keys":{"1ca15c7f4b2b0c6efce202a545e7267152da28ab7c91590b3b60bdb4da723aad":{"keytype":"ecdsa","keyval":{"private":null,"public":"MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEb0720c99Cj6ZmuDlznEZ52NA6YpeY9Sj45z51XvPnG63Bi2RSBezMJlPzbSfP39mXKXqOJyT+z9BZhi3FVWczg=="}},"b1d6813b55442ecbfb1f4b40eb1fcdb4290e53434cfc9ba2da24c26c9143873b":{"keytype":"ecdsa-x509","keyval":{"private":null,"public":"LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJVekNCKzZBREFnRUNBaEFCWDNKLzkzaW8zbHcrZUsvNFhvSHhNQW9HQ0NxR1NNNDlCQU1DTUJFeER6QU4KQmdOVkJBTVRCbVY0Y0dseVpUQWVGdzB4TlRBM01qQXlNekU1TVRkYUZ3MHlOVEEzTVRjeU16RTVNVGRhTUJFeApEekFOQmdOVkJBTVRCbVY0Y0dseVpUQlpNQk1HQnlxR1NNNDlBZ0VHQ0NxR1NNNDlBd0VIQTBJQUJFTDhOTFhQCitreUJZYzhYY0FTMXB2S2l5MXRQUDlCZHJ1dEdrWlR3Z0dEYTM1THMzSUFXaWlrUmlPbGRuWmxVVEE5cG5JekoKOFlRQThhTjQ1TDQvUlplak5UQXpNQTRHQTFVZER3RUIvd1FFQXdJQW9EQVRCZ05WSFNVRUREQUtCZ2dyQmdFRgpCUWNEQXpBTUJnTlZIUk1CQWY4RUFqQUFNQW9HQ0NxR1NNNDlCQU1DQTBjQU1FUUNJRVJ1ZUVURG5xMlRqRFBmClhGRStqUFJqMEtqdXdEOG9HSmtoVGpMUDAycjhBaUI5cUNyL2ZqSXpJZ1NQcTJVSXZqR0hlYmZOYXh1QlpZZUUKYW8xNjd6dHNYZz09Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K"}},"fbddae7f25a6c23ca735b017206a849d4c89304a4d8de4dcc4b3d6f3eb22ce3b":{"keytype":"ecdsa","keyval":{"private":null,"public":"MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE/xS5fBHK2HKmlGcvAr06vwPITvmxWP4P3CMDCgY25iSaIiM21OiXA1/Uvo3Pa3xh5G3cwCtDvi+4FpflW2iB/w=="}},"fd75751f010c3442e23b3e3e99a1442a112f2f21038603cb8609d8b17c9e912a":{"keytype":"ed25519","keyval":{"private":null,"public":"rc+glN01m+q8jmX8SolGsjTfk6NMhUQTWyj10hjmne0="}}},"roles":{"root":{"keyids":["b1d6813b55442ecbfb1f4b40eb1fcdb4290e53434cfc9ba2da24c26c9143873b"],"threshold":1},"snapshot":{"keyids":["1ca15c7f4b2b0c6efce202a545e7267152da28ab7c91590b3b60bdb4da723aad"],"threshold":1},"targets":{"keyids":["fbddae7f25a6c23ca735b017206a849d4c89304a4d8de4dcc4b3d6f3eb22ce3b"],"threshold":1},"timestamp":{"keyids":["fd75751f010c3442e23b3e3e99a1442a112f2f21038603cb8609d8b17c9e912a"],"threshold":1}},"version":2},"signatures":[{"keyid":"b1d6813b55442ecbfb1f4b40eb1fcdb4290e53434cfc9ba2da24c26c9143873b","method":"ecdsa","sig":"A2lNVwxHBnD9ViFtRre8r5oG6VvcvJnC6gdvvxv/Jyag40q/fNMjllCqyHrb+6z8XDZcrTTDsFU1R3/e+92d1A=="}]}`

const testRootKey = "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJVekNCKzZBREFnRUNBaEFCWDNKLzkzaW8zbHcrZUsvNFhvSHhNQW9HQ0NxR1NNNDlCQU1DTUJFeER6QU4KQmdOVkJBTVRCbVY0Y0dseVpUQWVGdzB4TlRBM01qQXlNekU1TVRkYUZ3MHlOVEEzTVRjeU16RTVNVGRhTUJFeApEekFOQmdOVkJBTVRCbVY0Y0dseVpUQlpNQk1HQnlxR1NNNDlBZ0VHQ0NxR1NNNDlBd0VIQTBJQUJFTDhOTFhQCitreUJZYzhYY0FTMXB2S2l5MXRQUDlCZHJ1dEdrWlR3Z0dEYTM1THMzSUFXaWlrUmlPbGRuWmxVVEE5cG5JekoKOFlRQThhTjQ1TDQvUlplak5UQXpNQTRHQTFVZER3RUIvd1FFQXdJQW9EQVRCZ05WSFNVRUREQUtCZ2dyQmdFRgpCUWNEQXpBTUJnTlZIUk1CQWY4RUFqQUFNQW9HQ0NxR1NNNDlCQU1DQTBjQU1FUUNJRVJ1ZUVURG5xMlRqRFBmClhGRStqUFJqMEtqdXdEOG9HSmtoVGpMUDAycjhBaUI5cUNyL2ZqSXpJZ1NQcTJVSXZqR0hlYmZOYXh1QlpZZUUKYW8xNjd6dHNYZz09Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K"

type TestRoundTripper struct{}

func (rt *TestRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return http.DefaultClient.Do(req)
}

func TestHTTPStoreGetSized(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(testRoot))
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()
	store, err := NewHTTPStore(
		server.URL,
		"metadata",
		"txt",
		"key",
		&http.Transport{},
	)
	require.NoError(t, err)
	j, err := store.GetSized("root", 4801)
	require.NoError(t, err)
	require.Equal(t, testRoot, string(j))
	p := &data.Signed{}
	err = json.Unmarshal(j, p)
	require.NoError(t, err)
}

// Test that passing -1 to httpstore's GetSized will return all content
func TestHTTPStoreGetAllMeta(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(testRoot))
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()
	store, err := NewHTTPStore(
		server.URL,
		"metadata",
		"txt",
		"key",
		&http.Transport{},
	)
	require.NoError(t, err)
	j, err := store.GetSized("root", NoSizeLimit)
	require.NoError(t, err)
	require.Equal(t, testRoot, string(j))
	p := &data.Signed{}
	err = json.Unmarshal(j, p)
	require.NoError(t, err)
}

func TestSetMultiMeta(t *testing.T) {
	metas := map[string][]byte{
		"root":    []byte("root data"),
		"targets": []byte("targets data"),
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		reader, err := r.MultipartReader()
		require.NoError(t, err)
		updates := make(map[string][]byte)
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			role := strings.TrimSuffix(part.FileName(), ".json")
			updates[role], err = ioutil.ReadAll(part)
			require.NoError(t, err)
		}
		rd, rok := updates["root"]
		require.True(t, rok)
		require.Equal(t, rd, metas["root"])

		td, tok := updates["targets"]
		require.True(t, tok)
		require.Equal(t, td, metas["targets"])

	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()
	store, err := NewHTTPStore(server.URL, "metadata", "json", "key", http.DefaultTransport)
	require.NoError(t, err)

	store.SetMulti(metas)
}

func testErrorCode(t *testing.T, errorCode int, errType error) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(errorCode)
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	store, err := NewHTTPStore(
		server.URL,
		"metadata",
		"txt",
		"key",
		&http.Transport{},
	)
	require.NoError(t, err)

	_, err = store.GetSized("root", 4801)
	require.Error(t, err)
	require.IsType(t, errType, err,
		fmt.Sprintf("%d should translate to %v", errorCode, errType))
}

func Test404Error(t *testing.T) {
	testErrorCode(t, http.StatusNotFound, ErrMetaNotFound{})
}

func Test50XErrors(t *testing.T) {
	fiveHundreds := []int{
		http.StatusInternalServerError,
		http.StatusNotImplemented,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout,
		http.StatusHTTPVersionNotSupported,
	}
	for _, code := range fiveHundreds {
		testErrorCode(t, code, ErrServerUnavailable{})
	}
}

func Test400Error(t *testing.T) {
	testErrorCode(t, http.StatusBadRequest, ErrInvalidOperation{})
}

// If it's a 400, translateStatusToError attempts to parse the body into
// an error.  If successful (and a recognized error) that error is returned.
func TestTranslateErrorsParse400Errors(t *testing.T) {
	origErr := validation.ErrBadRoot{Msg: "bad"}

	serialObj, err := validation.NewSerializableError(origErr)
	require.NoError(t, err)
	serialization, err := json.Marshal(serialObj)
	require.NoError(t, err)
	errorBody := bytes.NewBuffer([]byte(fmt.Sprintf(
		`{"errors": [{"otherstuff": "what", "detail": %s}]}`,
		string(serialization))))
	errorResp := http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       ioutil.NopCloser(errorBody),
	}

	finalError := translateStatusToError(&errorResp, "")
	require.Equal(t, origErr, finalError)
}

// If it's a 400, translateStatusToError attempts to parse the body into
// an error.  If parsing fails, an InvalidOperation is returned instead.
func TestTranslateErrorsWhenCannotParse400(t *testing.T) {
	invalids := []string{
		`{"errors": [{"otherstuff": "what", "detail": {"Name": "Muffin"}}]}`,
		`{"errors": [{"otherstuff": "what", "detail": {}}]}`,
		`{"errors": [{"otherstuff": "what"}]}`,
		`{"errors": []}`,
		`{}`,
		"400",
	}
	for _, body := range invalids {
		errorResp := http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       ioutil.NopCloser(bytes.NewBuffer([]byte(body))),
		}

		err := translateStatusToError(&errorResp, "")
		require.IsType(t, ErrInvalidOperation{}, err)
	}
}

func TestHTTPStoreRemoveAll(t *testing.T) {
	// Set up a simple handler and server for our store, just check that a non-error response back is fine
	handler := func(w http.ResponseWriter, r *http.Request) {}
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()
	store, err := NewHTTPStore(server.URL, "metadata", "json", "key", http.DefaultTransport)
	require.NoError(t, err)

	err = store.RemoveAll()
	require.NoError(t, err)
}

func TestHTTPStoreRotateKey(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(testRootKey))
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()
	store, err := NewHTTPStore(server.URL, "metadata", "json", "key", http.DefaultTransport)
	require.NoError(t, err)

	pubKeyBytes, err := store.RotateKey(data.CanonicalSnapshotRole)
	require.NoError(t, err)
	require.Equal(t, pubKeyBytes, []byte(testRootKey))
}

func TestHTTPOffline(t *testing.T) {
	s, err := NewHTTPStore("https://localhost/", "", "", "", nil)
	require.NoError(t, err)
	require.IsType(t, &OfflineStore{}, s)
}

func TestErrServerUnavailable(t *testing.T) {
	for i := 200; i < 600; i++ {
		err := ErrServerUnavailable{code: i}
		if i == 401 {
			require.Contains(t, err.Error(), "not authorized")
		} else {
			require.Contains(t, err.Error(), "unable to reach trust server")
		}
	}
}
