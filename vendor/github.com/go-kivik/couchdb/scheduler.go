package couchdb

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/driver"
	"github.com/flimzy/kivik/errors"
	"github.com/go-kivik/couchdb/chttp"
)

type schedulerDoc struct {
	Database      string    `json:"database"`
	DocID         string    `json:"doc_id"`
	ReplicationID string    `json:"id"`
	Source        string    `json:"source"`
	Target        string    `json:"target"`
	StartTime     time.Time `json:"start_time"`
	LastUpdated   time.Time `json:"last_updated"`
	State         string    `json:"state"`
	Info          repInfo   `json:"info"`
}

type repInfo struct {
	Error            error
	DocsRead         int64 `json:"docs_read"`
	DocsWritten      int64 `json:"docs_written"`
	DocWriteFailures int64 `json:"doc_write_failures"`
	Pending          int64 `json:"changes_pending"`
}

func (i *repInfo) UnmarshalJSON(data []byte) error {
	switch {
	case string(data) == "null":
		return nil
	case data[0] == '{':
		type repInfoClone repInfo
		var x repInfoClone
		if err := json.Unmarshal(data, &x); err != nil {
			return err
		}
		*i = repInfo(x)
	default:
		var e replicationError
		if err := json.Unmarshal(data, &e); err != nil {
			return err
		}
		i.Error = &e
	}
	return nil
}

type schedulerReplication struct {
	docID         string
	database      string
	replicationID string
	source        string
	target        string
	startTime     time.Time
	lastUpdated   time.Time
	state         string
	info          repInfo

	*db
}

var _ driver.Replication = &schedulerReplication{}

func (c *client) schedulerSupported(ctx context.Context) (bool, error) {
	c.sdMU.Lock()
	defer c.sdMU.Unlock()
	if c.schedulerDetected != nil {
		return *c.schedulerDetected, nil
	}
	resp, err := c.DoReq(ctx, kivik.MethodHead, "_scheduler/jobs", nil)
	if err != nil {
		return false, err
	}
	var supported bool
	switch resp.StatusCode {
	case kivik.StatusBadRequest:
		// 1.6.x, 1.7.x
		supported = false
	case kivik.StatusNotFound:
		// 2.0.x
		supported = false
	case kivik.StatusOK, kivik.StatusUnauthorized:
		// 2.1.x +
		supported = true
	default:
		return false, errors.Statusf(kivik.StatusBadResponse, "Unknown response code %d", resp.StatusCode)
	}
	c.schedulerDetected = &supported
	return supported, nil
}

func (c *client) newSchedulerReplication(doc *schedulerDoc) *schedulerReplication {
	rep := &schedulerReplication{
		db: &db{
			client: c,
			dbName: doc.Database,
		},
	}
	rep.setFromDoc(doc)
	return rep
}

func (r *schedulerReplication) setFromDoc(doc *schedulerDoc) {
	if r.source == "" {
		r.docID = doc.DocID
		r.database = doc.Database
		r.replicationID = doc.ReplicationID
		r.source = doc.Source
		r.target = doc.Target
		r.startTime = doc.StartTime
	}
	r.lastUpdated = doc.LastUpdated
	r.state = doc.State
	r.info = doc.Info
}

func (c *client) fetchSchedulerReplication(ctx context.Context, docID string) (*schedulerReplication, error) {
	rep := &schedulerReplication{
		docID:    docID,
		database: "_replicator",
		db: &db{
			client: c,
			dbName: "_replicator",
		},
	}
	for rep.source == "" {
		if err := rep.update(ctx); err != nil {
			return rep, err
		}
		time.Sleep(100 * time.Millisecond)
	}
	return rep, nil
}

func (r *schedulerReplication) StartTime() time.Time { return r.startTime }
func (r *schedulerReplication) EndTime() time.Time {
	if r.state == "failed" || r.state == "completed" {
		return r.lastUpdated
	}
	return time.Time{}
}
func (r *schedulerReplication) Err() error            { return r.info.Error }
func (r *schedulerReplication) ReplicationID() string { return r.replicationID }
func (r *schedulerReplication) Source() string        { return r.source }
func (r *schedulerReplication) Target() string        { return r.target }
func (r *schedulerReplication) State() string         { return r.state }

func (r *schedulerReplication) Update(ctx context.Context, rep *driver.ReplicationInfo) error {
	if err := r.update(ctx); err != nil {
		return err
	}
	rep.DocWriteFailures = r.info.DocWriteFailures
	rep.DocsRead = r.info.DocsRead
	rep.DocsWritten = r.info.DocsWritten
	return nil
}

func (r *schedulerReplication) Delete(ctx context.Context) error {
	rev, err := r.Rev(ctx, r.docID)
	if err != nil {
		return err
	}
	_, err = r.db.Delete(ctx, r.docID, rev)
	return err
}

func (r *schedulerReplication) update(ctx context.Context) error {
	path := fmt.Sprintf("/_scheduler/docs/%s/%s", r.database, chttp.EncodeDocID(r.docID))
	var doc schedulerDoc
	if _, err := r.db.Client.DoJSON(ctx, kivik.MethodGet, path, nil, &doc); err != nil {
		if cerr, ok := err.(*chttp.HTTPError); ok {
			if cerr.Code == 500 && cerr.Reason == "function_clause" {
				// This is a race condition bug in CouchDB 2.1.x. So try again.
				// See https://github.com/apache/couchdb/issues/1000
				return r.update(ctx)
			}
		}
		return err
	}
	r.setFromDoc(&doc)
	return nil
}

func (c *client) getReplicationsFromScheduler(ctx context.Context, options map[string]interface{}) ([]driver.Replication, error) {
	params, err := optionsToParams(options)
	if err != nil {
		return nil, err
	}
	var result struct {
		Docs []schedulerDoc `json:"docs"`
	}
	path := "/_scheduler/docs"
	if params != nil {
		path = path + "?" + params.Encode()
	}
	if _, err = c.DoJSON(ctx, kivik.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}
	reps := make([]driver.Replication, 0, len(result.Docs))
	for _, row := range result.Docs {
		rep := c.newSchedulerReplication(&row)
		reps = append(reps, rep)
	}
	return reps, nil
}
