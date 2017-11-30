// Package confadmin provides an authentication service for admins configured
// in server configuration.
package confadmin

import (
	"context"
	"strconv"
	"strings"

	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/authdb"
	"github.com/flimzy/kivik/errors"
	"github.com/go-kivik/kivikd/conf"
)

type confadmin struct {
	*conf.Conf
}

var _ authdb.UserStore = &confadmin{}

// New returns a new confadmin authentication service provider.
func New(c *conf.Conf) authdb.UserStore {
	return &confadmin{c}
}

func (c *confadmin) Validate(ctx context.Context, username, password string) (*authdb.UserContext, error) {
	derivedKey, salt, iterations, err := c.getKeySaltIter(ctx, username)
	if err != nil {
		if kivik.StatusCode(err) == kivik.StatusNotFound {
			return nil, errors.Status(kivik.StatusUnauthorized, "unauthorized")
		}
		return nil, errors.Wrap(err, "unrecognized password hash")
	}
	if !authdb.ValidatePBKDF2(password, salt, derivedKey, iterations) {
		return nil, errors.Status(kivik.StatusUnauthorized, "unauthorized")
	}
	return &authdb.UserContext{
		Name:  username,
		Roles: []string{"_admin"},
		Salt:  salt,
	}, nil
}

const hashPrefix = "-" + authdb.SchemePBKDF2 + "-"

func (c *confadmin) getKeySaltIter(ctx context.Context, username string) (key, salt string, iterations int, err error) {
	confName := "admins." + username
	if !c.IsSet(confName) {
		return "", "", 0, errors.Status(kivik.StatusNotFound, "user not found")
	}
	hash := c.GetString(confName)
	if !strings.HasPrefix(hash, hashPrefix) {
		return "", "", 0, errors.New("unrecognized password scheme")
	}
	parts := strings.Split(strings.TrimPrefix(hash, hashPrefix), ",")
	if len(parts) != 3 {
		return "", "", 0, errors.New("unrecognized hash format")
	}
	if iterations, err = strconv.Atoi(parts[2]); err != nil {
		return "", "", 0, errors.New("unrecognized has format")
	}
	return parts[0], parts[1], iterations, nil
}

func (c *confadmin) UserCtx(ctx context.Context, username string) (*authdb.UserContext, error) {
	_, salt, _, err := c.getKeySaltIter(ctx, username)
	if err != nil {
		if kivik.StatusCode(err) == kivik.StatusNotFound {
			return nil, errors.Status(kivik.StatusNotFound, "user does not exist")
		}
		return nil, errors.Wrap(err, "unrecognized password hash")
	}
	return &authdb.UserContext{
		Name:  username,
		Roles: []string{"_admin"},
		Salt:  salt,
	}, nil
}
