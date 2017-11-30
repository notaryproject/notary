// Package authgroup groups two or more authentication backends together, trying
// one, then falling through to the others.
package authgroup

import (
	"context"

	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/authdb"
	"github.com/flimzy/kivik/errors"
)

// AuthGroup is a group of auth handlers, to be tried in turn.
type AuthGroup []authdb.UserStore

var _ authdb.UserStore = AuthGroup{}

// New initializes a group of auth handlers. Each one is tried in turn, in the
// order passed to New.
func New(userStores ...authdb.UserStore) authdb.UserStore {
	return append(AuthGroup{}, userStores...)
}

func (g AuthGroup) loop(ctx context.Context, fn func(authdb.UserStore) (*authdb.UserContext, error)) (*authdb.UserContext, error) {
	var firstErr error
	for _, store := range g {
		uCtx, err := fn(store)
		if err == nil {
			return uCtx, nil
		}
		if kivik.StatusCode(err) != kivik.StatusNotFound && firstErr == nil {
			firstErr = err
		}
		select {
		// See if our context has expired
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
	}
	if firstErr == nil {
		return nil, errors.Status(kivik.StatusNotFound, "user not found")
	}
	return nil, firstErr
}

// Validate loops through each of the auth handlers, in the order passed to New,
// until the context is cancelled, in which case the context's error is returned.
// The first validation success returns. Errors are discarded unless all auth
// handlers fail to validate the user, in which case only the first error
// received will be returned.
func (g AuthGroup) Validate(ctx context.Context, username, password string) (*authdb.UserContext, error) {
	return g.loop(ctx, func(store authdb.UserStore) (*authdb.UserContext, error) {
		return store.Validate(ctx, username, password)
	})
}

// UserCtx loops through each of the auth handlers, in the order passed to New
// until the context is cancelled, in which case the context's error is returned.
// The first one to not return an error returns. If all of the handlers return
// a Not Found error, Not Found is returned. If any other errors are returned,
// the first is returned to the caller.
func (g AuthGroup) UserCtx(ctx context.Context, username string) (*authdb.UserContext, error) {
	return g.loop(ctx, func(store authdb.UserStore) (*authdb.UserContext, error) {
		return store.UserCtx(ctx, username)
	})
}
