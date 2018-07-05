package mock

import (
	"context"
	"time"

	"github.com/go-kivik/kivik/driver"
)

// Replication mocks driver.Replication
type Replication struct {
	// ID identifies a specific Replication instance
	ID                string
	DeleteFunc        func(context.Context) error
	StartTimeFunc     func() time.Time
	EndTimeFunc       func() time.Time
	ErrFunc           func() error
	ReplicationIDFunc func() string
	SourceFunc        func() string
	TargetFunc        func() string
	StateFunc         func() string
	UpdateFunc        func(context.Context, *driver.ReplicationInfo) error
}

var _ driver.Replication = &Replication{}

// Delete calls r.DeleteFunc
func (r *Replication) Delete(ctx context.Context) error {
	return r.DeleteFunc(ctx)
}

// StartTime calls r.StartTimeFunc
func (r *Replication) StartTime() time.Time {
	return r.StartTimeFunc()
}

// EndTime calls r.EndTimeFunc
func (r *Replication) EndTime() time.Time {
	return r.EndTimeFunc()
}

// Err calls r.ErrFunc
func (r *Replication) Err() error {
	return r.ErrFunc()
}

// ReplicationID calls r.ReplicatoinIDFunc
func (r *Replication) ReplicationID() string {
	return r.ReplicationIDFunc()
}

// Source calls r.SourceFunc or returns a default value if SourceFunc is nil
func (r *Replication) Source() string {
	if r.SourceFunc == nil {
		return r.ID + "-source"
	}
	return r.SourceFunc()
}

// Target calls r.TargetFunc or returns a default if TargetFunc is nil
func (r *Replication) Target() string {
	if r.TargetFunc == nil {
		return r.ID + "-target"
	}
	return r.TargetFunc()
}

// State calls r.StateFunc
func (r *Replication) State() string {
	return r.StateFunc()
}

// Update calls r.UpdateFunc
func (r *Replication) Update(ctx context.Context, rep *driver.ReplicationInfo) error {
	return r.UpdateFunc(ctx, rep)
}
