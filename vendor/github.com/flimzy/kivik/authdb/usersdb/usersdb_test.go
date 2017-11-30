package usersdb

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/authdb"
	_ "github.com/go-kivik/couchdb"
	"github.com/go-kivik/kiviktest/kt"
)

type tuser struct {
	ID       string   `json:"_id"`
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Roles    []string `json:"roles"`
	Password string   `json:"password"`
}

func TestCouchAuth(t *testing.T) {
	client := kt.GetClient(t)
	db, err := client.DB(context.Background(), "_users")
	if err != nil {
		t.Fatalf("Failed to connect to db: %s", err)
	}
	name := kt.TestDBName(t)
	user := &tuser{
		ID:       kivik.UserPrefix + name,
		Name:     name,
		Type:     "user",
		Roles:    []string{"coolguy"},
		Password: "abc123",
	}
	rev, err := db.Put(context.Background(), user.ID, user)
	if err != nil {
		t.Fatalf("Failed to create user: %s", err)
	}
	defer db.Delete(context.Background(), user.ID, rev)
	auth := New(db)
	t.Run("sync", func(t *testing.T) {
		t.Run("Validate", func(t *testing.T) {
			t.Parallel()
			t.Run("ValidUser", func(t *testing.T) {
				uCtx, err := auth.Validate(context.Background(), user.Name, "abc123")
				if err != nil {
					t.Errorf("Validation failure for good password: %s", err)
				}
				if uCtx == nil {
					t.Errorf("User should have been validated")
				}
			})
			t.Run("WrongPassword", func(t *testing.T) {
				uCtx, err := auth.Validate(context.Background(), user.Name, "foobar")
				if kivik.StatusCode(err) != kivik.StatusUnauthorized {
					t.Errorf("Expected Unauthorized password, got %s", err)
				}
				if uCtx != nil {
					t.Errorf("User should not have been validated with wrong password")
				}
			})
			t.Run("MissingUser", func(t *testing.T) {
				t.Parallel()
				uCtx, err := auth.Validate(context.Background(), "nobody", "foo")
				if kivik.StatusCode(err) != kivik.StatusUnauthorized {
					t.Errorf("Expected Unauthorized for bad username, got %s", err)
				}
				if uCtx != nil {
					t.Errorf("User should not have been validated with wrong username")
				}
			})
		})

		t.Run("Context", func(t *testing.T) {
			t.Parallel()
			t.Run("ValidUser", func(t *testing.T) {
				t.Parallel()
				uCtx, err := auth.UserCtx(context.Background(), user.Name)
				if err != nil {
					t.Errorf("Failed to get roles: %s", err)
				}
				uCtx.Salt = "" // It's random, so remove it
				if !reflect.DeepEqual(uCtx, &authdb.UserContext{Name: user.Name, Roles: []string{"coolguy"}}) {
					t.Errorf("Got unexpected output: %v", uCtx)
				}
			})
			t.Run("MissingUser", func(t *testing.T) {
				t.Parallel()
				_, err := auth.UserCtx(context.Background(), "nobody")
				if kivik.StatusCode(err) != kivik.StatusNotFound {
					var msg string
					if err != nil {
						msg = fmt.Sprintf(" Got: %s", err)
					}
					t.Errorf("Expected Not Found fetching roles: %s", msg)
				}
			})
		})
	})
}
