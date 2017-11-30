package couchdb

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/driver"
	"github.com/flimzy/kivik/driver/couchdb/chttp"
)

func (d *db) PutAttachment(ctx context.Context, docID, rev, filename, contentType string, body io.Reader) (newRev string, err error) {
	opts := &chttp.Options{
		Body:        body,
		ContentType: contentType,
	}
	query := url.Values{}
	if rev != "" {
		query.Add("rev", rev)
	}
	var response struct {
		Rev string `json:"rev"`
	}
	_, err = d.Client.DoJSON(ctx, kivik.MethodPut, d.path(chttp.EncodeDocID(docID)+"/"+filename, query), opts, &response)
	if err != nil {
		return "", err
	}
	return response.Rev, nil
}

func (d *db) GetAttachmentMeta(ctx context.Context, docID, rev, filename string) (cType string, md5sum driver.MD5sum, err error) {
	resp, err := d.fetchAttachment(ctx, kivik.MethodHead, docID, rev, filename)
	if err != nil {
		return "", driver.MD5sum{}, err
	}
	cType, md5sum, body, err := d.decodeAttachment(resp)
	body.Close()
	return cType, md5sum, err
}

func (d *db) GetAttachment(ctx context.Context, docID, rev, filename string) (cType string, md5sum driver.MD5sum, body io.ReadCloser, err error) {
	resp, err := d.fetchAttachment(ctx, kivik.MethodGet, docID, rev, filename)
	if err != nil {
		return "", driver.MD5sum{}, nil, err
	}
	return d.decodeAttachment(resp)
}

func (d *db) fetchAttachment(ctx context.Context, method, docID, rev, filename string) (*http.Response, error) {
	query := url.Values{}
	if rev != "" {
		query.Add("rev", rev)
	}
	resp, err := d.Client.DoReq(ctx, method, d.path(chttp.EncodeDocID(docID)+"/"+filename, query), nil)
	if err != nil {
		return nil, err
	}
	return resp, chttp.ResponseError(resp)
}

func (d *db) decodeAttachment(resp *http.Response) (cType string, md5sum driver.MD5sum, body io.ReadCloser, err error) {
	var ok bool
	if cType, ok = getContentType(resp); !ok {
		return "", driver.MD5sum{}, nil, errors.New("no Content-Type in response")
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
	hash, err := base64.StdEncoding.DecodeString(resp.Header.Get("Content-MD5"))
	if err != nil {
		err = fmt.Errorf("failed to decode MD5 checksum: %s", err)
	}
	copy(md5sum[:], hash)
	return md5sum, err
}

func (d *db) DeleteAttachment(ctx context.Context, docID, rev, filename string) (newRev string, err error) {
	query := url.Values{}
	if rev != "" {
		query.Add("rev", rev)
	}
	var response struct {
		Rev string `json:"rev"`
	}
	_, err = d.Client.DoJSON(ctx, kivik.MethodDelete, d.path(chttp.EncodeDocID(docID)+"/"+filename, query), nil, &response)
	if err != nil {
		return "", err
	}
	return response.Rev, nil
}
