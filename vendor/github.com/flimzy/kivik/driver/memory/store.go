package memory

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/driver"
	"github.com/flimzy/kivik/errors"
)

type file struct {
	ContentType string
	Data        []byte
}

type document struct {
	revs []*revision
}

type revision struct {
	data        []byte
	ID          int64
	Rev         string
	Deleted     bool
	Attachments map[string]file
}

type database struct {
	mu        sync.RWMutex
	docs      map[string]*document
	deleted   bool
	security  *driver.Security
	updateSeq int64
}

var rnd *rand.Rand
var rndMU = &sync.Mutex{}

func init() {
	rnd = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func (d *database) getRevision(docID, rev string) (*revision, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	doc, ok := d.docs[docID]
	if !ok {
		return nil, false
	}
	for _, r := range doc.revs {
		if rev == fmt.Sprintf("%d-%s", r.ID, r.Rev) {
			return r, true
		}
	}
	return nil, false
}

func (d *database) docExists(docID string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	_, ok := d.docs[docID]
	return ok
}

func (d *database) latestRevision(docID string) (*revision, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	doc, ok := d.docs[docID]
	if ok {
		last := doc.revs[len(doc.revs)-1]
		return last, true
	}
	return nil, false
}

type couchDoc map[string]interface{}

func (d couchDoc) ID() string {
	id, _ := d["_id"].(string)
	return id
}

func (d couchDoc) Rev() string {
	rev, _ := d["_rev"].(string)
	return rev
}

func toCouchDoc(i interface{}) (couchDoc, error) {
	switch t := i.(type) {
	case couchDoc:
		return t, nil
	}
	asJSON, err := json.Marshal(i)
	if err != nil {
		return nil, errors.WrapStatus(kivik.StatusBadRequest, err)
	}
	var m couchDoc
	if e := json.Unmarshal(asJSON, &m); e != nil {
		return nil, errors.Status(kivik.StatusInternalServerError, "failed to decode encoded document; this is a bug!")
	}
	return m, nil
}

func (d *database) addRevision(doc couchDoc) string {
	d.mu.Lock()
	defer d.mu.Unlock()
	id, ok := doc["_id"].(string)
	if !ok {
		panic("_id missing or not a string")
	}
	isLocal := strings.HasPrefix(id, "_local/")
	if d.docs[id] == nil {
		d.docs[id] = &document{
			revs: make([]*revision, 0, 1),
		}
	}
	var revID int64
	var revStr string
	if isLocal {
		revID = 1
		revStr = "0"
	} else {
		l := len(d.docs[id].revs)
		if l == 0 {
			revID = 1
		} else {
			revID = d.docs[id].revs[l-1].ID + 1
		}
		revStr = randStr()
	}
	rev := fmt.Sprintf("%d-%s", revID, revStr)
	doc["_rev"] = rev
	data, err := json.Marshal(doc)
	if err != nil {
		panic(err)
	}
	deleted, _ := doc["_deleted"].(bool)
	newRev := &revision{
		data:    data,
		ID:      revID,
		Rev:     revStr,
		Deleted: deleted,
	}
	if isLocal {
		d.docs[id].revs = []*revision{newRev}
	} else {
		d.docs[id].revs = append(d.docs[id].revs, newRev)
	}
	return rev
}
