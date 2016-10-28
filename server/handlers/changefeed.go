package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	ctxu "github.com/docker/distribution/context"
	"golang.org/x/net/context"

	"github.com/docker/notary"
	"github.com/docker/notary/server/errors"
	"github.com/docker/notary/server/storage"
)

type changefeedResponse struct {
	NumberOfRecords int              `json:"count"`
	Records         []storage.Change `json:"records"`
}

// Changefeed returns a list of changes according to the provided filters
func Changefeed(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	logger := ctxu.GetLogger(ctx)
	s := ctx.Value(notary.CtxKeyMetaStore)
	store, ok := s.(storage.MetaStore)
	if !ok {
		logger.Error("500 GET unable to retrieve storage")
		return errors.ErrNoStorage.WithDetail(nil)
	}

	qs := r.URL.Query()
	imageName := qs.Get("filter")
	if imageName == "" {
		// no image name means global feed
		imageName = "*"
	}
	pageSizeStr := qs.Get("page_size")
	pageSize, err := strconv.ParseInt(pageSizeStr, 10, 32)
	if err != nil {
		logger.Errorf("400 GET invalid pageSize: %s", pageSizeStr)
		return errors.ErrInvalidParams.WithDetail("invalid pageSize parameter, must be an integer >= 0")
	}
	if pageSize == 0 {
		pageSize = notary.DefaultPageSize
	}

	changeID := qs.Get("change_id")
	reversedStr := qs.Get("reversed")
	reversed := reversedStr == "1" || strings.ToLower(reversedStr) == "true"

	changes, err := store.GetChanges(changeID, int(pageSize), imageName, reversed)
	if err != nil {
		logger.Errorf("500 GET could not retrieve records: %s", err.Error())
		return errors.ErrUnknown.WithDetail(err)
	}

	// if reversed, we need to flip the list order so oldest is first
	if reversed {
		for i, j := 0, len(changes)-1; i < j; i, j = i+1, j-1 {
			changes[i], changes[j] = changes[j], changes[i]
		}
	}

	out, err := json.Marshal(&changefeedResponse{
		NumberOfRecords: len(changes),
		Records:         changes,
	})
	if err != nil {
		logger.Error("500 GET could not json.Marshal changefeedResponse")
		return errors.ErrUnknown.WithDetail(err)
	}
	w.Write(out)
	return nil
}
