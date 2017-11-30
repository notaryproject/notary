package pouchdb

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/flimzy/kivik/driver/pouchdb/bindings"
	"github.com/gopherjs/gopherjs/js"
	"github.com/gopherjs/jsbuiltin"
)

type replicationState struct {
	*js.Object
	startTime        time.Time `js:"start_time"`
	endTime          time.Time `js:"end_time"`
	DocsRead         int64     `js:"docs_read"`
	DocsWritten      int64     `js:"docs_written"`
	DocWriteFailures int64     `js:"doc_write_failures"`
	LastSeq          string    `js:"last_seq"`
}

func (rs *replicationState) StartTime() time.Time {
	value := rs.Get("start_time")
	if jsbuiltin.InstanceOf(value, js.Global.Get("Date")) {
		return rs.startTime
	}
	t, err := convertTime(value)
	if err != nil {
		panic("start time: " + err.Error())
	}
	return t
}

func (rs *replicationState) EndTime() time.Time {
	value := rs.Get("end_time")
	if jsbuiltin.InstanceOf(value, js.Global.Get("Date")) {
		return rs.endTime
	}
	t, err := convertTime(value)
	if err != nil {
		panic("end time: " + err.Error())
	}
	return t
}

func convertTime(value *js.Object) (time.Time, error) {
	if value == js.Undefined {
		return time.Time{}, nil
	}
	switch jsbuiltin.TypeOf(value) {
	case jsbuiltin.TypeString:
		return time.Parse(time.RFC3339, value.String())
	}
	return time.Time{}, fmt.Errorf("unsupported type")
}

type replicationHandler struct {
	event *string
	state *replicationState

	mu       sync.Mutex
	wg       sync.WaitGroup
	complete bool
	obj      *js.Object
}

func (r *replicationHandler) Cancel() {
	r.obj.Call("cancel")
}

// Status returns the last-read status. If the last-read status was already read,
// this blocks until the next event.  If the replication is complete, it will
// return io.EOF immediately.
func (r *replicationHandler) Status() (string, *replicationState, error) {
	if r.complete && r.event == nil {
		return "", nil, io.EOF
	}
	r.mu.Lock()
	if r.event == nil {
		r.mu.Unlock()
		// Wait for an event to be ready to read
		r.wg.Wait()
		r.mu.Lock()
	}
	event, state := r.event, r.state
	r.event = nil
	r.mu.Unlock()
	r.wg.Add(1)
	return *event, state, nil
}

func (r *replicationHandler) handleEvent(event string, info *js.Object) {
	if r.complete {
		panic(fmt.Sprintf("Unexpected replication event after complete. %v %v", event, info))
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.event = &event
	switch event {
	case bindings.ReplicationEventDenied, bindings.ReplicationEventError, bindings.ReplicationEventComplete:
		r.complete = true
	}
	if info != nil && info != js.Undefined {
		r.state = &replicationState{Object: info}
	}
	r.wg.Done()
}

func newReplicationHandler(rep *js.Object) *replicationHandler {
	r := &replicationHandler{obj: rep}
	for _, event := range []string{
		bindings.ReplicationEventChange,
		bindings.ReplicationEventComplete,
		bindings.ReplicationEventPaused,
		bindings.ReplicationEventActive,
		bindings.ReplicationEventDenied,
		bindings.ReplicationEventError,
	} {
		func(e string) {
			rep.Call("on", e, func(info *js.Object) {
				r.handleEvent(e, info)
			})
		}(event)
	}
	r.wg.Add(1)
	return r
}
