package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/driver"
	"github.com/flimzy/kivik/errors"
)

var notYetImplemented = errors.Status(kivik.StatusNotImplemented, "kivik: not yet implemented in memory driver")

// database is an in-memory database representation.
type db struct {
	*client
	dbName   string
	db       *database
	security *kivik.Security
}

type indexDoc struct {
	ID    string        `json:"id"`
	Key   string        `json:"key"`
	Value indexDocValue `json:"value"`
}

type indexDocValue struct {
	Rev string `json:"rev"`
}

func (d *db) Query(ctx context.Context, ddoc, view string, opts map[string]interface{}) (driver.Rows, error) {
	// FIXME: Unimplemented
	return nil, notYetImplemented
}

func (d *db) Get(_ context.Context, docID string, opts map[string]interface{}) (json.RawMessage, error) {
	if !d.db.docExists(docID) {
		return nil, errors.Status(kivik.StatusNotFound, "missing")
	}
	if rev, ok := opts["rev"].(string); ok {
		if doc, found := d.db.getRevision(docID, rev); found {
			return doc.data, nil
		}
		return nil, errors.Status(kivik.StatusNotFound, "missing")
	}
	last, _ := d.db.latestRevision(docID)
	if last.Deleted {
		return nil, errors.Status(kivik.StatusNotFound, "missing")
	}
	return last.data, nil
}

func (d *db) CreateDoc(ctx context.Context, doc interface{}) (docID, rev string, err error) {
	couchDoc, err := toCouchDoc(doc)
	if err != nil {
		return "", "", err
	}
	if id, ok := couchDoc["_id"].(string); ok {
		docID = id
	} else {
		docID = randStr()
	}
	rev, err = d.Put(ctx, docID, doc)
	return docID, rev, err
}

func (d *db) Put(_ context.Context, docID string, doc interface{}) (rev string, err error) {
	isLocal := strings.HasPrefix(docID, "_local/")
	if !isLocal && docID[0] == '_' && !strings.HasPrefix(docID, "_design/") {
		return "", errors.Status(kivik.StatusBadRequest, "Only reserved document ids may start with underscore.")
	}
	couchDoc, err := toCouchDoc(doc)
	if err != nil {
		return "", err
	}
	couchDoc["_id"] = docID
	// TODO: Add support for storing attachments.
	delete(couchDoc, "_attachments")

	if last, ok := d.db.latestRevision(docID); ok {
		if !last.Deleted && !isLocal && couchDoc.Rev() != fmt.Sprintf("%d-%s", last.ID, last.Rev) {
			return "", errors.Status(kivik.StatusConflict, "document update conflict")
		}
		return d.db.addRevision(couchDoc), nil
	}

	if couchDoc.Rev() != "" {
		// Rev should not be set for a new document
		return "", errors.Status(kivik.StatusConflict, "document update conflict")
	}
	return d.db.addRevision(couchDoc), nil
}

var revRE = regexp.MustCompile("^[0-9]+-[a-f0-9]{32}$")

func validRev(rev string) bool {
	return revRE.MatchString(rev)
}

func (d *db) Delete(ctx context.Context, docID, rev string) (newRev string, err error) {
	if !strings.HasPrefix(docID, "_local/") && !validRev(rev) {
		return "", errors.Status(kivik.StatusBadRequest, "Invalid rev format")
	}
	if !d.db.docExists(docID) {
		return "", errors.Status(kivik.StatusNotFound, "missing")
	}
	return d.Put(ctx, docID, map[string]interface{}{
		"_id":      docID,
		"_rev":     rev,
		"_deleted": true,
	})
}

func (d *db) Stats(_ context.Context) (*driver.DBStats, error) {
	return &driver.DBStats{
		Name: d.dbName,
		// DocCount:     0,
		// DeletedCount: 0,
		// UpdateSeq:    "",
		// DiskSize:     0,
		// ActiveSize:   0,
		// ExternalSize: 0,
	}, nil
}

func (c *client) Compact(_ context.Context) error {
	// FIXME: Unimplemented
	return notYetImplemented
}

func (d *db) CompactView(_ context.Context, _ string) error {
	// FIXME: Unimplemented
	return notYetImplemented
}

func (d *db) ViewCleanup(_ context.Context) error {
	// FIXME: Unimplemented
	return notYetImplemented
}

func (d *db) Changes(ctx context.Context, opts map[string]interface{}) (driver.Changes, error) {
	// FIXME: Unimplemented
	return nil, notYetImplemented
}

func (d *db) PutAttachment(_ context.Context, _, _, _, _ string, _ io.Reader) (string, error) {
	// FIXME: Unimplemented
	return "", notYetImplemented
}

func (d *db) GetAttachment(ctx context.Context, docID, rev, filename string) (contentType string, md5sum driver.MD5sum, body io.ReadCloser, err error) {
	// FIXME: Unimplemented
	return "", driver.MD5sum{}, nil, notYetImplemented
}

func (d *db) DeleteAttachment(ctx context.Context, docID, rev, filename string) (newRev string, err error) {
	// FIXME: Unimplemented
	return "", notYetImplemented
}
