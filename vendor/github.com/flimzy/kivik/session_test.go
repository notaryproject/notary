package kivik

import (
	"context"
	"errors"
	"testing"

	"github.com/flimzy/diff"
	"github.com/flimzy/kivik/driver"
)

type nonSessioner struct {
	driver.Client
}

type sessioner struct {
	driver.Client
	session *driver.Session
	err     error
}

func (s *sessioner) Session(_ context.Context) (*driver.Session, error) {
	return s.session, s.err
}

func TestSession(t *testing.T) {
	tests := []struct {
		name     string
		client   driver.Client
		expected interface{}
		status   int
		err      string
	}{
		{
			name:   "driver doesn't implement Sessioner",
			client: &nonSessioner{},
			status: StatusNotImplemented,
			err:    "kivik: driver does not support sessions",
		},
		{
			name:   "driver returns error",
			client: &sessioner{err: errors.New("session error")},
			status: StatusInternalServerError,
			err:    "session error",
		},
		{
			name: "good response",
			client: &sessioner{session: &driver.Session{
				Name:  "curly",
				Roles: []string{"stooges"},
			}},
			expected: &Session{
				Name:  "curly",
				Roles: []string{"stooges"},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := &Client{driverClient: test.client}
			session, err := client.Session(context.Background())
			var errMsg string
			if err != nil {
				errMsg = err.Error()
			}
			if errMsg != test.err {
				t.Errorf("Unexpected error: %s", errMsg)
			}
			if err != nil {
				return
			}
			if d := diff.Interface(test.expected, session); d != nil {
				t.Error(d)
			}
		})
	}
}
