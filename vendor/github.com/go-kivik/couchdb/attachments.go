package couchdb

import (
	"context"
	"encoding/base64"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/driver"
	"github.com/flimzy/kivik/errors"
	"github.com/go-kivik/couchdb/chttp"
)

func (d *db) PutAttachment(ctx context.Context, docID, rev, filename, contentType string, body io.Reader) (newRev string, err error) {
	return d.PutAttachmentOpts(ctx, docID, rev, filename, contentType, body, nil)
}

func (d *db) PutAttachmentOpts(ctx context.Context, docID, rev, filename, contentType string, body io.Reader, options map[string]interface{}) (newRev string, err error) {
	if docID == "" {
		return "", missingArg("docID")
	}
	if filename == "" {
		return "", missingArg("filename")
	}
	if contentType == "" {
		return "", missingArg("contentType")
	}
	if body == nil {
		return "", errors.Status(kivik.StatusBadRequest, "kivik: body is nil")
	}

	fullCommit, err := fullCommit(d.fullCommit, options)
	if err != nil {
		return "", err
	}

	query, err := optionsToParams(options)
	if err != nil {
		return "", err
	}
	if rev != "" {
		query.Set("rev", rev)
	}
	var response struct {
		Rev string `json:"rev"`
	}
	var bodyCloser io.ReadCloser
	if rc, ok := body.(io.ReadCloser); ok {
		bodyCloser = rc
	} else {
		bodyCloser = ioutil.NopCloser(body)
	}
	opts := &chttp.Options{
		Body:        bodyCloser,
		ContentType: contentType,
		FullCommit:  fullCommit,
	}
	_, err = d.Client.DoJSON(ctx, kivik.MethodPut, d.path(chttp.EncodeDocID(docID)+"/"+filename, query), opts, &response)
	if err != nil {
		return "", err
	}
	return response.Rev, nil
}

func (d *db) GetAttachmentMeta(ctx context.Context, docID, rev, filename string, options map[string]interface{}) (cType string, md5sum driver.MD5sum, err error) {
	resp, err := d.fetchAttachment(ctx, kivik.MethodHead, docID, rev, filename, options)
	if err != nil {
		return "", driver.MD5sum{}, err
	}
	cType, md5sum, body, err := decodeAttachment(resp)
	_ = body.Close()
	return cType, md5sum, err
}

func (d *db) GetAttachment(ctx context.Context, docID, rev, filename string) (cType string, md5sum driver.MD5sum, content io.ReadCloser, err error) {
	return d.GetAttachmentOpts(ctx, docID, rev, filename, nil)
}

func (d *db) GetAttachmentOpts(ctx context.Context, docID, rev, filename string, options map[string]interface{}) (cType string, md5sum driver.MD5sum, content io.ReadCloser, err error) {
	resp, err := d.fetchAttachment(ctx, kivik.MethodGet, docID, rev, filename, options)
	if err != nil {
		return "", driver.MD5sum{}, nil, err
	}
	return decodeAttachment(resp)
}

func (d *db) fetchAttachment(ctx context.Context, method, docID, rev, filename string, options map[string]interface{}) (*http.Response, error) {
	if method == "" {
		return nil, errors.New("method required")
	}
	if docID == "" {
		return nil, missingArg("docID")
	}
	if filename == "" {
		return nil, missingArg("filename")
	}

	inm, err := ifNoneMatch(options)
	if err != nil {
		return nil, err
	}

	query, err := optionsToParams(options)
	if err != nil {
		return nil, err
	}
	if rev != "" {
		query.Add("rev", rev)
	}
	opts := &chttp.Options{
		IfNoneMatch: inm,
	}
	resp, err := d.Client.DoReq(ctx, method, d.path(chttp.EncodeDocID(docID)+"/"+filename, query), opts)
	if err != nil {
		return nil, err
	}
	return resp, chttp.ResponseError(resp)
}

func decodeAttachment(resp *http.Response) (cType string, md5sum driver.MD5sum, content io.ReadCloser, err error) {
	var ok bool
	if cType, ok = getContentType(resp); !ok {
		return "", driver.MD5sum{}, nil, errors.Status(kivik.StatusBadResponse, "no Content-Type in response")
	}

	md5sum, err = getMD5Checksum(resp)

	return cType, md5sum, resp.Body, err
}

func getContentType(resp *http.Response) (ctype string, ok bool) {
	ctype = resp.Header.Get("Content-Type")
	_, ok = resp.Header["Content-Type"]
	return ctype, ok
}

func getMD5Checksum(resp *http.Response) (md5sum driver.MD5sum, err error) {
	etag, ok := resp.Header["Etag"]
	if !ok {
		etag, ok = resp.Header["ETag"]
	}
	if !ok {
		return driver.MD5sum{}, errors.Status(kivik.StatusBadResponse, "ETag header not found")
	}
	hash, err := base64.StdEncoding.DecodeString(strings.Trim(etag[0], `"`))
	if err != nil {
		err = errors.Statusf(kivik.StatusBadResponse, "failed to decode MD5 checksum: %s", err)
	}
	copy(md5sum[:], hash)
	return md5sum, err
}

func (d *db) DeleteAttachment(ctx context.Context, docID, rev, filename string) (newRev string, err error) {
	return d.DeleteAttachmentOpts(ctx, docID, rev, filename, nil)
}

func (d *db) DeleteAttachmentOpts(ctx context.Context, docID, rev, filename string, options map[string]interface{}) (newRev string, err error) {
	if docID == "" {
		return "", missingArg("docID")
	}
	if rev == "" {
		return "", missingArg("rev")
	}
	if filename == "" {
		return "", missingArg("filename")
	}

	fullCommit, err := fullCommit(d.fullCommit, options)
	if err != nil {
		return "", err
	}

	query, err := optionsToParams(options)
	if err != nil {
		return "", err
	}
	query.Set("rev", rev)
	var response struct {
		Rev string `json:"rev"`
	}

	opts := &chttp.Options{
		FullCommit: fullCommit,
	}
	_, err = d.Client.DoJSON(ctx, kivik.MethodDelete, d.path(chttp.EncodeDocID(docID)+"/"+filename, query), opts, &response)
	if err != nil {
		return "", err
	}
	return response.Rev, nil
}
