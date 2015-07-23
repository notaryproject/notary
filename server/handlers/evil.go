package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"golang.org/x/net/context"

	"github.com/docker/notary/errors"
	"github.com/docker/notary/server/storage"
	"math/rand"
)

// EvilHandler returns a tampered json for a specified role and GUN.
func EvilHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	s := ctx.Value("metaStore")
	store, ok := s.(storage.MetaStore)
	if !ok {
		return errors.ErrNoStorage.WithDetail(nil)
	}
	vars := mux.Vars(r)
	gun := vars["imageName"]
	tufRole := vars["tufRole"]
	out, err := store.GetCurrent(gun, tufRole)
	if err != nil {
		if _, ok := err.(*storage.ErrNotFound); ok {
			return errors.ErrMetadataNotFound.WithDetail(nil)
		}
		logrus.Errorf("[Notary Server] 500 GET repository: %s, role: %s", gun, tufRole)
		return errors.ErrUnknown.WithDetail(err)
	}
	if out == nil {
		logrus.Errorf("[Notary Server] 404 GET repository: %s, role: %s", gun, tufRole)
		return errors.ErrMetadataNotFound.WithDetail(nil)
	}

	logrus.Debug("Tampering data")
	var objmap map[string]*json.RawMessage
	err = json.Unmarshal(out, &objmap)

	numMapItems := len(objmap)
	keyToTamper := rand.Int() % numMapItems
	i := 0
	for _, jsonMessage := range objmap {
		if i == keyToTamper {
			jsonMessageLength := len(*jsonMessage)
			byteToTamper := rand.Int() % jsonMessageLength

			// Flip a bit
			([]byte)(*jsonMessage)[byteToTamper] ^= 1
		}
		i++
	}

	tamperedOut, err := json.Marshal(objmap)

	w.Write(tamperedOut)

	return nil
}
