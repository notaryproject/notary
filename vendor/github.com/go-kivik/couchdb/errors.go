package couchdb

import (
	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/errors"
)

func missingArg(arg string) error {
	return errors.Statusf(kivik.StatusBadRequest, "kivik: %s required", arg)
}
