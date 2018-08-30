package couchdb

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-kivik/kivik"
	_ "github.com/go-kivik/couchdb" //
)

func makeDB(client *kivik.Client, name string) error {
	_, err := client.CreateDB(context.TODO(), name)
	if err != nil {
		exists, nerr := client.DBExists(context.TODO(), name)
		// we may not be allowed to create the DB but use it then
		if (nerr == nil || strings.Contains(nerr.Error(), "You are not a server admin")) && exists {
			return nil
		}
	}
	return err
}

// Table holds the configuration for setting up a CouchDB table
type Table struct {
	Name             string
	Indexes          map[string]interface{}
	JSONUnmarshaller func([]byte) (interface{}, error)
}

// SetupDB handles creating the database and creating all tables and indexes.
func SetupDB(client *kivik.Client, dbName string, tables []Table) error {
	var err error

	for _, table := range tables {
		_dbName := CreateDBName(dbName, table.Name)
		if err = makeDB(client, _dbName); err != nil {
			return fmt.Errorf("unable to create DB %s: %s", _dbName, err)
		}

		db, err := client.DB(context.TODO(), _dbName)
		if err != nil {
			return fmt.Errorf("unable to access DB %s: %s", _dbName, err)
		}

		for _, fieldnames := range table.Indexes {
			if err := db.CreateIndex(context.TODO(), "", "", map[string]interface{}{
				"fields": fieldnames,
			}); err != nil {
				return fmt.Errorf("Could not create indexes for fieldnames %s for DB %s: %s", fieldnames, _dbName, err)
			}
		}
	}

	return nil
}

// CreateAndGrantDBUser handles creating a couch user and granting it permissions to the provided db.
func CreateAndGrantDBUser(client *kivik.Client, dbName, username, password string) error {
	var err error

	exists, err := client.DBExists(context.TODO(), "_users")
	if err != nil {
		return fmt.Errorf("Could not check whether _users DB exists: %s", err)
	}
	if !exists {
		if _, err = client.CreateDB(context.TODO(), "_users"); err != nil {
			return fmt.Errorf("Could not create _users DB")
		}
	}

        usersDB, err := client.DB(context.TODO(), "_users")
        if err != nil {
                return fmt.Errorf("unable to access _users DB: %s", err)
        }

	id := kivik.UserPrefix + username

	user := map[string]interface{}{
		"_id":      id,
		"name":     username,
		"type":     "user",
		"password": password,
		"roles":    []string{},
	}
	_, rev, err := usersDB.GetMeta(context.TODO(), id, nil)
	if err == nil {
		user["_rev"] = rev
	}

	_, err = usersDB.Put(context.TODO(), kivik.UserPrefix+username, user)

	if err != nil {
		return fmt.Errorf("unable to add user %s to couchdb users table: %s", username, err)
	}

	return nil
}
