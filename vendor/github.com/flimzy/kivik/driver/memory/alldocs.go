package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/flimzy/kivik/driver"
)

func (d *db) AllDocs(ctx context.Context, opts map[string]interface{}) (driver.Rows, error) {
	rows := &alldocsResults{
		resultSet{
			docIDs: make([]string, 0),
			revs:   make([]*revision, 0),
		},
	}
	for docID := range d.db.docs {
		if doc, found := d.db.latestRevision(docID); found {
			rows.docIDs = append(rows.docIDs, docID)
			rows.revs = append(rows.revs, doc)
		}
	}
	rows.offset = 0
	rows.totalRows = int64(len(rows.docIDs))
	return rows, nil
}

type resultSet struct {
	docIDs            []string
	revs              []*revision
	offset, totalRows int64
	updateSeq         string
}

func (r *resultSet) Close() error {
	r.revs = nil
	return nil
}

func (r *resultSet) UpdateSeq() string { return r.updateSeq }
func (r *resultSet) TotalRows() int64  { return r.totalRows }
func (r *resultSet) Offset() int64     { return r.offset }

type alldocsResults struct {
	resultSet
}

var _ driver.Rows = &alldocsResults{}

func (r *alldocsResults) Next(row *driver.Row) error {
	if r.revs == nil || len(r.revs) == 0 {
		return io.EOF
	}
	row.ID, r.docIDs = r.docIDs[0], r.docIDs[1:]
	var next *revision
	next, r.revs = r.revs[0], r.revs[1:]
	row.Key = []byte(fmt.Sprintf(`"%s"`, row.ID))
	value := map[string]string{
		"rev": fmt.Sprintf("%d-%s", next.ID, next.Rev),
	}
	var err error
	row.Value, err = json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return nil
}
