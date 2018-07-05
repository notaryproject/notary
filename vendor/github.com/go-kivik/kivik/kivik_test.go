package kivik

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/flimzy/diff"
	"github.com/flimzy/testy"
	"github.com/go-kivik/kivik/driver"
	"github.com/go-kivik/kivik/mock"
)

func TestNew(t *testing.T) {
	registryMU.Lock()
	defer registryMU.Unlock()
	tests := []struct {
		name       string
		driver     driver.Driver
		driverName string
		dsn        string
		expected   *Client
		status     int
		err        string
	}{
		{
			name:       "Unregistered driver",
			driverName: "unregistered",
			dsn:        "unf",
			status:     StatusBadRequest,
			err:        `kivik: unknown driver "unregistered" (forgotten import?)`,
		},
		{
			name: "connection error",
			driver: &mock.Driver{
				NewClientFunc: func(_ context.Context, _ string) (driver.Client, error) {
					return nil, errors.New("connection error")
				},
			},
			driverName: "foo",
			status:     StatusInternalServerError,
			err:        "connection error",
		},
		{
			name: "success",
			driver: &mock.Driver{
				NewClientFunc: func(_ context.Context, dsn string) (driver.Client, error) {
					if dsn != "oink" {
						return nil, fmt.Errorf("Unexpected DSN: %s", dsn)
					}
					return &mock.Client{ID: "foo"}, nil
				},
			},
			driverName: "bar",
			dsn:        "oink",
			expected: &Client{
				dsn:          "oink",
				driverName:   "bar",
				driverClient: &mock.Client{ID: "foo"},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer func() {
				drivers = make(map[string]driver.Driver)
			}()
			if test.driver != nil {
				Register(test.driverName, test.driver)
			}
			result, err := New(context.Background(), test.driverName, test.dsn)
			testy.StatusError(t, test.err, test.status, err)
			if d := diff.Interface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestClientGetters(t *testing.T) {
	driverName := "foo"
	dsn := "bar"
	c := &Client{
		driverName: driverName,
		dsn:        dsn,
	}

	t.Run("Driver", func(t *testing.T) {
		result := c.Driver()
		if result != driverName {
			t.Errorf("Unexpected result: %s", result)
		}
	})

	t.Run("DSN", func(t *testing.T) {
		result := c.DSN()
		if result != dsn {
			t.Errorf("Unexpected result: %s", result)
		}
	})
}

func TestVersion(t *testing.T) {
	tests := []struct {
		name     string
		client   *Client
		expected *Version
		status   int
		err      string
	}{
		{
			name: "db error",
			client: &Client{
				driverClient: &mock.Client{
					VersionFunc: func(_ context.Context) (*driver.Version, error) {
						return nil, errors.New("db error")
					},
				},
			},
			status: StatusInternalServerError,
			err:    "db error",
		},
		{
			name: "success",
			client: &Client{
				driverClient: &mock.Client{
					VersionFunc: func(_ context.Context) (*driver.Version, error) {
						return &driver.Version{Version: "foo"}, nil
					},
				},
			},
			expected: &Version{Version: "foo"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.client.Version(context.Background())
			testy.StatusError(t, test.err, test.status, err)
			if d := diff.Interface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestDB(t *testing.T) {
	type Test struct {
		name     string
		client   *Client
		dbName   string
		options  Options
		expected *DB
		status   int
		err      string
	}
	tests := []Test{
		{
			name: "db error",
			client: &Client{
				driverClient: &mock.Client{
					DBFunc: func(_ context.Context, _ string, _ map[string]interface{}) (driver.DB, error) {
						return nil, errors.New("db error")
					},
				},
			},
			status: StatusInternalServerError,
			err:    "db error",
		},
		func() Test {
			client := &Client{
				driverClient: &mock.Client{
					DBFunc: func(_ context.Context, dbName string, opts map[string]interface{}) (driver.DB, error) {
						expectedDBName := "foo"
						expectedOpts := map[string]interface{}{"foo": 123}
						if dbName != expectedDBName {
							return nil, fmt.Errorf("Unexpected dbname: %s", dbName)
						}
						if d := diff.Interface(expectedOpts, opts); d != nil {
							return nil, fmt.Errorf("Unexpected options:\n%s", d)
						}
						return &mock.DB{ID: "abc"}, nil
					},
				},
			}
			return Test{
				name:    "success",
				client:  client,
				dbName:  "foo",
				options: map[string]interface{}{"foo": 123},
				expected: &DB{
					client:   client,
					name:     "foo",
					driverDB: &mock.DB{ID: "abc"},
				},
			}
		}(),
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.client.DB(context.Background(), test.dbName, test.options)
			testy.StatusError(t, test.err, test.status, err)
			if d := diff.Interface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestAllDBs(t *testing.T) {
	tests := []struct {
		name     string
		client   *Client
		options  Options
		expected []string
		status   int
		err      string
	}{
		{
			name: "db error",
			client: &Client{
				driverClient: &mock.Client{
					AllDBsFunc: func(_ context.Context, _ map[string]interface{}) ([]string, error) {
						return nil, errors.New("db error")
					},
				},
			},
			status: StatusInternalServerError,
			err:    "db error",
		},
		{
			name: "success",
			client: &Client{
				driverClient: &mock.Client{
					AllDBsFunc: func(_ context.Context, options map[string]interface{}) ([]string, error) {
						expectedOptions := map[string]interface{}{"foo": 123}
						if d := diff.Interface(expectedOptions, options); d != nil {
							return nil, fmt.Errorf("Unexpected options:\n%s", d)
						}
						return []string{"a", "b", "c"}, nil
					},
				},
			},
			options:  map[string]interface{}{"foo": 123},
			expected: []string{"a", "b", "c"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.client.AllDBs(context.Background(), test.options)
			testy.StatusError(t, test.err, test.status, err)
			if d := diff.Interface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestDBExists(t *testing.T) {
	tests := []struct {
		name     string
		client   *Client
		dbName   string
		options  Options
		expected bool
		status   int
		err      string
	}{
		{
			name: "db error",
			client: &Client{
				driverClient: &mock.Client{
					DBExistsFunc: func(_ context.Context, _ string, _ map[string]interface{}) (bool, error) {
						return false, errors.New("db error")
					},
				},
			},
			status: StatusInternalServerError,
			err:    "db error",
		},
		{
			name: "success",
			client: &Client{
				driverClient: &mock.Client{
					DBExistsFunc: func(_ context.Context, dbName string, opts map[string]interface{}) (bool, error) {
						expectedDBName := "foo"
						expectedOpts := map[string]interface{}{"foo": 123}
						if dbName != expectedDBName {
							return false, fmt.Errorf("Unexpected db name: %s", dbName)
						}
						if d := diff.Interface(expectedOpts, opts); d != nil {
							return false, fmt.Errorf("Unexpected opts:\n%s", d)
						}
						return true, nil
					},
				},
			},
			dbName:   "foo",
			options:  map[string]interface{}{"foo": 123},
			expected: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.client.DBExists(context.Background(), test.dbName, test.options)
			testy.StatusError(t, test.err, test.status, err)
			if test.expected != result {
				t.Errorf("Unexpected result: %v", result)
			}
		})
	}
}

func TestCreateDB(t *testing.T) {
	tests := []struct {
		name     string
		client   *Client
		dbName   string
		opts     Options
		expected *DB
		status   int
		err      string
	}{
		{
			name: "db error",
			client: &Client{
				driverClient: &mock.Client{
					CreateDBFunc: func(_ context.Context, _ string, _ map[string]interface{}) error {
						return errors.New("db error")
					},
				},
			},
			status: StatusInternalServerError,
			err:    "db error",
		},
		{
			name: "success",
			client: &Client{
				driverClient: &mock.Client{
					CreateDBFunc: func(_ context.Context, dbName string, opts map[string]interface{}) error {
						expectedDBName := "foo"
						expectedOpts := map[string]interface{}{"foo": 123}
						if dbName != expectedDBName {
							return fmt.Errorf("Unexpected dbname: %s", dbName)
						}
						if d := diff.Interface(expectedOpts, opts); d != nil {
							return fmt.Errorf("Unexpected opts:\n%s", d)
						}
						return nil
					},
					DBFunc: func(_ context.Context, dbName string, _ map[string]interface{}) (driver.DB, error) {
						return &mock.DB{ID: "abc"}, nil
					},
				},
			},
			dbName: "foo",
			opts:   map[string]interface{}{"foo": 123},
			expected: &DB{
				name:     "foo",
				driverDB: &mock.DB{ID: "abc"},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db, err := test.client.CreateDB(context.Background(), test.dbName, test.opts)
			testy.StatusError(t, test.err, test.status, err)
			db.client = nil // Determinism
			if d := diff.Interface(test.expected, db); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestDestroyDB(t *testing.T) {
	tests := []struct {
		name   string
		client *Client
		dbName string
		opts   Options
		status int
		err    string
	}{
		{
			name: "db error",
			client: &Client{
				driverClient: &mock.Client{
					DestroyDBFunc: func(_ context.Context, _ string, _ map[string]interface{}) error {
						return errors.New("db error")
					},
				},
			},
			status: StatusInternalServerError,
			err:    "db error",
		},
		{
			name: "success",
			client: &Client{
				driverClient: &mock.Client{
					DestroyDBFunc: func(_ context.Context, dbName string, opts map[string]interface{}) error {
						expectedDBName := "foo"
						expectedOpts := map[string]interface{}{"foo": 123}
						if dbName != expectedDBName {
							return fmt.Errorf("Unexpected dbname: %s", dbName)
						}
						if d := diff.Interface(expectedOpts, opts); d != nil {
							return fmt.Errorf("Unexpected opts:\n%s", d)
						}
						return nil
					},
				},
			},
			dbName: "foo",
			opts:   map[string]interface{}{"foo": 123},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.client.DestroyDB(context.Background(), test.dbName, test.opts)
			testy.StatusError(t, test.err, test.status, err)
		})
	}
}

func TestAuthenticate(t *testing.T) {
	tests := []struct {
		name   string
		client *Client
		auth   interface{}
		status int
		err    string
	}{
		{
			name: "non-authenticator",
			client: &Client{
				driverClient: &mock.Client{},
			},
			status: StatusNotImplemented,
			err:    "kivik: driver does not support authentication",
		},
		{
			name: "auth error",
			client: &Client{
				driverClient: &mock.Authenticator{
					AuthenticateFunc: func(_ context.Context, _ interface{}) error {
						return errors.New("auth error")
					},
				},
			},
			status: StatusInternalServerError,
			err:    "auth error",
		},
		{
			name: "success",
			client: &Client{
				driverClient: &mock.Authenticator{
					AuthenticateFunc: func(_ context.Context, a interface{}) error {
						expected := int(3)
						if d := diff.Interface(expected, a); d != nil {
							return fmt.Errorf("Unexpected authenticator:\n%s", d)
						}
						return nil
					},
				},
			},
			auth: int(3),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.client.Authenticate(context.Background(), test.auth)
			testy.StatusError(t, test.err, test.status, err)
		})
	}
}
