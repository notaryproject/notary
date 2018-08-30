package kivik

import (
	"context"
	"errors"
	"testing"

	"github.com/flimzy/diff"
	"github.com/go-kivik/kivik/driver"
	"github.com/go-kivik/kivik/mock"
)

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
			client: &mock.Client{},
			status: StatusNotImplemented,
			err:    "kivik: driver does not support sessions",
		},
		{
			name: "driver returns error",
			client: &mock.Sessioner{
				SessionFunc: func(_ context.Context) (*driver.Session, error) {
					return nil, errors.New("session error")
				},
			},
			status: StatusInternalServerError,
			err:    "session error",
		},
		{
			name: "good response",
			client: &mock.Sessioner{
				SessionFunc: func(_ context.Context) (*driver.Session, error) {
					return &driver.Session{
						Name:  "curly",
						Roles: []string{"stooges"},
					}, nil
				},
			},
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
