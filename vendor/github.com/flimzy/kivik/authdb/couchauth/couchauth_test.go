package couchauth

import (
	"context"
	"testing"

	"github.com/flimzy/kivik"
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

func TestBadDSN(t *testing.T) {
	if _, err := New(context.Background(), "http://foo.com:port with spaces/"); err == nil {
		t.Errorf("Expected error for invalid URL.")
	}
	if _, err := New(context.Background(), "http://foo:bar@foo.com/"); err == nil {
		t.Error("Expected error for DSN with credentials.")
	}
}

func TestCouchAuth(t *testing.T) {
	client := kt.GetClient(t)
	db, e := client.DB(context.Background(), "_users")
	if e != nil {
		t.Fatalf("Failed to connect to db: %s", e)
	}
	name := kt.TestDBName(t)
	user := &tuser{
		ID:       kivik.UserPrefix + name,
		Name:     name,
		Type:     "user",
		Roles:    []string{"coolguy"},
		Password: "abc123",
	}
	rev, e := db.Put(context.Background(), user.ID, user)
	if e != nil {
		t.Fatalf("Failed to create user: %s", e)
	}
	defer db.Delete(context.Background(), user.ID, rev)
	auth, e := New(context.Background(), kt.NoAuthDSN(t))
	if e != nil {
		t.Fatalf("Failed to connect to remote server: %s", e)
	}
	t.Run("sync", func(t *testing.T) {
		t.Run("Validate", func(t *testing.T) {
			t.Parallel()
			t.Run("ValidUser", func(t *testing.T) {
				t.Parallel()
				uCtx, err := auth.Validate(context.Background(), user.Name, "abc123")
				if err != nil {
					t.Errorf("Validation failure for good password: %s", err)
				}
				if uCtx == nil {
					t.Errorf("User should have been validated")
				}
			})
			t.Run("WrongPassword", func(t *testing.T) {
				t.Parallel()
				uCtx, err := auth.Validate(context.Background(), user.Name, "foobar")
				if kivik.StatusCode(err) != kivik.StatusUnauthorized {
					t.Errorf("Expected Unauthorized for bad password, got %s", err)
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
	})

	// roles, err := auth.Roles(context.Background(), "test")
	// if err != nil {
	// 	t.Errorf("Failed to get roles for valid user: %s", err)
	// }
	// if !reflect.DeepEqual(roles, []string{"coolguy"}) {
	// 	t.Errorf("Got unexpected roles.")
	// }
	// _, err = auth.Roles(context.Background(), "nobody")
	// if errors.StatusCode(err) != kivik.StatusNotFound {
	// 	var msg string
	// 	if err != nil {
	// 		msg = fmt.Sprintf(" Got: %s", err)
	// 	}
	// 	t.Errorf("Expected Not Found fetching roles for bad username.%s", msg)
	// }
}
