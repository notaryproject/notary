package kivik

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"io/ioutil"

	"github.com/flimzy/kivik/errors"
)

// Attachments is a collection of one or more file attachments.
type Attachments map[string]*Attachment

// MD5sum is a 128-bit MD5 checksum.
type MD5sum [16]byte

// Attachment represents a file attachment on a CouchDB document.
type Attachment struct {
	io.ReadCloser   `json:"-"`
	Filename        string   `json:"-"`
	ContentType     string   `json:"content_type"`
	ContentEncoding string   `json:"encoding"`
	ContentLength   int64    `json:"length"`
	EncodedLength   int64    `json:"encoded_length"`
	RevPos          int64    `json:"revpos"`
	Digest          string   `json:"digest"`
	Stub            bool     `json:"stub"`
	MD5             [16]byte `json:"-"`
}

var _ io.ReadCloser = Attachment{}

func (a Attachment) Read(p []byte) (int, error) {
	if a.ReadCloser == nil {
		// TODO: Consider an alternative error code for this case
		return 0, errors.Status(StatusUnknownError, "kivik: attachment content not read")
	}
	return a.ReadCloser.Read(p)
}

// Close calls the underlying close method.
func (a Attachment) Close() error {
	if a.ReadCloser == nil {
		return nil
	}
	return a.ReadCloser.Close()
}

// bufCloser wraps a *bytes.Buffer to create an io.ReadCloser
type bufCloser struct {
	*bytes.Buffer
}

var _ io.ReadCloser = &bufCloser{}

func (b *bufCloser) Close() error { return nil }

// Bytes returns the attachment's body as a byte slice.
func (a *Attachment) Bytes() ([]byte, error) {
	if buf, ok := a.ReadCloser.(*bufCloser); ok {
		// Simple optimization
		return buf.Bytes(), nil
	}
	buf := &bytes.Buffer{}
	if _, err := io.Copy(buf, a); err != nil {
		return nil, err
	}
	a.ReadCloser = &bufCloser{buf}
	return buf.Bytes(), nil
}

// NewAttachment returns a new CouchDB attachment.
func NewAttachment(filename, contentType string, body io.ReadCloser) *Attachment {
	return &Attachment{
		ReadCloser:  body,
		Filename:    filename,
		ContentType: contentType,
	}
}

// validate returns an error if the attachment is invalid.
func (a *Attachment) validate() error {
	if a.Filename == "" {
		return missingArg("filename")
	}
	return nil
}

type jsonAttachment struct {
	ContentType string `json:"content_type"`
	Data        string `json:"data"`
}

func readEncoder(in io.ReadCloser) io.ReadCloser {
	r, w := io.Pipe()
	enc := base64.NewEncoder(base64.StdEncoding, w)
	go func() {
		_, err := io.Copy(enc, in)
		_ = enc.Close()
		_ = w.CloseWithError(err)
	}()
	return r
}

// MarshalJSON satisfis the json.Marshaler interface.
func (a *Attachment) MarshalJSON() ([]byte, error) {
	r := readEncoder(a)
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	att := &jsonAttachment{
		ContentType: a.ContentType,
		Data:        string(data),
	}
	return json.Marshal(att)
}

// UnmarshalJSON implements the json.Unmarshaler interface for an Attachment.
func (a *Attachment) UnmarshalJSON(data []byte) error {
	type clone Attachment
	type jsonAtt struct {
		clone
		Data []byte `json:"data"`
	}
	var att jsonAtt
	if err := json.Unmarshal(data, &att); err != nil {
		return err
	}
	*a = Attachment(att.clone)
	if att.Data != nil {
		a.ReadCloser = ioutil.NopCloser(bytes.NewReader(att.Data))
	}
	return nil
}

// UnmarshalJSON implements the json.Unmarshaler interface for a collection of
// Attachments.
func (a *Attachments) UnmarshalJSON(data []byte) error {
	atts := make(map[string]*Attachment)
	if err := json.Unmarshal(data, &atts); err != nil {
		return err
	}
	for filename, att := range atts {
		att.Filename = filename
	}
	*a = atts
	return nil
}
