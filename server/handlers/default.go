package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	ctxu "github.com/docker/distribution/context"
	"github.com/gorilla/mux"
	"golang.org/x/net/context"

	"github.com/docker/notary/server/errors"
	"github.com/docker/notary/server/snapshot"
	"github.com/docker/notary/server/storage"
	"github.com/docker/notary/server/timestamp"
	"github.com/docker/notary/tuf/data"
	"github.com/docker/notary/tuf/signed"
	"github.com/docker/notary/tuf/validation"
)

// MainHandler is the default handler for the server
func MainHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	// For now it only supports `GET`
	if r.Method != "GET" {
		return errors.ErrGenericNotFound.WithDetail(nil)
	}

	if _, err := w.Write([]byte("{}")); err != nil {
		return errors.ErrUnknown.WithDetail(err)
	}
	return nil
}

// AtomicUpdateHandler will accept multiple TUF files and ensure that the storage
// backend is atomically updated with all the new records.
func AtomicUpdateHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	defer r.Body.Close()
	vars := mux.Vars(r)
	return atomicUpdateHandler(ctx, w, r, vars)
}

func atomicUpdateHandler(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	gun := vars["imageName"]
	s := ctx.Value("metaStore")
	store, ok := s.(storage.MetaStore)
	if !ok {
		return errors.ErrNoStorage.WithDetail(nil)
	}
	cryptoServiceVal := ctx.Value("cryptoService")
	cryptoService, ok := cryptoServiceVal.(signed.CryptoService)
	if !ok {
		return errors.ErrNoCryptoService.WithDetail(nil)
	}

	reader, err := r.MultipartReader()
	if err != nil {
		return errors.ErrMalformedUpload.WithDetail(nil)
	}
	var updates []storage.MetaUpdate
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		role := strings.TrimSuffix(part.FileName(), ".json")
		if role == "" {
			return errors.ErrNoFilename.WithDetail(nil)
		} else if !data.ValidRole(role) {
			return errors.ErrInvalidRole.WithDetail(role)
		}
		meta := &data.SignedMeta{}
		var input []byte
		inBuf := bytes.NewBuffer(input)
		dec := json.NewDecoder(io.TeeReader(part, inBuf))
		err = dec.Decode(meta)
		if err != nil {
			return errors.ErrMalformedJSON.WithDetail(nil)
		}
		version := meta.Signed.Version
		updates = append(updates, storage.MetaUpdate{
			Role:    role,
			Version: version,
			Data:    inBuf.Bytes(),
		})
	}
	updates, err = validateUpdate(cryptoService, gun, updates, store)
	if err != nil {
		serializable, serializableError := validation.NewSerializableError(err)
		if serializableError != nil {
			return errors.ErrInvalidUpdate.WithDetail(nil)
		}
		return errors.ErrInvalidUpdate.WithDetail(serializable)
	}
	err = store.UpdateMany(gun, updates)
	if err != nil {
		// If we have an old version error, surface to user with error code
		if _, ok := err.(storage.ErrOldVersion); ok {
			return errors.ErrOldVersion.WithDetail(err)
		}
		// More generic storage update error, possibly due to attempted rollback
		return errors.ErrUpdating.WithDetail(nil)
	}
	return nil
}

// GetHandler returns the json for a specified role and GUN.
func GetHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	defer r.Body.Close()
	vars := mux.Vars(r)
	return getHandler(ctx, w, r, vars)
}

func getHandler(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	gun := vars["imageName"]
	checksum := vars["checksum"]
	tufRole := vars["tufRole"]
	s := ctx.Value("metaStore")
	store, ok := s.(storage.MetaStore)
	if !ok {
		return errors.ErrNoStorage.WithDetail(nil)
	}

	return getRole(ctx, w, store, gun, tufRole, checksum)
}

// DeleteHandler deletes all data for a GUN. A 200 responses indicates success.
func DeleteHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	s := ctx.Value("metaStore")
	store, ok := s.(storage.MetaStore)
	if !ok {
		return errors.ErrNoStorage.WithDetail(nil)
	}
	vars := mux.Vars(r)
	gun := vars["imageName"]
	logger := ctxu.GetLoggerWithField(ctx, gun, "gun")
	err := store.Delete(gun)
	if err != nil {
		logger.Error("500 DELETE repository")
		return errors.ErrUnknown.WithDetail(err)
	}
	return nil
}

// GetKeyHandler returns a public key for the specified role, creating a new key-pair
// it if it doesn't yet exist
func GetKeyHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	defer r.Body.Close()
	return getKeyHandler(ctx, w, mux.Vars(r))
}

func getKeyHandler(ctx context.Context, w io.Writer, vars map[string]string) error {
	keyInfo, err := parseKeyParams(ctx, vars)
	if err != nil {
		return err
	}
	pubKey, err := GetOrCreateKey(*keyInfo)
	if err != nil {
		return err
	}
	out, err := json.Marshal(pubKey)
	if err != nil {
		return errors.ErrUnknown.WithDetail(err)
	}
	w.Write(out)
	return nil
}

type serverKeyInfo struct {
	gun           string
	role          string
	store         storage.MetaStore
	crypto        signed.CryptoService
	keyAlgo       string
	rotateOncePer time.Duration
}

func parseKeyParams(ctx context.Context, vars map[string]string) (*serverKeyInfo, error) {
	gun, ok := vars["imageName"]
	if !ok || gun == "" {
		return nil, errors.ErrUnknown.WithDetail("no gun")
	}
	role, ok := vars["tufRole"]
	if !ok || role == "" {
		return nil, errors.ErrUnknown.WithDetail("no role")
	}
	if role != data.CanonicalTimestampRole && role != data.CanonicalSnapshotRole {
		return nil, errors.ErrInvalidRole.WithDetail(role)
	}

	s := ctx.Value("metaStore")
	store, ok := s.(storage.MetaStore)
	if !ok || store == nil {
		return nil, errors.ErrNoStorage.WithDetail(nil)
	}

	c := ctx.Value("cryptoService")
	crypto, ok := c.(signed.CryptoService)
	if !ok || crypto == nil {
		return nil, errors.ErrNoCryptoService.WithDetail(nil)
	}

	algo := ctx.Value("keyAlgorithm")
	keyAlgo, ok := algo.(string)
	if !ok || keyAlgo != data.ECDSAKey && keyAlgo != data.RSAKey && keyAlgo != data.ED25519Key {
		return nil, errors.ErrNoKeyAlgorithm.WithDetail(nil)
	}

	return &serverKeyInfo{
		gun:     gun,
		role:    role,
		store:   store,
		crypto:  crypto,
		keyAlgo: keyAlgo,
	}, nil
}

func createNewKey(s serverKeyInfo) (data.PublicKey, error) {
	key, err := s.crypto.Create(s.role, s.keyAlgo)
	if err != nil {
		return nil, errors.ErrUnknown.WithDetail(err)
	}
	logrus.Debug("Creating new timestamp key for ", s.gun, ". With algo: ", s.keyAlgo)
	err = s.store.AddKey(s.gun, s.role, key, time.Now().Add(s.rotateOncePer))
	if err != nil {
		return nil, errors.ErrUnknown.WithDetail(err)
	}
	return key, nil
}

// GetOrCreateKey returns either the current or pending timestamp key for the given
// gun and role, whichever key was the most recently created.  If no key is found,
// it creates a new one.
func GetOrCreateKey(s serverKeyInfo) (data.PublicKey, error) {
	key, err := s.store.GetLatestKey(s.gun, s.role)
	if _, ok := err.(storage.ErrNoKey); ok {
		return createNewKey(s)
	}

	if err == nil {
		return key.PublicKey, nil
	}

	return nil, errors.ErrUnknown.WithDetail(err)
}

// RotateKey attempts to rotate a key for a role.  If it has been rotated too
// recently without being signed into the root.json as an actively used signing
// key, rotation fails.  Activating the key (by signing it into the root.json)
// resets the rotation rate limit.
func RotateKey(s serverKeyInfo) (data.PublicKey, error) {
	// GetLatestKeys excludes expired keys, so we know that this key is unexpired
	key, err := s.store.GetLatestKey(s.gun, s.role)

	if err == nil && key.Pending { // there is already a good pending key
		return nil, errors.ErrCannotRotateKey.WithDetail(key.ID())
	}

	if _, ok := err.(storage.ErrNoKey); !ok && err != nil {
		return nil, errors.ErrUnknown.WithDetail(err)
	}

	return createNewKey(s)
}

// NotFoundHandler is used as a generic catch all handler to return the ErrMetadataNotFound
// 404 response
func NotFoundHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	return errors.ErrMetadataNotFound.WithDetail(nil)
}
