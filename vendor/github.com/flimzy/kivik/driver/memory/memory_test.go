package memory

import (
	"context"
	"sort"
	"testing"

	"github.com/flimzy/diff"
	"github.com/flimzy/kivik/driver"
)

var d = &memDriver{}

func setup(t *testing.T, setup func(driver.Client)) driver.Client {
	c, err := d.NewClient(context.Background(), "foo")
	if err != nil {
		t.Fatal(err)
	}
	if setup != nil {
		setup(c)
	}
	return c
}

func TestNewClient(t *testing.T) {
	_, err := d.NewClient(context.Background(), "foo")
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
}

func TestDBExists(t *testing.T) {
	type deTest struct {
		Name     string
		DBName   string
		Setup    func(driver.Client)
		Expected bool
		Error    string
	}
	tests := []deTest{
		{
			Name:     "NoDBs",
			DBName:   "foo",
			Expected: false,
		},
		{
			Name:   "ExistingDB",
			DBName: "foo",
			Setup: func(c driver.Client) {
				if err := c.CreateDB(context.Background(), "foo", nil); err != nil {
					panic(err)
				}
			},
			Expected: true,
		},
		{
			Name:   "OtherDB",
			DBName: "foo",
			Setup: func(c driver.Client) {
				if err := c.CreateDB(context.Background(), "bar", nil); err != nil {
					panic(err)
				}
			},
			Expected: false,
		},
	}
	for _, test := range tests {
		func(test deTest) {
			t.Run(test.Name, func(t *testing.T) {
				c := setup(t, test.Setup)
				result, err := c.DBExists(context.Background(), test.DBName, nil)
				var msg string
				if err != nil {
					msg = err.Error()
				}
				if test.Error != msg {
					t.Errorf("Unexpected error: %s", msg)
				}
				if result != test.Expected {
					t.Errorf("Expected: %t, Actual: %t", test.Expected, result)
				}
			})
		}(test)
	}
}

func TestCreateDB(t *testing.T) {
	type cdTest struct {
		Name   string
		DBName string
		Error  string
		Setup  func(driver.Client)
	}
	tests := []cdTest{
		{
			Name:   "FirstDB",
			DBName: "foo",
		},
		{
			Name:   "UsersDB",
			DBName: "_users",
		},
		{
			Name:   "SystemDB",
			DBName: "_foo",
			Error:  "invalid database name",
		},
		{
			Name:   "Duplicate",
			DBName: "foo",
			Setup: func(c driver.Client) {
				if e := c.CreateDB(context.Background(), "foo", nil); e != nil {
					panic(e)
				}
			},
			Error: "database exists",
		},
	}
	for _, test := range tests {
		func(test cdTest) {
			t.Run(test.Name, func(t *testing.T) {
				c := setup(t, test.Setup)
				var msg string
				if e := c.CreateDB(context.Background(), test.DBName, nil); e != nil {
					msg = e.Error()
				}
				if msg != test.Error {
					t.Errorf("Unexpected error: %s", msg)
				}
			})
		}(test)
	}
}

func TestAllDBs(t *testing.T) {
	type adTest struct {
		Name     string
		Setup    func(driver.Client)
		Expected []string
		Error    string
	}
	tests := []adTest{
		{
			Name:     "NoDBs",
			Expected: []string{},
		},
		{
			Name: "2DBs",
			Setup: func(c driver.Client) {
				if err := c.CreateDB(context.Background(), "foo", nil); err != nil {
					panic(err)
				}
				if err := c.CreateDB(context.Background(), "bar", nil); err != nil {
					panic(err)
				}
			},
			Expected: []string{"foo", "bar"},
		},
	}
	for _, test := range tests {
		func(test adTest) {
			t.Run(test.Name, func(t *testing.T) {
				c := setup(t, test.Setup)
				result, err := c.AllDBs(context.Background(), nil)
				var msg string
				if err != nil {
					msg = err.Error()
				}
				if msg != test.Error {
					t.Errorf("Unexpected error: %s", msg)
				}
				sort.Strings(test.Expected)
				sort.Strings(result)
				if d := diff.Interface(test.Expected, result); d != nil {
					t.Error(d)
				}
			})
		}(test)
	}
}

func TestDestroyDB(t *testing.T) {
	type ddTest struct {
		Name   string
		DBName string
		Setup  func(driver.Client)
		Error  string
	}
	tests := []ddTest{
		{
			Name:   "NoDBs",
			DBName: "foo",
			Error:  "database does not exist",
		},
		{
			Name:   "ExistingDB",
			DBName: "foo",
			Setup: func(c driver.Client) {
				if err := c.CreateDB(context.Background(), "foo", nil); err != nil {
					panic(err)
				}
			},
		},
		{
			Name:   "OtherDB",
			DBName: "foo",
			Setup: func(c driver.Client) {
				if err := c.CreateDB(context.Background(), "bar", nil); err != nil {
					panic(err)
				}
			},
			Error: "database does not exist",
		},
	}
	for _, test := range tests {
		func(test ddTest) {
			t.Run(test.Name, func(t *testing.T) {
				c := setup(t, test.Setup)
				var msg string
				if e := c.DestroyDB(context.Background(), test.DBName, nil); e != nil {
					msg = e.Error()
				}
				if msg != test.Error {
					t.Errorf("Unexpected error: %s", msg)
				}
			})
		}(test)
	}
}

func TestDB(t *testing.T) {
	type dbTest struct {
		Name   string
		DBName string
		Setup  func(driver.Client)
		Error  string
	}
	tests := []dbTest{
		{
			Name:   "NoDBs",
			DBName: "foo",
			Error:  "database does not exist",
		},
		{
			Name:   "ExistingDB",
			DBName: "foo",
			Setup: func(c driver.Client) {
				if err := c.CreateDB(context.Background(), "foo", nil); err != nil {
					panic(err)
				}
			},
		},
		{
			Name:   "OtherDB",
			DBName: "foo",
			Setup: func(c driver.Client) {
				if err := c.CreateDB(context.Background(), "bar", nil); err != nil {
					panic(err)
				}
			},
			Error: "database does not exist",
		},
	}
	for _, test := range tests {
		func(test dbTest) {
			t.Run(test.Name, func(t *testing.T) {
				c := setup(t, test.Setup)
				_, err := c.DB(context.Background(), test.DBName, nil)
				var msg string
				if err != nil {
					msg = err.Error()
				}
				if msg != test.Error {
					t.Errorf("Unexpected error: %s", msg)
				}
			})
		}(test)
	}
}
