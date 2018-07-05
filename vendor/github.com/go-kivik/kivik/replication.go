package kivik

import (
	"context"
	"sync"
	"time"

	"github.com/go-kivik/kivik/driver"
	"github.com/go-kivik/kivik/errors"
)

// ReplicationState represents a replication's state
type ReplicationState string

// The possible values for the _replication_state field in _replicator documents
// plus a blank value for unstarted replications.
const (
	ReplicationNotStarted ReplicationState = ""
	ReplicationStarted    ReplicationState = "triggered"
	ReplicationError      ReplicationState = "error"
	ReplicationComplete   ReplicationState = "completed"
)

// Replication represents a CouchDB replication process.
type Replication struct {
	Source string
	Target string

	infoMU    sync.RWMutex
	info      *driver.ReplicationInfo
	statusErr error
	irep      driver.Replication
}

// DocsWritten returns the number of documents written, if known.
func (r *Replication) DocsWritten() int64 {
	if r != nil && r.info != nil {
		r.infoMU.RLock()
		defer r.infoMU.RUnlock()
		return r.info.DocsWritten
	}
	return 0
}

// DocsRead returns the number of documents read, if known.
func (r *Replication) DocsRead() int64 {
	if r != nil && r.info != nil {
		r.infoMU.RLock()
		defer r.infoMU.RUnlock()
		return r.info.DocsRead
	}
	return 0
}

// DocWriteFailures returns the number of doc write failures, if known.
func (r *Replication) DocWriteFailures() int64 {
	if r != nil && r.info != nil {
		r.infoMU.RLock()
		defer r.infoMU.RUnlock()
		return r.info.DocWriteFailures
	}
	return 0
}

// Progress returns the current replication progress, if known.
func (r *Replication) Progress() float64 {
	if r != nil && r.info != nil {
		r.infoMU.RLock()
		defer r.infoMU.RUnlock()
		return r.info.Progress
	}
	return 0
}

func newReplication(rep driver.Replication) *Replication {
	return &Replication{
		Source: rep.Source(),
		Target: rep.Target(),
		irep:   rep,
	}
}

// ReplicationID returns the _replication_id field of the replicator document.
func (r *Replication) ReplicationID() string {
	return r.irep.ReplicationID()
}

// StartTime returns the replication start time, once the replication has been
// triggered.
func (r *Replication) StartTime() time.Time {
	return r.irep.StartTime()
}

// EndTime returns the replication end time, once the replication has terminated.
func (r *Replication) EndTime() time.Time {
	return r.irep.EndTime()
}

// State returns the current replication state
func (r *Replication) State() ReplicationState {
	return ReplicationState(r.irep.State())
}

// Err returns the error, if any, that caused the replication to abort.
func (r *Replication) Err() error {
	if r == nil {
		return nil
	}
	return r.irep.Err()
}

// IsActive returns true if the replication has not yet completed or
// errored.
func (r *Replication) IsActive() bool {
	if r == nil {
		return false
	}
	return r.State() != ReplicationError && r.State() != ReplicationComplete
}

// Delete deletes a replication. If it is currently running, it will be
// cancelled.
func (r *Replication) Delete(ctx context.Context) error {
	return r.irep.Delete(ctx)
}

// Update requests a replication state update from the server. If there is an
// error retrieving the update, it is returned and the replication state is
// unaltered.
func (r *Replication) Update(ctx context.Context) error {
	var info driver.ReplicationInfo
	r.statusErr = r.irep.Update(ctx, &info)
	if r.statusErr != nil {
		return r.statusErr
	}
	r.infoMU.Lock()
	r.info = &info
	r.infoMU.Unlock()
	return nil
}

// GetReplications returns a list of defined replications in the _replicator
// database. Options are in the same format as to AllDocs(), except that
// "conflicts" and "update_seq" are ignored.
func (c *Client) GetReplications(ctx context.Context, options ...Options) ([]*Replication, error) {
	if replicator, ok := c.driverClient.(driver.ClientReplicator); ok {
		opts, err := mergeOptions(options...)
		if err != nil {
			return nil, err
		}
		reps, err := replicator.GetReplications(ctx, opts)
		if err != nil {
			return nil, err
		}
		replications := make([]*Replication, len(reps))
		for i, rep := range reps {
			replications[i] = newReplication(rep)
		}
		return replications, nil
	}
	return nil, errors.Status(StatusNotImplemented, "kivik: driver does not support replication")
}

// Replicate initiates a replication from source to target.
func (c *Client) Replicate(ctx context.Context, targetDSN, sourceDSN string, options ...Options) (*Replication, error) {
	if replicator, ok := c.driverClient.(driver.ClientReplicator); ok {
		opts, err := mergeOptions(options...)
		if err != nil {
			return nil, err
		}
		rep, err := replicator.Replicate(ctx, targetDSN, sourceDSN, opts)
		if err != nil {
			return nil, err
		}
		return newReplication(rep), nil
	}
	return nil, errors.Status(StatusNotImplemented, "kivik: driver does not support replication")
}

// ReplicationInfo represents a snapshot of the status of a replication.
type ReplicationInfo struct {
	DocWriteFailures int64
	DocsRead         int64
	DocsWritten      int64
	Progress         float64
}
