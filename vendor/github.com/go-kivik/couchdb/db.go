package couchdb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/driver"
	"github.com/flimzy/kivik/errors"
	"github.com/go-kivik/couchdb/chttp"
)

type db struct {
	*client
	dbName     string
	fullCommit bool
}

func (d *db) path(path string, query url.Values) string {
	url, _ := url.Parse(d.dbName + "/" + strings.TrimPrefix(path, "/"))
	if query != nil {
		url.RawQuery = query.Encode()
	}
	return url.String()
}

func optionsToParams(opts ...map[string]interface{}) (url.Values, error) {
	params := url.Values{}
	for _, optsSet := range opts {
		for key, i := range optsSet {
			var values []string
			switch v := i.(type) {
			case string:
				values = []string{v}
			case []string:
				values = v
			case bool:
				values = []string{fmt.Sprintf("%t", v)}
			case int, uint, uint8, uint16, uint32, uint64, int8, int16, int32, int64:
				values = []string{fmt.Sprintf("%d", v)}
			default:
				return nil, errors.Statusf(kivik.StatusBadRequest, "kivik: invalid type %T for options", i)
			}
			for _, value := range values {
				params.Add(key, value)
			}
		}
	}
	return params, nil
}

// rowsQuery performs a query that returns a rows iterator.
func (d *db) rowsQuery(ctx context.Context, path string, opts map[string]interface{}) (driver.Rows, error) {
	options, err := optionsToParams(opts)
	if err != nil {
		return nil, err
	}
	resp, err := d.Client.DoReq(ctx, kivik.MethodGet, d.path(path, options), nil)
	if err != nil {
		return nil, err
	}
	if err = chttp.ResponseError(resp); err != nil {
		return nil, err
	}
	return newRows(resp.Body), nil
}

// AllDocs returns all of the documents in the database.
func (d *db) AllDocs(ctx context.Context, opts map[string]interface{}) (driver.Rows, error) {
	return d.rowsQuery(ctx, "_all_docs", opts)
}

// Query queries a view.
func (d *db) Query(ctx context.Context, ddoc, view string, opts map[string]interface{}) (driver.Rows, error) {
	return d.rowsQuery(ctx, fmt.Sprintf("_design/%s/_view/%s", chttp.EncodeDocID(ddoc), chttp.EncodeDocID(view)), opts)
}

// Get fetches the requested document.
func (d *db) Get(ctx context.Context, docID string, options map[string]interface{}) (json.RawMessage, error) {
	if docID == "" {
		return nil, missingArg("docID")
	}

	inm, err := ifNoneMatch(options)
	if err != nil {
		return nil, err
	}

	params, err := optionsToParams(options)
	if err != nil {
		return nil, err
	}
	opts := &chttp.Options{
		Accept:      "application/json; multipart/mixed",
		IfNoneMatch: inm,
	}
	resp, err := d.Client.DoReq(ctx, http.MethodGet, d.path(chttp.EncodeDocID(docID), params), opts)
	if err != nil {
		return nil, err
	}
	if respErr := chttp.ResponseError(resp); respErr != nil {
		return nil, respErr
	}
	defer func() { _ = resp.Body.Close() }()
	doc := &bytes.Buffer{}
	if _, err := doc.ReadFrom(resp.Body); err != nil {
		return nil, errors.WrapStatus(kivik.StatusUnknownError, err)
	}
	return doc.Bytes(), nil
}

func (d *db) CreateDoc(ctx context.Context, doc interface{}) (docID, rev string, err error) {
	return d.CreateDocOpts(ctx, doc, nil)
}

func (d *db) CreateDocOpts(ctx context.Context, doc interface{}, options map[string]interface{}) (docID, rev string, err error) {
	result := struct {
		ID  string `json:"id"`
		Rev string `json:"rev"`
	}{}

	fullCommit, err := fullCommit(d.fullCommit, options)
	if err != nil {
		return "", "", err
	}

	path := d.dbName
	if len(options) > 0 {
		params, e := optionsToParams(options)
		if e != nil {
			return "", "", e
		}
		path += "?" + params.Encode()
	}

	opts := &chttp.Options{
		Body:       chttp.EncodeBody(doc),
		FullCommit: fullCommit,
	}
	_, err = d.Client.DoJSON(ctx, kivik.MethodPost, path, opts, &result)
	return result.ID, result.Rev, err
}

func (d *db) Put(ctx context.Context, docID string, doc interface{}) (rev string, err error) {
	return d.PutOpts(ctx, docID, doc, nil)
}

func (d *db) PutOpts(ctx context.Context, docID string, doc interface{}, options map[string]interface{}) (rev string, err error) {
	if docID == "" {
		return "", missingArg("docID")
	}
	fullCommit, err := fullCommit(d.fullCommit, options)
	if err != nil {
		return "", err
	}
	opts := &chttp.Options{
		Body:       chttp.EncodeBody(doc),
		FullCommit: fullCommit,
	}
	var result struct {
		ID  string `json:"id"`
		Rev string `json:"rev"`
	}
	_, err = d.Client.DoJSON(ctx, kivik.MethodPut, d.path(chttp.EncodeDocID(docID), nil), opts, &result)
	if err != nil {
		return "", err
	}
	if result.ID != docID {
		// This should never happen; this is mostly for debugging and internal use
		return result.Rev, errors.Statusf(kivik.StatusBadResponse, "modified document ID (%s) does not match that requested (%s)", result.ID, docID)
	}
	return result.Rev, nil
}

func (d *db) Delete(ctx context.Context, docID, rev string) (string, error) {
	return d.DeleteOpts(ctx, docID, rev, nil)
}

func (d *db) DeleteOpts(ctx context.Context, docID, rev string, options map[string]interface{}) (string, error) {
	if docID == "" {
		return "", missingArg("docID")
	}
	if rev == "" {
		return "", missingArg("rev")
	}

	fullCommit, err := fullCommit(d.fullCommit, options)
	if err != nil {
		return "", err
	}

	query, err := optionsToParams(options)
	if err != nil {
		return "", err
	}
	query.Add("rev", rev)
	opts := &chttp.Options{
		FullCommit: fullCommit,
	}
	resp, err := d.Client.DoReq(ctx, kivik.MethodDelete, d.path(chttp.EncodeDocID(docID), query), opts)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	return chttp.GetRev(resp)
}

func (d *db) Flush(ctx context.Context) error {
	_, err := d.Client.DoError(ctx, kivik.MethodPost, d.path("/_ensure_full_commit", nil), nil)
	return err
}

func (d *db) Stats(ctx context.Context) (*driver.DBStats, error) {
	result := struct {
		driver.DBStats
		Sizes struct {
			File     int64 `json:"file"`
			External int64 `json:"external"`
			Active   int64 `json:"active"`
		} `json:"sizes"`
		UpdateSeq json.RawMessage `json:"update_seq"`
	}{}
	_, err := d.Client.DoJSON(ctx, kivik.MethodGet, d.dbName, nil, &result)
	stats := result.DBStats
	if result.Sizes.File > 0 {
		stats.DiskSize = result.Sizes.File
	}
	if result.Sizes.External > 0 {
		stats.ExternalSize = result.Sizes.External
	}
	if result.Sizes.Active > 0 {
		stats.ActiveSize = result.Sizes.Active
	}
	stats.UpdateSeq = string(bytes.Trim(result.UpdateSeq, `"`))
	return &stats, err
}

func (d *db) Compact(ctx context.Context) error {
	res, err := d.Client.DoReq(ctx, kivik.MethodPost, d.path("/_compact", nil), nil)
	if err != nil {
		return err
	}
	return chttp.ResponseError(res)
}

func (d *db) CompactView(ctx context.Context, ddocID string) error {
	if ddocID == "" {
		return missingArg("ddocID")
	}
	res, err := d.Client.DoReq(ctx, kivik.MethodPost, d.path("/_compact/"+ddocID, nil), nil)
	if err != nil {
		return err
	}
	return chttp.ResponseError(res)
}

func (d *db) ViewCleanup(ctx context.Context) error {
	res, err := d.Client.DoReq(ctx, kivik.MethodPost, d.path("/_view_cleanup", nil), nil)
	if err != nil {
		return err
	}
	return chttp.ResponseError(res)
}

func (d *db) Security(ctx context.Context) (*driver.Security, error) {
	var sec *driver.Security
	_, err := d.Client.DoJSON(ctx, kivik.MethodGet, d.path("/_security", nil), nil, &sec)
	return sec, err
}

func (d *db) SetSecurity(ctx context.Context, security *driver.Security) error {
	opts := &chttp.Options{
		Body: chttp.EncodeBody(security),
	}
	res, err := d.Client.DoReq(ctx, kivik.MethodPut, d.path("/_security", nil), opts)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()
	return chttp.ResponseError(res)
}

// Rev returns the most current rev of the requested document.
func (d *db) Rev(ctx context.Context, docID string) (rev string, err error) {
	if docID == "" {
		return "", missingArg("docID")
	}
	res, err := d.Client.DoError(ctx, http.MethodHead, d.path(chttp.EncodeDocID(docID), nil), nil)
	if err != nil {
		return "", err
	}
	return chttp.GetRev(res)
}

func (d *db) Copy(ctx context.Context, targetID, sourceID string, options map[string]interface{}) (targetRev string, err error) {
	if sourceID == "" {
		return "", errors.Status(kivik.StatusBadRequest, "kivik: sourceID required")
	}
	if targetID == "" {
		return "", errors.Status(kivik.StatusBadRequest, "kivik: targetID required")
	}
	fullCommit, err := fullCommit(d.fullCommit, options)
	if err != nil {
		return "", err
	}
	params, err := optionsToParams(options)
	if err != nil {
		return "", err
	}
	opts := &chttp.Options{
		FullCommit:  fullCommit,
		Destination: targetID,
	}
	resp, err := d.Client.DoReq(ctx, kivik.MethodCopy, d.path(chttp.EncodeDocID(sourceID), params), opts)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close() // nolint: errcheck
	return chttp.GetRev(resp)
}
