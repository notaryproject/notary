package kivik

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/flimzy/diff"
	"github.com/flimzy/testy"
	"github.com/go-kivik/kivik/driver"
	"github.com/go-kivik/kivik/mock"
)

func TestReplicationDocsWritten(t *testing.T) {
	t.Run("No Info", func(t *testing.T) {
		r := &Replication{}
		result := r.DocsWritten()
		if result != 0 {
			t.Errorf("Unexpected doc count: %d", result)
		}
	})
	t.Run("With Info", func(t *testing.T) {
		r := &Replication{
			info: &driver.ReplicationInfo{
				DocsWritten: 123,
			},
		}
		result := r.DocsWritten()
		if result != 123 {
			t.Errorf("Unexpected doc count: %d", result)
		}
	})
	t.Run("Nil", func(t *testing.T) {
		var r *Replication
		result := r.DocsWritten()
		if result != 0 {
			t.Errorf("Unexpected doc count: %d", result)
		}
	})
}

func TestDocsRead(t *testing.T) {
	t.Run("No Info", func(t *testing.T) {
		r := &Replication{}
		result := r.DocsRead()
		if result != 0 {
			t.Errorf("Unexpected doc count: %d", result)
		}
	})
	t.Run("With Info", func(t *testing.T) {
		r := &Replication{
			info: &driver.ReplicationInfo{
				DocsRead: 123,
			},
		}
		result := r.DocsRead()
		if result != 123 {
			t.Errorf("Unexpected doc count: %d", result)
		}
	})
	t.Run("Nil", func(t *testing.T) {
		var r *Replication
		result := r.DocsRead()
		if result != 0 {
			t.Errorf("Unexpected doc count: %d", result)
		}
	})
}

func TestDocWriteFailures(t *testing.T) {
	t.Run("No Info", func(t *testing.T) {
		r := &Replication{}
		result := r.DocWriteFailures()
		if result != 0 {
			t.Errorf("Unexpected doc count: %d", result)
		}
	})
	t.Run("With Info", func(t *testing.T) {
		r := &Replication{
			info: &driver.ReplicationInfo{
				DocWriteFailures: 123,
			},
		}
		result := r.DocWriteFailures()
		if result != 123 {
			t.Errorf("Unexpected doc count: %d", result)
		}
	})
	t.Run("Nil", func(t *testing.T) {
		var r *Replication
		result := r.DocWriteFailures()
		if result != 0 {
			t.Errorf("Unexpected doc count: %d", result)
		}
	})
}

func TestProgress(t *testing.T) {
	t.Run("No Info", func(t *testing.T) {
		r := &Replication{}
		result := r.Progress()
		if result != 0 {
			t.Errorf("Unexpected doc count: %v", result)
		}
	})
	t.Run("With Info", func(t *testing.T) {
		r := &Replication{
			info: &driver.ReplicationInfo{
				Progress: 123,
			},
		}
		result := r.Progress()
		if result != 123 {
			t.Errorf("Unexpected doc count: %v", result)
		}
	})
	t.Run("Nil", func(t *testing.T) {
		var r *Replication
		result := r.Progress()
		if result != 0 {
			t.Errorf("Unexpected doc count: %v", result)
		}
	})
}

func TestNewReplication(t *testing.T) {
	source := "foo"
	target := "bar"
	rep := &mock.Replication{
		SourceFunc: func() string { return source },
		TargetFunc: func() string { return target },
	}
	expected := &Replication{
		Source: source,
		Target: target,
		irep:   rep,
	}
	result := newReplication(rep)
	if d := diff.Interface(expected, result); d != nil {
		t.Error(d)
	}
}

func TestReplicationGetters(t *testing.T) {
	repID := "repID"
	start := parseTime(t, "2018-01-01T00:00:00Z")
	end := parseTime(t, "2019-01-01T00:00:00Z")
	state := "confusion"
	r := &Replication{
		irep: &mock.Replication{
			ReplicationIDFunc: func() string { return repID },
			StartTimeFunc:     func() time.Time { return start },
			EndTimeFunc:       func() time.Time { return end },
			StateFunc:         func() string { return state },
		},
	}

	t.Run("ReplicationID", func(t *testing.T) {
		result := r.ReplicationID()
		if result != repID {
			t.Errorf("Unexpected result: %v", result)
		}
	})

	t.Run("StartTime", func(t *testing.T) {
		result := r.StartTime()
		if !result.Equal(start) {
			t.Errorf("Unexpected result: %v", result)
		}
	})

	t.Run("EndTime", func(t *testing.T) {
		result := r.EndTime()
		if !result.Equal(end) {
			t.Errorf("Unexpected result: %v", result)
		}
	})

	t.Run("State", func(t *testing.T) {
		result := r.State()
		if result != ReplicationState(state) {
			t.Errorf("Unexpected result: %v", result)
		}
	})
}

func TestReplicationErr(t *testing.T) {
	t.Run("No error", func(t *testing.T) {
		r := &Replication{
			irep: &mock.Replication{
				ErrFunc: func() error { return nil },
			},
		}
		if err := r.Err(); err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
	})
	t.Run("Error", func(t *testing.T) {
		r := &Replication{
			irep: &mock.Replication{
				ErrFunc: func() error {
					return errors.New("rep error")
				},
			},
		}
		if err := r.Err(); err == nil || err.Error() != "rep error" {
			t.Errorf("Unexpected error: %s", err)
		}
	})
	t.Run("Nil", func(t *testing.T) {
		var r *Replication
		if err := r.Err(); err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
	})
}

func TestReplicationIsActive(t *testing.T) {
	t.Run("Active", func(t *testing.T) {
		r := &Replication{
			irep: &mock.Replication{
				StateFunc: func() string {
					return "active"
				},
			},
		}
		if !r.IsActive() {
			t.Errorf("Expected active")
		}
	})
	t.Run("Complete", func(t *testing.T) {
		r := &Replication{
			irep: &mock.Replication{
				StateFunc: func() string {
					return string(ReplicationComplete)
				},
			},
		}
		if r.IsActive() {
			t.Errorf("Expected not active")
		}
	})
	t.Run("Nil", func(t *testing.T) {
		var r *Replication
		if r.IsActive() {
			t.Errorf("Expected not active")
		}
	})
}

func TestReplicationDelete(t *testing.T) {
	expected := "delete error"
	r := &Replication{
		irep: &mock.Replication{
			DeleteFunc: func(_ context.Context) error { return errors.New(expected) },
		},
	}
	err := r.Delete(context.Background())
	testy.Error(t, expected, err)
}

func TestReplicationUpdate(t *testing.T) {
	t.Run("update error", func(t *testing.T) {
		expected := "rep error"
		r := &Replication{
			irep: &mock.Replication{
				UpdateFunc: func(_ context.Context, _ *driver.ReplicationInfo) error {
					return errors.New(expected)
				},
			},
		}
		err := r.Update(context.Background())
		testy.Error(t, expected, err)
	})

	t.Run("success", func(t *testing.T) {
		expected := driver.ReplicationInfo{
			DocsRead: 123,
		}
		r := &Replication{
			irep: &mock.Replication{
				UpdateFunc: func(_ context.Context, i *driver.ReplicationInfo) error {
					*i = driver.ReplicationInfo{
						DocsRead: 123,
					}
					return nil
				},
			},
		}
		err := r.Update(context.Background())
		testy.Error(t, "", err)
		if d := diff.Interface(&expected, r.info); d != nil {
			t.Error(d)
		}
	})
}

func TestGetReplications(t *testing.T) {
	tests := []struct {
		name     string
		client   *Client
		options  Options
		expected []*Replication
		status   int
		err      string
	}{
		{
			name: "non-replicator",
			client: &Client{
				driverClient: &mock.Client{},
			},
			status: StatusNotImplemented,
			err:    "kivik: driver does not support replication",
		},
		{
			name: "db error",
			client: &Client{
				driverClient: &mock.ClientReplicator{
					GetReplicationsFunc: func(_ context.Context, _ map[string]interface{}) ([]driver.Replication, error) {
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
				driverClient: &mock.ClientReplicator{
					GetReplicationsFunc: func(_ context.Context, opts map[string]interface{}) ([]driver.Replication, error) {
						expectedOpts := map[string]interface{}{"foo": 123}
						if d := diff.Interface(expectedOpts, opts); d != nil {
							return nil, fmt.Errorf("Unexpected options:\n%v", d)
						}
						return []driver.Replication{
							&mock.Replication{ID: "1"},
							&mock.Replication{ID: "2"},
						}, nil
					},
				},
			},
			options: map[string]interface{}{"foo": 123},
			expected: []*Replication{
				{
					Source: "1-source",
					Target: "1-target",
					irep:   &mock.Replication{ID: "1"},
				},
				{
					Source: "2-source",
					Target: "2-target",
					irep:   &mock.Replication{ID: "2"},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.client.GetReplications(context.Background(), test.options)
			testy.StatusError(t, test.err, test.status, err)
			if d := diff.Interface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}

func TestReplicate(t *testing.T) {
	tests := []struct {
		name           string
		client         *Client
		target, source string
		options        Options
		expected       *Replication
		status         int
		err            string
	}{
		{
			name: "non-replicator",
			client: &Client{
				driverClient: &mock.Client{},
			},
			status: StatusNotImplemented,
			err:    "kivik: driver does not support replication",
		},
		{
			name: "db error",
			client: &Client{
				driverClient: &mock.ClientReplicator{
					ReplicateFunc: func(_ context.Context, _, _ string, _ map[string]interface{}) (driver.Replication, error) {
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
				driverClient: &mock.ClientReplicator{
					ReplicateFunc: func(_ context.Context, target, source string, opts map[string]interface{}) (driver.Replication, error) {
						expectedTarget := "foo"
						expectedSource := "bar"
						expectedOpts := map[string]interface{}{"foo": 123}
						if target != expectedTarget {
							return nil, fmt.Errorf("Unexpected target: %s", target)
						}
						if source != expectedSource {
							return nil, fmt.Errorf("Unexpected source: %s", source)
						}
						if d := diff.Interface(expectedOpts, opts); d != nil {
							return nil, fmt.Errorf("Unexpected options:\n%v", d)
						}
						return &mock.Replication{ID: "a"}, nil
					},
				},
			},
			target:  "foo",
			source:  "bar",
			options: map[string]interface{}{"foo": 123},
			expected: &Replication{
				Source: "a-source",
				Target: "a-target",
				irep:   &mock.Replication{ID: "a"},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.client.Replicate(context.Background(), test.target, test.source, test.options)
			testy.StatusError(t, test.err, test.status, err)
			if d := diff.Interface(test.expected, result); d != nil {
				t.Error(d)
			}
		})
	}
}
