package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/net/context"

	ctxu "github.com/docker/distribution/context"
	"github.com/docker/distribution/registry/api/errcode"
	"github.com/docker/notary/cryptoservice"
	"github.com/docker/notary/passphrase"
	"github.com/docker/notary/server/errors"
	"github.com/docker/notary/server/storage"
	"github.com/docker/notary/trustmanager"
	"github.com/docker/notary/tuf/data"
	"github.com/docker/notary/tuf/signed"
	"github.com/docker/notary/tuf/store"
	"github.com/docker/notary/tuf/validation"

	"github.com/docker/notary/tuf/testutils"
	"github.com/docker/notary/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type handlerState struct {
	// interface{} so we can test invalid values
	metaStore interface{}
	crypto    interface{}
	keyAlgo   interface{}
}

func defaultState() handlerState {
	return handlerState{
		metaStore: storage.NewMemStorage(),
		crypto:    signed.NewEd25519(),
		keyAlgo:   data.ED25519Key,
	}
}

func getContext(h handlerState) context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, "metaStore", h.metaStore)
	ctx = context.WithValue(ctx, "keyAlgorithm", h.keyAlgo)
	ctx = context.WithValue(ctx, "cryptoService", h.crypto)
	return ctxu.WithLogger(ctx, ctxu.GetRequestLogger(ctx))
}

func TestMainHandlerGet(t *testing.T) {
	hand := utils.RootHandlerFactory(nil, context.Background(), &signed.Ed25519{})
	handler := hand(MainHandler)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	_, err := http.Get(ts.URL)
	if err != nil {
		t.Fatalf("Received error on GET /: %s", err.Error())
	}
}

func TestMainHandlerNotGet(t *testing.T) {
	hand := utils.RootHandlerFactory(nil, context.Background(), &signed.Ed25519{})
	handler := hand(MainHandler)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	res, err := http.Head(ts.URL)
	if err != nil {
		t.Fatalf("Received error on GET /: %s", err.Error())
	}
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("Expected 404, received %d", res.StatusCode)
	}
}

// GetKeyHandler needs to have access to a metadata metaStore and cryptoservice,
// a key algorithm
func TestGetKeyHandlerInvalidConfiguration(t *testing.T) {
	noStore := defaultState()
	noStore.metaStore = nil

	invalidStore := defaultState()
	invalidStore.metaStore = "not a metaStore"

	noCrypto := defaultState()
	noCrypto.crypto = nil

	invalidCrypto := defaultState()
	invalidCrypto.crypto = "not a cryptoservice"

	noKeyAlgo := defaultState()
	noKeyAlgo.keyAlgo = ""

	invalidKeyAlgo := defaultState()
	invalidKeyAlgo.keyAlgo = 1

	invalidStates := map[string][]handlerState{
		"no storage":       {noStore, invalidStore},
		"no cryptoservice": {noCrypto, invalidCrypto},
		"no keyalgorithm":  {noKeyAlgo, invalidKeyAlgo},
	}

	vars := map[string]string{
		"imageName": "gun",
		"tufRole":   data.CanonicalTimestampRole,
	}
	var buf bytes.Buffer
	for errString, states := range invalidStates {
		for _, s := range states {
			err := getKeyHandler(getContext(s), &buf, vars)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), errString)
		}
	}
}

// GetKeyHandler needs to be set up such that an imageName and tufRole are both
// provided and non-empty.
func TestGetKeyHandlerNoRoleOrRepo(t *testing.T) {
	state := defaultState()

	for _, key := range []string{"imageName", "tufRole"} {
		vars := map[string]string{
			"imageName": "gun",
			"tufRole":   data.CanonicalTimestampRole,
		}

		// not provided
		var buf bytes.Buffer
		delete(vars, key)
		err := getKeyHandler(getContext(state), &buf, vars)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown")

		// empty
		vars[key] = ""
		err = getKeyHandler(getContext(state), &buf, vars)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown")
	}
}

// Getting a key for a non-supported role results in a 400.
func TestGetKeyHandlerInvalidRole(t *testing.T) {
	state := defaultState()
	vars := map[string]string{
		"imageName": "gun",
		"tufRole":   data.CanonicalRootRole,
	}
	var buf bytes.Buffer

	err := getKeyHandler(getContext(state), &buf, vars)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid role")
}

// Getting the key for a valid role and gun succeeds
func TestGetKeyHandlerCreatesOnce(t *testing.T) {
	state := defaultState()
	roles := []string{data.CanonicalTimestampRole, data.CanonicalSnapshotRole}
	var buf bytes.Buffer

	for _, role := range roles {
		vars := map[string]string{"imageName": "gun", "tufRole": role}
		err := getKeyHandler(getContext(state), &buf, vars)
		assert.NoError(t, err)
		assert.True(t, len(buf.String()) > 0)
	}
}

func TestGetHandlerRoot(t *testing.T) {
	metaStore := storage.NewMemStorage()
	_, repo, _, err := testutils.EmptyRepo("gun")
	assert.NoError(t, err)

	ctx := context.Background()
	ctx = context.WithValue(ctx, "metaStore", metaStore)

	root, err := repo.SignRoot(data.DefaultExpires("root"))
	rootJSON, err := json.Marshal(root)
	assert.NoError(t, err)
	metaStore.UpdateCurrent("gun", storage.MetaUpdate{Role: "root", Version: 1, Data: rootJSON})

	req := &http.Request{
		Body: ioutil.NopCloser(bytes.NewBuffer(nil)),
	}

	vars := map[string]string{
		"imageName": "gun",
		"tufRole":   "root",
	}

	rw := httptest.NewRecorder()

	err = getHandler(ctx, rw, req, vars)
	assert.NoError(t, err)
}

func TestGetHandlerTimestamp(t *testing.T) {
	metaStore := storage.NewMemStorage()
	_, repo, crypto, err := testutils.EmptyRepo("gun")
	assert.NoError(t, err)

	ctx := getContext(handlerState{metaStore: metaStore, crypto: crypto})

	sn, err := repo.SignSnapshot(data.DefaultExpires("snapshot"))
	snJSON, err := json.Marshal(sn)
	assert.NoError(t, err)
	metaStore.UpdateCurrent(
		"gun", storage.MetaUpdate{Role: "snapshot", Version: 1, Data: snJSON})

	ts, err := repo.SignTimestamp(data.DefaultExpires("timestamp"))
	tsJSON, err := json.Marshal(ts)
	assert.NoError(t, err)
	metaStore.UpdateCurrent(
		"gun", storage.MetaUpdate{Role: "timestamp", Version: 1, Data: tsJSON})

	req := &http.Request{
		Body: ioutil.NopCloser(bytes.NewBuffer(nil)),
	}

	vars := map[string]string{
		"imageName": "gun",
		"tufRole":   "timestamp",
	}

	rw := httptest.NewRecorder()

	err = getHandler(ctx, rw, req, vars)
	assert.NoError(t, err)
}

func TestGetHandlerSnapshot(t *testing.T) {
	metaStore := storage.NewMemStorage()
	_, repo, crypto, err := testutils.EmptyRepo("gun")
	assert.NoError(t, err)

	ctx := getContext(handlerState{metaStore: metaStore, crypto: crypto})

	sn, err := repo.SignSnapshot(data.DefaultExpires("snapshot"))
	snJSON, err := json.Marshal(sn)
	assert.NoError(t, err)
	metaStore.UpdateCurrent(
		"gun", storage.MetaUpdate{Role: "snapshot", Version: 1, Data: snJSON})

	req := &http.Request{
		Body: ioutil.NopCloser(bytes.NewBuffer(nil)),
	}

	vars := map[string]string{
		"imageName": "gun",
		"tufRole":   "snapshot",
	}

	rw := httptest.NewRecorder()

	err = getHandler(ctx, rw, req, vars)
	assert.NoError(t, err)
}

func TestGetHandler404(t *testing.T) {
	metaStore := storage.NewMemStorage()

	ctx := context.Background()
	ctx = context.WithValue(ctx, "metaStore", metaStore)

	req := &http.Request{
		Body: ioutil.NopCloser(bytes.NewBuffer(nil)),
	}

	vars := map[string]string{
		"imageName": "gun",
		"tufRole":   "root",
	}

	rw := httptest.NewRecorder()

	err := getHandler(ctx, rw, req, vars)
	assert.Error(t, err)
}

func TestGetHandlerNilData(t *testing.T) {
	metaStore := storage.NewMemStorage()
	metaStore.UpdateCurrent("gun", storage.MetaUpdate{Role: "root", Version: 1, Data: nil})

	ctx := context.Background()
	ctx = context.WithValue(ctx, "metaStore", metaStore)

	req := &http.Request{
		Body: ioutil.NopCloser(bytes.NewBuffer(nil)),
	}

	vars := map[string]string{
		"imageName": "gun",
		"tufRole":   "root",
	}

	rw := httptest.NewRecorder()

	err := getHandler(ctx, rw, req, vars)
	assert.Error(t, err)
}

func TestGetHandlerNoStorage(t *testing.T) {
	ctx := context.Background()

	req := &http.Request{
		Body: ioutil.NopCloser(bytes.NewBuffer(nil)),
	}

	err := GetHandler(ctx, nil, req)
	assert.Error(t, err)
}

// a validation failure, such as a snapshots file being missing, will be
// propagated as a detail in the error (which gets serialized as the body of the
// response)
func TestAtomicUpdateValidationFailurePropagated(t *testing.T) {
	metaStore := storage.NewMemStorage()
	gun := "testGUN"
	vars := map[string]string{"imageName": gun}

	kdb, repo, cs, err := testutils.EmptyRepo(gun)
	assert.NoError(t, err)
	copyTimestampKey(t, kdb, metaStore, gun)
	state := handlerState{metaStore: metaStore, crypto: cs}

	r, tg, sn, ts, err := testutils.Sign(repo)
	assert.NoError(t, err)
	rs, tgs, _, _, err := testutils.Serialize(r, tg, sn, ts)
	assert.NoError(t, err)

	req, err := store.NewMultiPartMetaRequest("", map[string][]byte{
		data.CanonicalRootRole:    rs,
		data.CanonicalTargetsRole: tgs,
	})

	rw := httptest.NewRecorder()

	err = atomicUpdateHandler(getContext(state), rw, req, vars)
	assert.Error(t, err)
	errorObj, ok := err.(errcode.Error)
	assert.True(t, ok, "Expected an errcode.Error, got %v", err)
	assert.Equal(t, errors.ErrInvalidUpdate, errorObj.Code)
	serializable, ok := errorObj.Detail.(*validation.SerializableError)
	assert.True(t, ok, "Expected a SerializableObject, got %v", errorObj.Detail)
	assert.IsType(t, validation.ErrBadHierarchy{}, serializable.Error)
}

type failStore struct {
	storage.MemStorage
}

func (s *failStore) GetCurrent(_, _ string) ([]byte, error) {
	return nil, fmt.Errorf("oh no! storage has failed")
}

// a non-validation failure, such as the storage failing, will not be propagated
// as a detail in the error (which gets serialized as the body of the response)
func TestAtomicUpdateNonValidationFailureNotPropagated(t *testing.T) {
	metaStore := storage.NewMemStorage()
	gun := "testGUN"
	vars := map[string]string{"imageName": gun}

	kdb, repo, cs, err := testutils.EmptyRepo(gun)
	assert.NoError(t, err)
	copyTimestampKey(t, kdb, metaStore, gun)
	state := handlerState{metaStore: &failStore{*metaStore}, crypto: cs}

	r, tg, sn, ts, err := testutils.Sign(repo)
	assert.NoError(t, err)
	rs, tgs, sns, _, err := testutils.Serialize(r, tg, sn, ts)
	assert.NoError(t, err)

	req, err := store.NewMultiPartMetaRequest("", map[string][]byte{
		data.CanonicalRootRole:     rs,
		data.CanonicalTargetsRole:  tgs,
		data.CanonicalSnapshotRole: sns,
	})

	rw := httptest.NewRecorder()

	err = atomicUpdateHandler(getContext(state), rw, req, vars)
	assert.Error(t, err)
	errorObj, ok := err.(errcode.Error)
	assert.True(t, ok, "Expected an errcode.Error, got %v", err)
	assert.Equal(t, errors.ErrInvalidUpdate, errorObj.Code)
	assert.Nil(t, errorObj.Detail)
}

type invalidVersionStore struct {
	storage.MemStorage
}

func (s *invalidVersionStore) UpdateMany(_ string, _ []storage.MetaUpdate) error {
	return storage.ErrOldVersion{}
}

// a non-validation failure, such as the storage failing, will be propagated
// as a detail in the error (which gets serialized as the body of the response)
func TestAtomicUpdateVersionErrorPropagated(t *testing.T) {
	metaStore := storage.NewMemStorage()
	gun := "testGUN"
	vars := map[string]string{"imageName": gun}

	kdb, repo, cs, err := testutils.EmptyRepo(gun)
	assert.NoError(t, err)
	copyTimestampKey(t, kdb, metaStore, gun)
	state := handlerState{metaStore: &invalidVersionStore{*metaStore}, crypto: cs}

	r, tg, sn, ts, err := testutils.Sign(repo)
	assert.NoError(t, err)
	rs, tgs, sns, _, err := testutils.Serialize(r, tg, sn, ts)
	assert.NoError(t, err)

	req, err := store.NewMultiPartMetaRequest("", map[string][]byte{
		data.CanonicalRootRole:     rs,
		data.CanonicalTargetsRole:  tgs,
		data.CanonicalSnapshotRole: sns,
	})

	rw := httptest.NewRecorder()

	err = atomicUpdateHandler(getContext(state), rw, req, vars)
	assert.Error(t, err)
	errorObj, ok := err.(errcode.Error)
	assert.True(t, ok, "Expected an errcode.Error, got %v", err)
	assert.Equal(t, errors.ErrOldVersion, errorObj.Code)
	assert.Equal(t, storage.ErrOldVersion{}, errorObj.Detail)
}

// If there are no keys for that role and that gun, GetOrCreateKey creates one
// and returns it.  GetOrCreateKeys will return that key from now on if
// are no key changes.
func TestGetOrCreateKeyCurrentNoKeys(t *testing.T) {
	s := serverKeyInfo{
		gun:           "gun",
		role:          data.CanonicalTimestampRole,
		store:         storage.NewMemStorage(),
		crypto:        signed.NewEd25519(),
		keyAlgo:       data.ED25519Key,
		rotateOncePer: 24 * time.Hour,
	}
	k1, err := GetOrCreateKey(s)
	require.NoError(t, err, "Expected nil error")
	require.NotNil(t, k1, "Key should not be nil")

	k2, err := GetOrCreateKey(s)
	require.NoError(t, err, "Expected nil error")
	require.NotNil(t, k2, "Key should not be nil")

	// trying to get the same key again should return the same value
	require.Equal(t, k1.ID(), k2.ID(), "Did not receive same key when attempting to recreate.")
}

// If there are lots of keys, it returns the most recently used key, whether it's
// pending or not.
func TestGetOrCreateKeyCurrentMostRecentUsedKeyPendingOrNot(t *testing.T) {
	s := serverKeyInfo{
		gun:           "gun",
		role:          data.CanonicalSnapshotRole,
		store:         storage.NewMemStorage(),
		crypto:        signed.NewEd25519(),
		keyAlgo:       data.ED25519Key,
		rotateOncePer: 24 * time.Hour,
	}

	for i := 0; i < 2; i++ {
		k, err := s.crypto.Create(s.role, data.ED25519Key)
		require.NoError(t, err)
		s.store.AddKey(s.gun, s.role, k, time.Now().AddDate(1, 1, 1))

		if i == 0 {
			s.store.MarkActiveKeys(s.gun, s.role, []string{k.ID()})
		}

		gotten, err := GetOrCreateKey(s)
		require.NoError(t, err, "Expected nil error")
		require.Equal(t, k.ID(), gotten.ID(), "Key should not be nil")
	}
}

// If there is an existing most recently used key, it is returned whether the
// algorithm is what is desired or not.
func TestGetOrCreateKeyCurrentMostRecentUsedKeyIgnoreAlgorithm(t *testing.T) {
	gun := "gun"
	s := serverKeyInfo{
		gun:   gun,
		role:  data.CanonicalSnapshotRole,
		store: storage.NewMemStorage(),
		crypto: cryptoservice.NewCryptoService(
			gun, trustmanager.NewKeyMemoryStore(passphrase.ConstantRetriever(""))),
		keyAlgo:       data.ECDSAKey,
		rotateOncePer: 24 * time.Hour,
	}

	k, err := s.crypto.Create(s.role, data.RSAKey)
	require.NoError(t, err)
	s.store.AddKey(s.gun, s.role, k, time.Now().AddDate(1, 1, 1))

	gotten, err := GetOrCreateKey(s)
	require.NoError(t, err, "Expected nil error")
	require.Equal(t, k.ID(), gotten.ID(), "Key should not be nil")
}

// If there is an existing most recently used key that is not expired
func TestGetOrCreateKeyCurrentMostRecentUsedKeyIgnoreExpired(t *testing.T) {
	s := serverKeyInfo{
		gun:           "gun",
		role:          data.CanonicalSnapshotRole,
		store:         storage.NewMemStorage(),
		crypto:        signed.NewEd25519(),
		keyAlgo:       data.ED25519Key,
		rotateOncePer: 24 * time.Hour,
	}

	k1, err := s.crypto.Create(s.role, data.ED25519Key)
	require.NoError(t, err)
	s.store.AddKey(s.gun, s.role, k1, time.Now().AddDate(1, 1, 1))

	k2, err := s.crypto.Create(s.role, data.ED25519Key)
	require.NoError(t, err)
	s.store.AddKey(s.gun, s.role, k2, time.Now().AddDate(-1, -1, -1))

	gotten, err := GetOrCreateKey(s)
	require.NoError(t, err, "Expected nil error")
	require.Equal(t, k1.ID(), gotten.ID(), "Key should not be nil")
}

// If there are no keys for that role and that gun, RotateKey creates one
// and returns it.  RotateKey will return that key from now on if
// are no key changes, until it expires
func TestRotateKeyNoKeys(t *testing.T) {
	s := serverKeyInfo{
		gun:           "gun",
		role:          data.CanonicalTimestampRole,
		store:         storage.NewMemStorage(),
		crypto:        signed.NewEd25519(),
		keyAlgo:       data.ED25519Key,
		rotateOncePer: 24 * time.Hour,
	}
	k1, err := RotateKey(s)
	require.NoError(t, err, "Expected nil error")
	require.NotNil(t, k1, "Key should not be nil")

	_, err = RotateKey(s)
	require.Error(t, err, "Expected an error trying to rotate again")
	errCode, ok := err.(errcode.Error)
	require.True(t, ok)
	require.Equal(t, errors.ErrCannotRotateKey, errCode.Code)
}

// If there are no pending keys (even if there are current keys) for that role and
// that gun, RotateKey creates one and returns it.  RotateKey will return that key
// from now on if are no key changes, until it expires
func TestRotateKeyNoPendingKeys(t *testing.T) {
	s := serverKeyInfo{
		gun:           "gun",
		role:          data.CanonicalTimestampRole,
		store:         storage.NewMemStorage(),
		crypto:        signed.NewEd25519(),
		keyAlgo:       data.ED25519Key,
		rotateOncePer: 24 * time.Hour,
	}
	activeKey, err := s.crypto.Create(s.role, data.ED25519Key)
	require.NoError(t, err)
	require.NoError(t, s.store.AddKey(s.gun, s.role, activeKey, time.Now().AddDate(1, 1, 1)))
	require.NoError(t, s.store.MarkActiveKeys(s.gun, s.role, []string{activeKey.ID()}))

	key, err := RotateKey(s)
	require.NoError(t, err, "Expected nil error")
	require.NotNil(t, key, "Key should not be nil")

	require.NotEqual(t, activeKey.ID(), key.ID(), "Received same key when attempting to rotate.")
}

// If is a pending key for that role and that gun, but it's expired, RotateKey
// creates a new one one and returns it.  RotateKey will return that key from now on if
// are no key changes, until it expires
func TestRotateKeyExistingExpired(t *testing.T) {
	s := serverKeyInfo{
		gun:           "gun",
		role:          data.CanonicalTimestampRole,
		store:         storage.NewMemStorage(),
		crypto:        signed.NewEd25519(),
		keyAlgo:       data.ED25519Key,
		rotateOncePer: 24 * time.Hour,
	}
	expiredKey, err := s.crypto.Create(s.role, data.ED25519Key)
	require.NoError(t, err)
	require.NoError(t, s.store.AddKey(s.gun, s.role, expiredKey, time.Now().AddDate(-1, -1, -1)))

	key, err := RotateKey(s)
	require.NoError(t, err, "Expected nil error")
	require.NotNil(t, key, "Key should not be nil")

	require.NotEqual(t, expiredKey.ID(), key.ID(), "Received same key when attempting to rotate.")
}
