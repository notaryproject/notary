package couchdb

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/flimzy/diff"
	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/driver"
	"github.com/flimzy/testy"
)

func TestChanges(t *testing.T) {
	tests := []struct {
		name    string
		options map[string]interface{}
		db      *db
		status  int
		err     string
	}{
		{
			name:    "invalid options",
			options: map[string]interface{}{"foo": make(chan int)},
			status:  kivik.StatusBadRequest,
			err:     "kivik: invalid type chan int for options",
		},
		{
			name:   "network error",
			db:     newTestDB(nil, errors.New("net error")),
			status: kivik.StatusNetworkError,
			err:    "Get http://example.com/testdb/_changes?feed=continuous&heartbeat=6000&since=now: net error",
		},
		{
			name: "error response",
			db: newTestDB(&http.Response{
				StatusCode: kivik.StatusBadRequest,
				Body:       Body(""),
			}, nil),
			status: kivik.StatusBadRequest,
			err:    "Bad Request",
		},
		{
			name: "success 1.6.1",
			db: newTestDB(&http.Response{
				StatusCode: 200,
				Header: http.Header{
					"Transfer-Encoding": {"chunked"},
					"Server":            {"CouchDB/1.6.1 (Erlang OTP/17)"},
					"Date":              {"Fri, 27 Oct 2017 14:43:57 GMT"},
					"Content-Type":      {"text/plain; charset=utf-8"},
					"Cache-Control":     {"must-revalidate"},
				},
				Body: Body(`{"seq":3,"id":"43734cf3ce6d5a37050c050bb600006b","changes":[{"rev":"2-185ccf92154a9f24a4f4fd12233bf463"}],"deleted":true}
                    `),
			}, nil),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := test.db.Changes(context.Background(), test.options)
			testy.StatusError(t, test.err, test.status, err)
		})
	}
}

func TestChangesNext(t *testing.T) {
	tests := []struct {
		name     string
		changes  *changesRows
		status   int
		err      string
		expected *driver.Change
	}{
		{
			name: "invalid json",
			changes: &changesRows{
				body: Body("invalid json"),
			},
			status: kivik.StatusBadResponse,
			err:    "invalid character 'i' looking for beginning of value",
		},
		{
			name: "success",
			changes: &changesRows{
				body: Body(`{"seq":3,"id":"43734cf3ce6d5a37050c050bb600006b","changes":[{"rev":"2-185ccf92154a9f24a4f4fd12233bf463"}],"deleted":true}
                `),
			},
			expected: &driver.Change{
				ID:      "43734cf3ce6d5a37050c050bb600006b",
				Seq:     "3",
				Deleted: true,
				Changes: []string{"2-185ccf92154a9f24a4f4fd12233bf463"},
			},
		},
		{
			name: "end of input",
			changes: &changesRows{
				body: Body(``),
			},
			status: 500,
			err:    "EOF",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			row := new(driver.Change)
			err := test.changes.Next(row)
			testy.StatusError(t, test.err, test.status, err)
			if d := diff.Interface(test.expected, row); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestChangesClose(t *testing.T) {
	body := &closeTracker{ReadCloser: Body("foo")}
	feed := &changesRows{body: body}
	_ = feed.Close()
	if !body.closed {
		t.Errorf("Failed to close")
	}
}
