package memory

import (
	"context"
	"testing"

	"github.com/flimzy/diff"
	"github.com/flimzy/kivik/driver"
)

func TestGetSecurity(t *testing.T) {
	type secTest struct {
		Name     string
		DB       driver.DB
		Error    string
		Expected interface{}
	}
	tests := []secTest{
		{
			Name:  "DBNotFound",
			Error: "missing",
			DB: func() driver.DB {
				c := setup(t, nil)
				if err := c.CreateDB(context.Background(), "foo", nil); err != nil {
					t.Fatal(err)
				}
				db, err := c.DB(context.Background(), "foo", nil)
				if err != nil {
					t.Fatal(err)
				}
				if e := c.DestroyDB(context.Background(), "foo", nil); e != nil {
					t.Fatal(e)
				}
				return db
			}(),
		},
		{
			Name:     "EmptySecurity",
			Expected: &driver.Security{},
		},
		{
			Name: "AdminsAndMembers",
			DB: func() driver.DB {
				db := &db{
					db: &database{
						security: &driver.Security{
							Admins: driver.Members{
								Names: []string{"foo", "bar", "baz"},
								Roles: []string{"morons"},
							},
							Members: driver.Members{
								Names: []string{"bob"},
								Roles: []string{"boring"},
							},
						},
					},
				}
				return db
			}(),
			Expected: &driver.Security{
				Admins: driver.Members{
					Names: []string{"foo", "bar", "baz"},
					Roles: []string{"morons"},
				},
				Members: driver.Members{
					Names: []string{"bob"},
					Roles: []string{"boring"},
				},
			},
		},
	}
	for _, test := range tests {
		func(test secTest) {
			t.Run(test.Name, func(t *testing.T) {
				t.Parallel()
				db := test.DB
				if db == nil {
					db = setupDB(t, nil)
				}
				sec, err := db.Security(context.Background())
				var msg string
				if err != nil {
					msg = err.Error()
				}
				if msg != test.Error {
					t.Errorf("Unexpected error: %s", msg)
				}
				if err != nil {
					return
				}
				if d := diff.AsJSON(test.Expected, sec); d != nil {
					t.Error(d)
				}
			})
		}(test)
	}
}

func TestSetSecurity(t *testing.T) {
	type setTest struct {
		Name     string
		Security *driver.Security
		Error    string
		Expected *driver.Security
		DB       driver.DB
	}
	tests := []setTest{
		{
			Name:  "DBNotFound",
			Error: "missing",
			DB: func() driver.DB {
				c := setup(t, nil)
				if err := c.CreateDB(context.Background(), "foo", nil); err != nil {
					t.Fatal(err)
				}
				db, err := c.DB(context.Background(), "foo", nil)
				if err != nil {
					t.Fatal(err)
				}
				if e := c.DestroyDB(context.Background(), "foo", nil); e != nil {
					t.Fatal(e)
				}
				return db
			}(),
		},
		{
			Name: "Valid",
			Security: &driver.Security{
				Admins: driver.Members{
					Names: []string{"foo", "bar", "baz"},
					Roles: []string{"morons"},
				},
				Members: driver.Members{
					Names: []string{"bob"},
					Roles: []string{"boring"},
				},
			},
			Expected: &driver.Security{
				Admins: driver.Members{
					Names: []string{"foo", "bar", "baz"},
					Roles: []string{"morons"},
				},
				Members: driver.Members{
					Names: []string{"bob"},
					Roles: []string{"boring"},
				},
			},
		},
	}
	for _, test := range tests {
		func(test setTest) {
			t.Run(test.Name, func(t *testing.T) {
				t.Parallel()
				db := test.DB
				if db == nil {
					db = setupDB(t, nil)
				}
				err := db.SetSecurity(context.Background(), test.Security)
				var msg string
				if err != nil {
					msg = err.Error()
				}
				if msg != test.Error {
					t.Errorf("Unexpected error: %s", msg)
				}
				if err != nil {
					return
				}
				sec, err := db.Security(context.Background())
				if err != nil {
					t.Fatal(err)
				}
				if d := diff.AsJSON(test.Expected, sec); d != nil {
					t.Error(d)
				}
			})
		}(test)
	}
}
