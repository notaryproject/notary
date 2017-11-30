// Package usersdb provides auth facilities from a CouchDB _users database.
package usersdb

import (
	"context"
	"crypto/sha1"
	"fmt"

	"golang.org/x/crypto/pbkdf2"

	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/authdb"
	"github.com/flimzy/kivik/errors"
)

type db struct {
	*kivik.DB
}

var _ authdb.UserStore = &db{}

// New returns a new authdb.UserStore backed by a the provided database.
func New(userDB *kivik.DB) authdb.UserStore {
	return &db{userDB}
}

type user struct {
	Name           string   `json:"name"`
	Roles          []string `json:"roles"`
	PasswordScheme string   `json:"password_scheme,omitempty"`
	Salt           string   `json:"salt,omitempty"`
	Iterations     int      `json:"iterations,omitempty"`
	DerivedKey     string   `json:"derived_key,omitempty"`
}

func (db *db) getUser(ctx context.Context, username string) (*user, error) {
	row, err := db.Get(ctx, kivik.UserPrefix+username, nil)
	if err != nil {
		return nil, err
	}
	var u user
	if err = row.ScanDoc(&u); err != nil {
		return nil, err
	}
	return &u, nil
}

func (db *db) Validate(ctx context.Context, username, password string) (*authdb.UserContext, error) {
	u, err := db.getUser(ctx, username)
	if err != nil {
		if kivik.StatusCode(err) == kivik.StatusNotFound {
			err = errors.Status(kivik.StatusUnauthorized, "unauthorized")
		}
		return nil, err
	}

	switch u.PasswordScheme {
	case "":
		return nil, errors.New("no password scheme set for user")
	case authdb.SchemePBKDF2:
	default:
		return nil, errors.Errorf("unsupported password scheme: %s", u.PasswordScheme)
	}
	key := fmt.Sprintf("%x", pbkdf2.Key([]byte(password), []byte(u.Salt), u.Iterations, authdb.PBKDF2KeyLength, sha1.New))
	if key != u.DerivedKey {
		return nil, errors.Status(kivik.StatusUnauthorized, "unauthorized")
	}
	return &authdb.UserContext{
		Name:  u.Name,
		Roles: u.Roles,
		Salt:  u.Salt,
	}, nil
}

func (db *db) UserCtx(ctx context.Context, username string) (*authdb.UserContext, error) {
	u, err := db.getUser(ctx, username)
	if err != nil {
		return nil, err
	}
	return &authdb.UserContext{
		Name:  u.Name,
		Roles: u.Roles,
		Salt:  u.Salt,
	}, nil
}
