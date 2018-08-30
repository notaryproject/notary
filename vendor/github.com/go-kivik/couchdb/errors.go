package couchdb

import (
	"github.com/go-kivik/kivik"
	"github.com/go-kivik/kivik/errors"
)

func missingArg(arg string) error {
	return errors.Statusf(kivik.StatusBadRequest, "kivik: %s required", arg)
}
