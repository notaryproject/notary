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
	"github.com/flimzy/kivik/driver/couchdb/chttp"
)

type db struct {
	*client
	dbName      string
	forceCommit bool
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
				return nil, fmt.Errorf("cannot convert type %T to []string", i)
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
func (d *db) Get(ctx context.Context, docID string, opts map[string]interface{}) (json.RawMessage, error) {
	params, err := optionsToParams(opts)
	if err != nil {
		return nil, err
	}
	resp, err := d.Client.DoReq(ctx, http.MethodGet, d.path(chttp.EncodeDocID(docID), params), &chttp.Options{Accept: "application/json; multipart/mixed"})
	if err != nil {
		return nil, err
	}
	if respErr := chttp.ResponseError(resp); respErr != nil {
		return nil, respErr
	}
	defer resp.Body.Close()
	doc := &bytes.Buffer{}
	doc.ReadFrom(resp.Body)
	return doc.Bytes(), nil
}

func (d *db) CreateDoc(ctx context.Context, doc interface{}) (docID, rev string, err error) {
	result := struct {
		ID  string `json:"id"`
		Rev string `json:"rev"`
	}{}
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	defer cancel()
	body, errFunc := chttp.EncodeBody(doc, cancel)
	opts := &chttp.Options{
		Body:        body,
		ForceCommit: d.forceCommit,
	}
	_, err = d.Client.DoJSON(ctx, kivik.MethodPost, d.dbName, opts, &result)
	if jsonErr := errFunc(); jsonErr != nil {
		return "", "", jsonErr
	}
	return result.ID, result.Rev, err
}

func (d *db) Put(ctx context.Context, docID string, doc interface{}) (rev string, err error) {
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	defer cancel()
	body, errFunc := chttp.EncodeBody(doc, cancel)
	opts := &chttp.Options{
		Body:        body,
		ForceCommit: d.forceCommit,
	}
	var result struct {
		ID  string `json:"id"`
		Rev string `json:"rev"`
	}
	_, err = d.Client.DoJSON(ctx, kivik.MethodPut, d.path(chttp.EncodeDocID(docID), nil), opts, &result)
	if jsonErr := errFunc(); jsonErr != nil {
		return "", jsonErr
	}
	if err != nil {
		return "", err
	}
	if result.ID != docID {
		// This should never happen; this is mostly for debugging and internal use
		return result.Rev, fmt.Errorf("created document ID (%s) does not match that requested (%s)", result.ID, docID)
	}
	return result.Rev, nil
}

func (d *db) Delete(ctx context.Context, docID, rev string) (string, error) {
	query := url.Values{}
	query.Add("rev", rev)
	opts := &chttp.Options{
		ForceCommit: d.forceCommit,
	}
	resp, err := d.Client.DoReq(ctx, kivik.MethodDelete, d.path(chttp.EncodeDocID(docID), query), opts)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
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
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	defer cancel()
	body, errFunc := chttp.EncodeBody(security, cancel)
	opts := &chttp.Options{
		Body: body,
	}
	res, err := d.Client.DoReq(ctx, kivik.MethodPut, d.path("/_security", nil), opts)
	if jsonErr := errFunc(); jsonErr != nil {
		if res != nil && res.Body != nil {
			res.Body.Close()
		}
		return jsonErr
	}
	if err != nil {
		return err
	}
	defer res.Body.Close()
	return chttp.ResponseError(res)
}

// Rev returns the most current rev of the requested document.
func (d *db) Rev(ctx context.Context, docID string) (rev string, err error) {
	res, err := d.Client.DoError(ctx, http.MethodHead, d.path(chttp.EncodeDocID(docID), nil), nil)
	if err != nil {
		return "", err
	}
	return strings.Trim(res.Header.Get("Etag"), `""`), nil
}

func (d *db) Copy(ctx context.Context, targetID, sourceID string, options map[string]interface{}) (targetRev string, err error) {
	params, err := optionsToParams(options)
	if err != nil {
		return "", err
	}
	opts := &chttp.Options{
		ForceCommit: d.forceCommit,
		Destination: targetID,
	}
	resp, err := d.Client.DoReq(ctx, kivik.MethodCopy, d.path(chttp.EncodeDocID(sourceID), params), opts)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	return chttp.GetRev(resp)
}
