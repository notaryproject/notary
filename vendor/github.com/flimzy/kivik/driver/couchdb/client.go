package couchdb

import (
	"context"

	"github.com/flimzy/kivik"
)

func (c *client) AllDBs(ctx context.Context, _ map[string]interface{}) ([]string, error) {
	var allDBs []string
	_, err := c.DoJSON(ctx, kivik.MethodGet, "/_all_dbs", nil, &allDBs)
	return allDBs, err
}

func (c *client) DBExists(ctx context.Context, dbName string, _ map[string]interface{}) (bool, error) {
	_, err := c.DoError(ctx, kivik.MethodHead, dbName, nil)
	if kivik.StatusCode(err) == kivik.StatusNotFound {
		return false, nil
	}
	return err == nil, err
}

func (c *client) CreateDB(ctx context.Context, dbName string, _ map[string]interface{}) error {
	_, err := c.DoError(ctx, kivik.MethodPut, dbName, nil)
	return err
}

func (c *client) DestroyDB(ctx context.Context, dbName string, _ map[string]interface{}) error {
	_, err := c.DoError(ctx, kivik.MethodDelete, dbName, nil)
	return err
}
