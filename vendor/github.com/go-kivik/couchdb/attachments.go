package couchdb

import (
	"context"
	"net/http"

	"github.com/go-kivik/couchdb/chttp"
	"github.com/go-kivik/kivik"
	"github.com/go-kivik/kivik/driver"
	"github.com/go-kivik/kivik/errors"
)

func (d *db) PutAttachment(ctx context.Context, docID, rev string, att *driver.Attachment, options map[string]interface{}) (newRev string, err error) {
	if docID == "" {
		return "", missingArg("docID")
	}
	if att == nil {
		return "", missingArg("att")
	}
	if att.Filename == "" {
		return "", missingArg("att.Filename")
	}
	if att.ContentType == "" {
		return "", missingArg("att.ContentType")
	}
	if att.Content == nil {
		return "", missingArg("att.Content")
	}

	fullCommit, err := fullCommit(false, options)
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
	opts := &chttp.Options{
		Body:        att.Content,
		ContentType: att.ContentType,
		FullCommit:  fullCommit,
	}
	_, err = d.Client.DoJSON(ctx, kivik.MethodPut, d.path(chttp.EncodeDocID(docID)+"/"+att.Filename, query), opts, &response)
	if err != nil {
		return "", err
	}
	return response.Rev, nil
}

func (d *db) GetAttachmentMeta(ctx context.Context, docID, rev, filename string, options map[string]interface{}) (*driver.Attachment, error) {
	resp, err := d.fetchAttachment(ctx, kivik.MethodHead, docID, rev, filename, options)
	if err != nil {
		return nil, err
	}
	att, err := decodeAttachment(resp)
	return att, err
}

func (d *db) GetAttachment(ctx context.Context, docID, rev, filename string, options map[string]interface{}) (*driver.Attachment, error) {
	resp, err := d.fetchAttachment(ctx, kivik.MethodGet, docID, rev, filename, options)
	if err != nil {
		return nil, err
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

func decodeAttachment(resp *http.Response) (*driver.Attachment, error) {
	cType, err := getContentType(resp)
	if err != nil {
		return nil, err
	}
	digest, err := getDigest(resp)
	if err != nil {
		return nil, err
	}

	return &driver.Attachment{
		ContentType: cType,
		Digest:      digest,
		Size:        resp.ContentLength,
		Content:     resp.Body,
	}, nil
}

func getContentType(resp *http.Response) (string, error) {
	ctype := resp.Header.Get("Content-Type")
	if _, ok := resp.Header["Content-Type"]; !ok {
		return "", errors.Status(kivik.StatusBadResponse, "no Content-Type in response")
	}
	return ctype, nil
}

func getDigest(resp *http.Response) (string, error) {
	etag, ok := chttp.ETag(resp)
	if !ok {
		return "", errors.Status(kivik.StatusBadResponse, "ETag header not found")
	}
	return etag, nil
}

func (d *db) DeleteAttachment(ctx context.Context, docID, rev, filename string, options map[string]interface{}) (newRev string, err error) {
	if docID == "" {
		return "", missingArg("docID")
	}
	if rev == "" {
		return "", missingArg("rev")
	}
	if filename == "" {
		return "", missingArg("filename")
	}

	fullCommit, err := fullCommit(false, options)
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
