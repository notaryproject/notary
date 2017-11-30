package couchdb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/driver"
	"github.com/flimzy/kivik/driver/couchdb/chttp"
	"github.com/flimzy/kivik/errors"
)

type replicationError struct {
	status int
	reason string
}

func (re *replicationError) Error() string {
	return re.reason
}

func (re *replicationError) StatusCode() int {
	return re.status
}

func (re *replicationError) UnmarshalJSON(data []byte) error {
	reason := bytes.Trim(data, `"`)
	re.reason = string(reason)
	parts := bytes.SplitN(reason, []byte(":"), 2)
	switch string(parts[0]) {
	case "db_not_found":
		re.status = kivik.StatusNotFound
	case "timeout":
		re.status = kivik.StatusRequestTimeout
	default:
		re.status = kivik.StatusInternalServerError
	}
	return nil
}

type replicationStateTime time.Time

func (t *replicationStateTime) UnmarshalJSON(data []byte) error {
	input := string(bytes.Trim(data, `"`))
	if ts, err := time.Parse(time.RFC3339, input); err == nil {
		*t = replicationStateTime(ts)
		return nil
	}
	// Fallback for really old versions of CouchDB
	if seconds, err := strconv.ParseInt(input, 10, 64); err == nil {
		epochTime := replicationStateTime(time.Unix(seconds, 0).UTC())
		*t = epochTime
		return nil
	}
	return fmt.Errorf("kivik: '%s' does not appear to be a valid timestamp", string(data))
}

type replication struct {
	docID         string
	replicationID string
	source        string
	target        string
	startTime     time.Time
	endTime       time.Time
	state         string
	err           error

	// mu protects the above values
	mu sync.RWMutex

	*db
}

var _ driver.Replication = &replication{}

func (c *client) newReplication(docID string) *replication {
	return &replication{
		docID: docID,
		db: &db{
			client: c,
			dbName: "_replicator",
		},
	}
}

func (r *replication) readLock() func() {
	r.mu.RLock()
	return r.mu.RUnlock
}

func (r *replication) ReplicationID() string { defer r.readLock()(); return r.replicationID }
func (r *replication) Source() string        { defer r.readLock()(); return r.source }
func (r *replication) Target() string        { defer r.readLock()(); return r.target }
func (r *replication) StartTime() time.Time  { defer r.readLock()(); return r.startTime }
func (r *replication) EndTime() time.Time    { defer r.readLock()(); return r.endTime }
func (r *replication) State() string         { defer r.readLock()(); return r.state }
func (r *replication) Err() error            { defer r.readLock()(); return r.err }

func (r *replication) Update(ctx context.Context, state *driver.ReplicationInfo) error {
	if err := r.updateMain(ctx); err != nil {
		return err
	}
	if r.State() == "complete" {
		state.Progress = 100
		return nil
	}
	info, err := r.updateActiveTasks(ctx)
	if err != nil {
		if kivik.StatusCode(err) == kivik.StatusNotFound {
			// not listed in _active_tasks (because the replication is done, or
			// hasn't yet started), but this isn't an error
			return nil
		}
		return err
	}
	state.DocWriteFailures = info.DocWriteFailures
	state.DocsRead = info.DocsRead
	state.DocsWritten = info.DocsWritten
	// state.progress = info.Progress
	return nil
}

type activeTask struct {
	Type             string `json:"type"`
	ReplicationID    string `json:"replication_id"`
	DocsWritten      int64  `json:"docs_written"`
	DocsRead         int64  `json:"docs_read"`
	DocWriteFailures int64  `json:"doc_write_failures"`
}

func (r *replication) updateActiveTasks(ctx context.Context) (*activeTask, error) {
	resp, err := r.client.DoReq(ctx, kivik.MethodGet, "/_active_tasks", nil)
	if err != nil {
		return nil, err
	}
	if err = chttp.ResponseError(resp); err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var tasks []*activeTask
	if err = json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
		return nil, err
	}
	for _, task := range tasks {
		if task.Type != "replication" {
			continue
		}
		repIDparts := strings.SplitN(task.ReplicationID, "+", 2)
		if repIDparts[0] != r.replicationID {
			continue
		}
		return task, nil
	}
	return nil, errors.Status(kivik.StatusNotFound, "task not found")
}

// updateMain updates the "main" fields: those stored directly in r.
func (r *replication) updateMain(ctx context.Context) error {
	doc, err := r.getReplicatorDoc(ctx)
	if err != nil {
		return err
	}
	r.setFromReplicatorDoc(doc)
	return nil
}

func (r *replication) getReplicatorDoc(ctx context.Context) (*replicatorDoc, error) {
	body, err := r.db.Get(ctx, r.docID, nil)
	if err != nil {
		return nil, err
	}
	var doc replicatorDoc
	err = json.Unmarshal(body, &doc)
	return &doc, err
}

func (r *replication) setFromReplicatorDoc(doc *replicatorDoc) {
	r.mu.Lock()
	r.mu.Unlock()
	switch kivik.ReplicationState(doc.State) {
	case kivik.ReplicationStarted:
		r.startTime = time.Time(doc.StateTime)
	case kivik.ReplicationError, kivik.ReplicationComplete:
		r.endTime = time.Time(doc.StateTime)
	}
	r.state = doc.State
	if doc.Error != nil {
		r.err = doc.Error
	} else {
		r.err = nil
	}
	if r.source == "" {
		r.source = doc.Source
	}
	if r.target == "" {
		r.target = doc.Target
	}
	if r.replicationID == "" {
		r.replicationID = doc.ReplicationID
	}
}

func (r *replication) Delete(ctx context.Context) error {
	rev, err := r.Rev(ctx, r.docID)
	if err != nil {
		return err
	}
	_, err = r.db.Delete(ctx, r.docID, rev)
	return err
}

type replicatorDoc struct {
	DocID         string               `json:"_id"`
	ReplicationID string               `json:"_replication_id"`
	Source        string               `json:"source"`
	Target        string               `json:"target"`
	State         string               `json:"_replication_state"`
	StateTime     replicationStateTime `json:"_replication_state_time"`
	Error         *replicationError    `json:"_replication_state_reason,omitempty"`
}

func (c *client) GetReplications(ctx context.Context, options map[string]interface{}) ([]driver.Replication, error) {
	if options == nil {
		options = map[string]interface{}{}
	}
	delete(options, "conflicts")
	delete(options, "update_seq")
	options["include_docs"] = true
	params, err := optionsToParams(options)
	if err != nil {
		return nil, err
	}
	var result struct {
		Rows []struct {
			Doc replicatorDoc `json:"doc"`
		} `json:"rows"`
	}
	path := "/_replicator/_all_docs"
	if params != nil {
		path += "?" + params.Encode()
	}
	if _, err = c.DoJSON(ctx, kivik.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}
	reps := make([]driver.Replication, 0, len(result.Rows))
	for _, row := range result.Rows {
		if row.Doc.DocID == "_design/_replicator" {
			continue
		}
		rep := c.newReplication(row.Doc.DocID)
		rep.setFromReplicatorDoc(&row.Doc)
		reps = append(reps, rep)
	}
	return reps, nil
}

func (c *client) Replicate(ctx context.Context, targetDSN, sourceDSN string, options map[string]interface{}) (driver.Replication, error) {
	if options == nil {
		options = make(map[string]interface{})
	}
	// Allow overriding source and target with options, i.e. for OAuth1 options
	if _, ok := options["source"]; !ok {
		options["source"] = sourceDSN
	}
	if _, ok := options["target"]; !ok {
		options["target"] = targetDSN
	}
	body := &bytes.Buffer{}
	if err := json.NewEncoder(body).Encode(options); err != nil {
		return nil, err
	}
	var repStub struct {
		ID string `json:"id"`
	}
	_, err := c.Client.DoJSON(ctx, kivik.MethodPost, "/_replicator", &chttp.Options{Body: body}, &repStub)
	if err != nil {
		return nil, err
	}
	rep := c.newReplication(repStub.ID)
	rep.db = &db{client: c, dbName: "_replicator", forceCommit: true}
	// Do an update to get the initial state, but don't fail if there's an error
	// at this stage, because we successfully created the replication doc.
	_ = rep.updateMain(ctx)
	return rep, nil
}
